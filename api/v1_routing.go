package api

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/internal"
	"github.com/MOPDev/mop-backend-api/internal/logger"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func derefUint(number *uint) uint {
	if number == nil {
		return 0
	}
	return *number
}

func fixRouteOrder(tx *gorm.DB, groupId uint, insertedVisitID uint) error {
	if tx == nil {
		tx = initializers.DB // ponytail: fallback for callers with no active transaction
	}

	var visits []models.Visit
	if err := tx.Where("group_id = ?", groupId).
		Order("stop_nr ASC").
		Find(&visits).Error; err != nil {
		return err
	}

	// pull the inserted visit out, re-insert before the last one
	ordered := make([]models.Visit, 0, len(visits))
	var inserted *models.Visit
	for i := range visits {
		if visits[i].ID == insertedVisitID {
			inserted = &visits[i]
			continue
		}
		ordered = append(ordered, visits[i])
	}
	if inserted == nil {
		return fmt.Errorf("visit %d not in group %d", insertedVisitID, groupId)
	}

	if len(ordered) == 0 {
		ordered = append(ordered, *inserted)
	} else {
		last := len(ordered) - 1
		ordered = append(ordered[:last], append([]models.Visit{*inserted}, ordered[last:]...)...)
	}

	for i := range ordered {
		stop := uint(i)
		if err := tx.Model(&models.Visit{}).
			Where("id = ?", ordered[i].ID).
			Update("stop_nr", stop).Error; err != nil {
			return err
		}
	}
	return nil
}

// ReorderVisit swaps stop_nr with the neighbour above or below inside the group.
// body: {"visitId": 12, "direction": "up"|"down"}
func ReorderVisit(c *gin.Context) {
	user, ok := getVerifyUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	groupId, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group ID"})
		return
	}

	var input struct {
		VisitId   uint   `json:"visitId" binding:"required"`
		Direction string `json:"direction" binding:"required,oneof=up down"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "visitId and direction (up|down) are required"})
		return
	}

	err = initializers.DB.Transaction(func(tx *gorm.DB) error {
		// lock the rows of the group so two users cannot swap at the same time
		var visit models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND id = ?", groupId, input.VisitId).
			First(&visit).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("visit not found in this group")
			}
			return err
		}

		if visit.Stopnr == nil {
			return errors.New("visit has no stop_nr")
		}

		// find the neighbour: the nearest stop_nr above or below
		var neighbour models.Visit
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND id != ?", groupId, visit.ID)

		if input.Direction == "up" {
			q = q.Where("stop_nr < ?", *visit.Stopnr).Order("stop_nr DESC")
		} else {
			q = q.Where("stop_nr > ?", *visit.Stopnr).Order("stop_nr ASC")
		}

		if err := q.First(&neighbour).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("visit is already at the end of the route")
			}
			return err
		}

		if derefUint(visit.SegmentIndex) != derefUint(neighbour.SegmentIndex) {
			return errors.New("cannot reorder across a segment boundary")
		}

		visitStop, neighbourStop := *visit.Stopnr, *neighbour.Stopnr

		// log both changes before the update
		if err := internal.UpdateVisitValue(tx, visit.ID, fmt.Sprintf("%d", neighbourStop), user.ID, "stop_nr"); err != nil {
			return err
		}
		if err := internal.UpdateVisitValue(tx, neighbour.ID, fmt.Sprintf("%d", visitStop), user.ID, "stop_nr"); err != nil {
			return err
		}

		// park visit out of the way so a unique (group_id, stop_nr) index
		// does not trip mid-swap
		if err := tx.Model(&models.Visit{}).Where("id = ?", visit.ID).
			Update("stop_nr", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Visit{}).Where("id = ?", neighbour.ID).
			Update("stop_nr", visitStop).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Visit{}).Where("id = ?", visit.ID).
			Update("stop_nr", neighbourStop).Error; err != nil {
			return err
		}

		// verify the invariant on the whole group, not just the shifted tail.
		// a return here rolls the transaction back.
		var all []models.Visit
		if err := tx.Where("group_id = ?", groupId).Find(&all).Error; err != nil {
			return err
		}
		return assertSegmentsValid(all)
	})

	if err != nil {
		logger.Errorf("ReorderVisit: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Route order updated successfully"})
}

// SplitSegment starts a new segment at this visit. The visit and every visit
// after it (in stop_nr order) get segment_index + 1.
// body: {"visitId": 12}
func SplitSegment(c *gin.Context) {
	user, ok := getVerifyUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	groupId, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group ID"})
		return
	}

	var input struct {
		VisitId uint `json:"visitId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "visitId is required"})
		return
	}

	err = initializers.DB.Transaction(func(tx *gorm.DB) error {
		var visit models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND id = ?", groupId, input.VisitId).
			First(&visit).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("visit not found in this group")
			}
			return err
		}

		if visit.Stopnr == nil {
			return errors.New("visit has no stop_nr")
		}

		// the first stop cannot start a new segment, there is nothing before it
		var before int64
		if err := tx.Model(&models.Visit{}).
			Where("group_id = ? AND stop_nr < ?", groupId, *visit.Stopnr).
			Count(&before).Error; err != nil {
			return err
		}
		if before == 0 {
			return errors.New("cannot split at the first stop of the route")
		}

		// all visits from this stop and onwards
		var affected []models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND stop_nr >= ?", groupId, *visit.Stopnr).
			Order("stop_nr ASC").Find(&affected).Error; err != nil {
			return err
		}

		for _, v := range affected {
			newIndex := uint(1)
			if v.SegmentIndex != nil {
				newIndex = *v.SegmentIndex + 1
			}
			if err := internal.UpdateVisitValue(tx, v.ID, fmt.Sprintf("%d", newIndex), user.ID, "segment_index"); err != nil {
				return err
			}
		}

		// MAX keeps segment_index at 0 or higher
		if err := tx.Model(&models.Visit{}).
			Where("group_id = ? AND stop_nr >= ?", groupId, *visit.Stopnr).
			Update("segment_index", gorm.Expr("COALESCE(segment_index, 0) + 1")).Error; err != nil {
			return err
		} // IF DATABASE CHANGE TO SOMETHING ELSE THEN SQLITE THEN USE GREATEST() INSTEAD OF MAX

		// verify the invariant on the whole group, not just the shifted tail.
		// a return here rolls the transaction back.
		var all []models.Visit
		if err := tx.Where("group_id = ?", groupId).Find(&all).Error; err != nil {
			return err
		}
		return assertSegmentsValid(all)
	})

	if err != nil {
		logger.Errorf("SplitSegment: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Segment split successfully"})
}

