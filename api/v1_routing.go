package api

import (
	"fmt"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func ReorderVisit(c *gin.Context) { // body: {visitId, direction: "up"|"down"}, swaps stop_nr with neighbour, atomic

}

func SplitSegment(c *gin.Context) { // body: {visitId}, bumps segment_index for visit + all after it in stop_nr order

}

func JoinSegment(c *gin.Context) { // body: {visitId}, merges visit's segment into previous stop's segment_index

}

/*
	apiv1.POST("/visits/group/:groupId/reorder",api.ReorderVisit) //
	apiv1.POST("/visits/group/:groupId/split",  api.SplitSegment)
	apiv1.POST("/visits/group/:groupId/join", api.JoinSegment)
*/
