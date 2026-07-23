package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/internal"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/gin-gonic/gin"
)

func VisitPDF(c *gin.Context) {

	visitID, err := strconv.ParseInt(c.Query("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "The id could not be parsed",
			"err":   err.Error(),
			"id":    visitID,
		})
		return
	}
	// does the actual visit have a response?
	var visitcheck models.Visit
	initializers.DB.Preload("VisitResponse").First(&visitcheck, visitID)
	if visitcheck.VisitResponse == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The given visit does not have a visitresponse",
			"id":    visitID,
		})
		return
	}

	pdfBytes, err := internal.GeneratePDFVisit(uint(visitID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"id":    visitID,
		})
		return
	}
	/*
		ok := internal.AddNoteToAdvopro(visitcheck)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Advopro integration went wrong",
				"id":    visitcheck.Sagsnr,
			})
			return
		}
	*/
	var visit models.Visit
	initializers.DB.First(&visit, visitID)
	filename := "id" + strconv.Itoa(int(visit.ID)) + "_sagsnr" + strconv.Itoa(int(visit.Sagsnr)) + ".pdf"

	// Set headers for PDF download
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))

	// Send PDF bytes
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// savePDFTemp writes pdfBytes to a temp file and returns its path.
// Caller is responsible for removing it.
func savePDFTemp(pdfBytes []byte, name string) (string, error) {
	f, err := os.CreateTemp("", "visit-"+name+"-*.pdf")
	// → visit-20260611-430415-1234567890.pdf
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(pdfBytes); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	return f.Name(), nil
}

// Upload to Advopro
func ReviewedVisit(c *gin.Context) {
	user, ok := getVerifyUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, nil)
		return
	}

	var body struct {
		ReviewedIds []uint `json:"reviewed_ids"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	type iErr struct {
		Err string `json:"err"`
		ID  uint   `json:"id"`
	}

	var iErrs []iErr

	for _, visitId := range body.ReviewedIds {
		item := iErr{ID: visitId, Err: "no error"}

		// 1. Get sagsnr
		var visit models.Visit
		if res := initializers.DB.Preload("VisitResponse").First(&visit, visitId); res.Error != nil {
			item.Err = res.Error.Error()
			iErrs = append(iErrs, item)
			continue
		}

		// 1.5 generate a name
		if visit.VisitResponse == nil {
			item.Err = "Visit has no visitresponse"
			iErrs = append(iErrs, item)
			continue
		}
		docTitle := "besog " + visit.VisitResponse.ActDate.Format("2006-01-02") + "-" + strconv.FormatUint(uint64(visit.Sagsnr), 10)

		// 2. Generate PDF
		pdfBytes, err := internal.GeneratePDFVisit(visitId)
		if err != nil {
			item.Err = err.Error()
			iErrs = append(iErrs, item)
			continue
		}

		// 3. Save to temp file
		path, err := savePDFTemp(pdfBytes, docTitle)
		if err != nil {
			item.Err = err.Error()
			iErrs = append(iErrs, item)
			continue
		}

		// 4. Upload document
		if err := internal.UploadDocument(path, uint64(visit.Sagsnr), docTitle); err != nil {
			os.Remove(path)
			item.Err = err.Error()
			iErrs = append(iErrs, item)
			continue
		}
		os.Remove(path) // clean up temp file after successful upload

		// 5. Only NOW update the status, since the document is on the case
		if err := internal.UpdateVisitStatus(visitId, 5, user.ID); err != nil {
			item.Err = err.Error()
			iErrs = append(iErrs, item)
			continue
		}

		iErrs = append(iErrs, item)
	}
	// If every item has an error, return 500
	allFailed := len(iErrs) > 0
	for _, e := range iErrs {
		if e.Err == "no error" {
			allFailed = false
			break
		}
	}

	if allFailed {
		c.JSON(http.StatusInternalServerError, iErrs)
		return
	}

	c.JSON(http.StatusOK, iErrs)
}