// JoinSegment merges the segment of this visit into the segment of the stop
// before it. The visit and every visit after it get segment_index - 1.
// body: {"visitId": 12}
func JoinSegment(c *gin.Context) {
	user, ok := getVerifyUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	groupId, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Group ID"})
		return
	}

	var input struct {
		VisitId uint `json:"visitId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "visitId is required"})
		return
	}

	err = initializers.DB.Transaction(func(tx *gorm.DB) error {
		var visit models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND id = ?", groupId, input.VisitId).
			First(&visit).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("visit not found in this group")
			}
			return err
		}

		if visit.Stopnr == nil || visit.SegmentIndex == nil {
			return errors.New("visit has no stop_nr or segment_index")
		}

		// the stop before this one
		var previous models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND stop_nr < ?", groupId, *visit.Stopnr).
			Order("stop_nr DESC").First(&previous).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("cannot join: this is the first stop of the route")
			}
			return err
		}

		// a join is only legal at the first stop of a segment. Anywhere else it
		// would cut the segment in two and break contiguity.
		if previous.SegmentIndex != nil && *previous.SegmentIndex == *visit.SegmentIndex {
			return errors.New("can only join at the first stop of a segment")
		}

		var affected []models.Visit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("group_id = ? AND stop_nr >= ?", groupId, *visit.Stopnr).
			Order("stop_nr ASC").Find(&affected).Error; err != nil {
			return err
		}

		for _, v := range affected {
			// guard above proves *v.SegmentIndex >= 1 for every affected row
			newIndex := *v.SegmentIndex - 1
			if err := internal.UpdateVisitValue(tx, v.ID, fmt.Sprintf("%d", newIndex), user.ID, "segment_index"); err != nil {
				return err
			}
		}

		// MAX keeps segment_index at 0 or higher
		if err := tx.Model(&models.Visit{}).
			Where("group_id = ? AND stop_nr >= ?", groupId, *visit.Stopnr).
			Update("segment_index", gorm.Expr("segment_index - 1")).Error; err != nil {
			return err
		} // IF DATABASE CHANGE TO SOMETHING ELSE THEN SQLITE THEN USE GREATEST() INSTEAD OF MAX

		// verify the invariant on the whole group, not just the shifted tail.
		// a return here rolls the transaction back.
		var all []models.Visit
		if err := tx.Where("group_id = ?", groupId).Find(&all).Error; err != nil {
			return err
		}
		return assertSegmentsValid(all)
	})

	if err != nil {
		logger.Errorf("JoinSegment: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Segment joined successfully"})
}

// ponytail: asserts contiguity + density, the two invariants the optimizer needs
func assertSegmentsValid(visits []models.Visit) error {
	sorted := append([]models.Visit(nil), visits...)
	sort.Slice(sorted, func(i, j int) bool {
		return derefUint(sorted[i].Stopnr) < derefUint(sorted[j].Stopnr)
	})
	want := uint(0)
	for i, v := range sorted {
		got := derefUint(v.SegmentIndex)
		if i > 0 && got != derefUint(sorted[i-1].SegmentIndex) {
			want++
		}
		if got != want {
			return fmt.Errorf("assert failed, visit %d: segment_index %d, want %d", v.ID, got, want)
		}
	}
	return nil
}

/*
	apiv1.POST("/visits/group/:groupId/reorder",api.ReorderVisit) //
	apiv1.POST("/visits/group/:groupId/split",  api.SplitSegment)
	apiv1.POST("/visits/group/:groupId/join", api.JoinSegment)
*/
