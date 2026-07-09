package internal

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	fpdf "github.com/phpdave11/gofpdf"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func SaveFile(c *gin.Context, file *multipart.FileHeader) (string, error) {
	// Create upload directory if not exists
	uploadDir := "uploads/visit_images"
	os.MkdirAll(uploadDir, 0755)

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(),
		strings.ReplaceAll(uuid.New().String(), "-", ""), ext)

	filepath := filepath.Join(uploadDir, filename)

	// Save file
	if err := c.SaveUploadedFile(file, filepath); err != nil {
		return "", err
	}

	return filepath, nil
}

func addImageFit(pdf *fpdf.Fpdf, path string) error {
	// Page size, margins, cursor
	pw, ph := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	y := pdf.GetY()

	// Bottom margin (via auto page break)

	_, bm := pdf.GetAutoPageBreak() //ab

	maxW := pw - lm - rm
	maxH := ph - bm - y
	if maxH <= 0 {
		pdf.AddPage()
		y = pdf.GetY()
		maxH = ph - bm - y
	}
	// is image there?
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("cant find image at %s: %w", path, err)
	}

	// Image natural size (respect DPI)
	info := pdf.RegisterImageOptions(path, fpdf.ImageOptions{ReadDpi: true})
	iw, ih := info.Extent()

	scale := math.Min(math.Min(maxW/iw, maxH/ih), 1.0)
	w, h := iw*scale, ih*scale

	// Draw
	pdf.ImageOptions(path, lm, y, w, h, false, fpdf.ImageOptions{ReadDpi: true}, 0, "")
	pdf.SetY(y + h + 2) // move cursor below image
	return nil
}

// Helper to handle optional numbers (uint)
func optionalUintToStr(val *uint) string {
	if val == nil {
		return "-"
	}
	return fmt.Sprint(*val)
}

// Helper to handle optional money (float32)
func optionalMoneyToStr(val *float32) string {
	if val == nil {
		return "-"
	}
	p := message.NewPrinter(language.Danish)
	return p.Sprintf("%.2f kr", *val)
}

func optionalBoolToStr(val *bool) string {
	if val == nil {
		return "-" // Or "Ingen data"
	}
	if *val {
		return "JA"
	}
	return "NEJ"
}

func questionRow(pdf *fpdf.Fpdf, label string, answer string, details string) {

	// should sum to 90
	//
	labelW := 42.0
	answerW := 20.0
	detailW := 30.0
	x, _ := pdf.GetXY()

	pdf.SetFontStyle("B")

	pdf.CellFormat(labelW, 6, label, "", 0, "", false, 0, "")

	pdf.SetFontStyle("")

	pdf.CellFormat(answerW, 6, answer, "", 0, "", false, 0, "")

	pdf.CellFormat(detailW, 6, details, "", 1, "", false, 0, "")
	_, y := pdf.GetXY()
	pdf.SetXY(x, y)
}

func civilStatusToString(status string) string {
	return status
}

func optionalpropertyTypeToString(propertytype *models.PropertyType) string {
	if propertytype == nil || *propertytype == "" {
		return "-"
	}

	return string(*propertytype)
}

func optionalMaintenanceToString(maintain_status *models.MaintenanceStatus) string {
	if maintain_status == nil || *maintain_status == "" {
		return "-"
	}

	return string(*maintain_status)
}

var pdfnormalFontSize float64 = 10
var pdflargerFontSize float64 = 30

func pdfHeader(pdf *fpdf.Fpdf, v models.Visit) {
	// this will be the overview box or header

	//box dimensions
	box_width := float64(190)
	box_heigt := float64(60)
	box_cornerX := float64(10)
	box_cornerY := float64(20)

	// LEASING
	// |---------------------------------------------------------------------------|
	// | Sagsnr    xxx-xxx         Adresse:  hyldekærparken    besøgsid:xxxxx      |
	// | dato for besøg 2026-03-04  kl: 21:30                                      |
	// | Debitorer:                                                                |
	// |  peter hansen, tlf: xxxx xxxx   mail: peter@hansen.com  debitorId:XXXXX   |
	// |  berit hansen, tlf: xxxx xxxx   mail: berit@hansen.com  debitorId:XXXXX   |
	// | konsulent: Markus kjeldsen,   tlfnr: xxxx xxxx                            |
	// |                                                                           |
	// |---------------------------------------------------------------------------|
	// the page is 210 wide and the box is 190 wide
	//0_ 10 ---------------------------------------------------- 200_210

	pdf.SetXY(10, 10)
	pdf.SetFont("Roboto", "", pdflargerFontSize)
	pdf.CellFormat(0, 10, strings.ToUpper(v.Type.Text), "", 1, "", false, 0, "")
	pdf.SetFont("Roboto", "", pdfnormalFontSize)

	// first a large box
	pdf.Rect(box_cornerX, box_cornerY, box_width, box_heigt, "D")
	pdf.SetXY(10, 20)

	// Case information - Adjusted widths to prevent Address overlap
	pdf.CellFormat(20, 6, "Sagsnr:", "", 0, "", false, 0, "")
	pdf.CellFormat(30, 6, fmt.Sprint(v.Sagsnr), "", 0, "", false, 0, "")
	pdf.CellFormat(15, 6, "Adresse:", "", 0, "", false, 0, "")
	// We use a slightly smaller width here and let it truncate or handle logic
	pdf.CellFormat(85, 6, v.Address, "", 0, "L", false, 0, "")
	pdf.CellFormat(15, 6, "BesøgsId:", "", 0, "", false, 0, "")
	pdf.CellFormat(25, 6, fmt.Sprint(v.ID), "", 1, "R", false, 0, "")

	// Visit information
	pdf.CellFormat(20, 6, "Dato", "", 0, "", false, 0, "")
	pdf.CellFormat(30, 6, v.VisitDate.Format("2006-01-02"), "", 0, "", false, 0, "")
	pdf.CellFormat(10, 6, "Kl:", "", 0, "", false, 0, "")
	pdf.CellFormat(40, 6, v.VisitResponse.ActTime[:5], "", 1, "", false, 0, "")

	pdf.Ln(2) // Small gap
	pdf.CellFormat(40, 6, "Debitorer:", "", 1, "", false, 0, "")
	//debitor information

	for _, deb := range v.Debitors {
		phone := strings.TrimSpace(deb.Phone)
		if phone == "" {
			phone = deb.PhoneWork
		}
		AdvoproDebitor := fmt.Sprint(deb.AdvoproDebitorId)

		// 1. NAME LINE (Using MultiCell to allow wrapping)
		pdf.SetX(15)          // Indent
		pdf.SetFontStyle("B") // Optional: make name bold
		pdf.CellFormat(12, 5, "Navn:", "", 0, "", false, 0, "")
		pdf.SetFontStyle("")

		// MultiCell will wrap the text if it exceeds 160 units
		// The '1' at the end moves the cursor to the next line automatically
		pdf.MultiCell(160, 5, deb.Name, "", "L", false)

		// 2. CONTACT INFO LINE (Below the name)
		pdf.SetX(20)                           // Further indent for details
		pdf.SetFontSize(pdfnormalFontSize - 1) // Optional: slightly smaller text for details

		contactLine := fmt.Sprintf("tlf: %s  |  mail: %s  |  debitorId: %s", phone, deb.Email, AdvoproDebitor)
		pdf.CellFormat(0, 5, contactLine, "", 1, "L", false, 0, "")

		pdf.Ln(1) // Small space between different debitors
		pdf.SetFontSize(pdfnormalFontSize)
	}
	// time spent
	// duration := time.Duration(v.VisitResponse.Duration) * time.Millisecond

	// worker information
	pdf.SetXY(box_cornerX, box_cornerY+box_heigt-7)
	pdf.CellFormat(20, 6, "Konsulent:", "", 0, "", false, 0, "")
	pdf.CellFormat(40, 6, v.User.Name, "", 0, "", false, 0, "")
	pdf.CellFormat(10, 6, "tlf:", "", 0, "", false, 0, "")
	pdf.CellFormat(40, 6, v.User.Phone, "", 0, "", false, 0, "")
	//pdf.CellFormat(25, 6, "tidsforbrug:", "", 0, "", false, 0, "")
	//pdf.CellFormat(0, 6, duration.String(), "", 1, "R", false, 0, "")

}

func fillLifeBox(pdf *fpdf.Fpdf, v models.Visit, LifeBoxX float64, LifeBoxY float64, LifeBoxWidth float64) {
	pdf.SetXY(LifeBoxX, LifeBoxY-6)
	pdf.SetFont("Roboto", "B", pdfnormalFontSize+3)
	pdf.CellFormat(LifeBoxWidth, 6, "LIVSSITUATION", "", 0, "L", false, 0, "")
	pdf.SetFont("Roboto", "", pdfnormalFontSize-1)

	paddingX := 4.0
	paddingY := 6.0
	// right side
	pdf.SetXY(LifeBoxX+paddingX, LifeBoxY+paddingY)

	questionRow(pdf, "Debitor hjemme", optionalBoolToStr(v.VisitResponse.Contact.DebitorMet), "")
	questionRow(pdf, "Civilstatus", civilStatusToString(v.VisitResponse.Monetary.CivilStatus), "")
	questionRow(pdf, "Børn u/18 hjemme", optionalUintToStr(v.VisitResponse.Monetary.ChildrenUnder18), "")
	//questionRow(pdf, "bønr u/18 udeboende", optionalUintToStr(v.VisitResponse.ChildrenUnder18), "")

	// Complex logic for child support
	/* new structs dont have this field anymore.
	childSupportDetails := ""
	if v.VisitResponse.ChildSupport != nil {
		childSupportDetails = optionalMoneyToStr(v.VisitResponse.ChildSupport)
	}
	*/

	// Assuming ChildSupport existence depends on if the float is > 0 or if the pointer is just present
	/* we dont ask about that with the new visitresponse
	hasChildSupportStr := "-"
	if v.VisitResponse.ChildSupport != nil {
		if *v.VisitResponse.ChildSupport > 0 {
			hasChildSupportStr = "JA"
		} else {
			hasChildSupportStr = "NEJ"
		}
	}
	questionRow(pdf, "Børnepenge", hasChildSupportStr, childSupportDetails)
	*/
	salary := ""
	if v.VisitResponse.Monetary.HasWork != nil {
		if *v.VisitResponse.Monetary.HasWork {
			salary = v.VisitResponse.Monetary.NetSalaryMin.FormatDK() + " " + v.VisitResponse.Monetary.NetSalaryMax.FormatDK()
		}
	}

	questionRow(pdf, "Arbejde", optionalBoolToStr(v.VisitResponse.Monetary.HasWork), v.VisitResponse.Monetary.Position)
	questionRow(pdf, "Arbejde inkosmt", "", salary)
	questionRow(pdf, "Off. ydelser", "", v.VisitResponse.Monetary.IncomePaymentMin.FormatDK()+" "+v.VisitResponse.Monetary.IncomePaymentMax.FormatDK())

	totalStr := "-" // total of offentlige ydelser and inkomst

	questionRow(pdf, "Total udbetalt", "", totalStr)
	questionRow(pdf, "Rådighedsbeløb", "", v.VisitResponse.Monetary.MonthlyDisposableMin.FormatDK()+"-"+v.VisitResponse.Monetary.MonthlyDisposableMax.FormatDK())

	questionRow(pdf, "Hus?", optionalpropertyTypeToString(v.VisitResponse.Property.PropertyType), "")
	// TODO: implement a custom visual aid for
	// v.VisitResponse.Property.BrokenWindows
	// v.VisitResponse.Property.ForsaleSign
	// v.VisitResponse.Property.MailboxFull
	// v.VisitResponse.Property.OvergrownGarden
	// v.VisitResponse.Property.AbandonedVehicles
	// v.VisitResponse.Property.TrashOverflown

	//questionRow(pdf, "Ejerskab?", "", v.VisitResponse.OwnershipStatus) // we no longer ask about this
}

func fillCarBox(pdf *fpdf.Fpdf, v models.Visit, CarBoxX float64, CarBoxY float64, CarWidth float64) {

	pdf.SetXY(CarBoxX, CarBoxY-6)
	pdf.SetFont("Roboto", "B", pdfnormalFontSize+3)
	pdf.CellFormat(CarWidth, 6, "BIL", "", 0, "L", false, 0, "")
	pdf.SetFont("Roboto", "", pdfnormalFontSize-1)

	paddingX := 4.0
	paddingY := 6.0
	pdf.SetXY(CarBoxX+paddingX, CarBoxY+paddingY)

	y := CarBoxY + 5

	pdf.SetXY(CarBoxX, y)

	questionRow(pdf, "Aktiv Skadet?", v.VisitResponse.Assets.AssetStatus, "")
	questionRow(pdf, "Received keys", optionalBoolToStr(v.VisitResponse.Assets.AssetKeysDelivered), "")
	questionRow(pdf, "Er den på adressen?", optionalBoolToStr(v.VisitResponse.Assets.AssetSeen), "")
	questionRow(pdf, "Er den ren?", v.VisitResponse.Assets.AssetCleanliness, "")
	questionRow(pdf, "Bilen afleveret?", optionalBoolToStr(v.VisitResponse.Assets.IsSeized), "")
	questionRow(pdf, "Salgsfuldmagt underskrevet", optionalBoolToStr(v.VisitResponse.Assets.SFSigned), "")

	//questionRow(pdf, "Skylderklæring underskrevet", optionalBoolToStr(v.VisitResponse.SESigned), "")

}

func fillFinanceBox(pdf *fpdf.Fpdf, v models.Visit, FinanceBoxX float64, FinanceBoxY float64, FinanceWidth float64) {
	pdf.SetXY(FinanceBoxX, FinanceBoxY-6)
	pdf.SetFont("Roboto", "B", pdfnormalFontSize+3)
	pdf.CellFormat(FinanceWidth, 6, "Anden gæld", "", 0, "L", false, 0, "")
	pdf.SetFont("Roboto", "", pdfnormalFontSize-1)

	paddingX := 4.0
	paddingY := 6.0
	pdf.SetXY(FinanceBoxX+paddingX, FinanceBoxY+paddingY)

	y := FinanceBoxY + 5

	pdf.SetXY(FinanceBoxX, y)

	// ask about
	// v.VisitResponse.Creditor
	// v.VisitResponse.DebtAmount
	// først disposable income

	text := ""
	amountStr := ""

	incomeMax := v.VisitResponse.Monetary.IncomePaymentMax
	incomeMin := v.VisitResponse.Monetary.IncomePaymentMin
	if incomeMin != nil && incomeMax != nil {
		amountStr = incomeMin.FormatDK() + incomeMax.FormatDK()
	}
	questionRow(pdf, "overførsler", amountStr, "")

	text = ""
	amountStr = ""
	salMax := v.VisitResponse.Monetary.NetSalaryMax
	salMin := v.VisitResponse.Monetary.NetSalaryMin
	if salMax != nil && salMin != nil {
		amountStr = salMin.FormatDK() + salMax.FormatDK()
	}
	questionRow(pdf, "netto inkomst", amountStr, "")

	text = ""
	amountStr = ""
	mMax := v.VisitResponse.Monetary.MonthlyDisposableMax
	mMin := v.VisitResponse.Monetary.MonthlyDisposableMin
	if mMax != nil && mMin != nil {
		amountStr = mMin.FormatDK() + mMax.FormatDK()
	}
	questionRow(pdf, "Rådigheds beløb", amountStr, "")

	text = ""
	amountStr = "0,00 kr."
	if p := v.VisitResponse.Monetary.DebtAmountPaid; p != nil {
		if float64(*p)/100.0 > 0 {
			text = "Penge som bliver brugt på anden gæld"
		}
		amountStr = p.FormatDK()
	}
	questionRow(pdf, "anden gæld", text, amountStr)
}

func fillCommentsBox(pdf *fpdf.Fpdf, v models.Visit, CommentsBoxX float64, CommentsBoxY float64, CommentsWidth float64) {
	pdf.SetXY(CommentsBoxX, CommentsBoxY-6)
	pdf.SetFont("Roboto", "B", pdfnormalFontSize+3)
	pdf.CellFormat(CommentsWidth, 6, "Kommentarer", "", 0, "L", false, 0, "")
	pdf.SetFont("Roboto", "", pdfnormalFontSize-1)

	paddingX := 4.0
	paddingY := 6.0
	pdf.SetXY(CommentsBoxX+paddingX, CommentsBoxY+paddingY)

	y := CommentsBoxY + 5

	pdf.SetXY(CommentsBoxX, y)

	comment := v.VisitResponse.Comments
	pdf.MultiCell(CommentsWidth, 5, comment, "", "TL", false)

}

func pdfBody(pdf *fpdf.Fpdf, v models.Visit) {
	// -----------------------------------------
	// |HEADER already prefilled               |
	// -----------------------------------------
	//
	// CAR                             life satus
	// ----------------------------    -------------------------
	// | Q?       A!     details  |    | Q?       A!    details|
	// |Destryed?  YES            |    |Civilstatus married    |
	// |                          |    |kids u/18 home  3      |
	// |                          |    |kids u/18 nothome 5    |
	// |                          |    |childsupport   500kr/md|
	// |                          |    |work?      yes   janitor|
	// |                          |    |work income 1000kr/md  |
	// |                          |    |off.ydelser  1000kr/md |
	// |                          |    |totaludbetalt 2000kr/md|
	// |                          |    |rådighedsbeløb 200kr/md|
	// |                          |    |house?                 |
	// |                          |    |owneship of home?      |
	// ----------------------------    ------------------------
	//

	// sizing
	boxHeightCar := 50.0
	boxHeightLife := 110.0
	boxHeightFinance := 25.0
	boxHeightComments := 50.0

	boxWidth := 90.0
	CommentsWidth := 190.0

	// header placement
	//HEADER_width := float64(190)
	HEADER_heigt := float64(60) // Increased from 50 to 60
	HEADER_cornerX := float64(10)
	HEADER_cornerY := float64(20)

	// how low the boxes are
	CarLifeY := 10.0 + HEADER_cornerY + HEADER_heigt // 30 margin

	CarX := HEADER_cornerX // should be the same as the header
	LifeX := CarX + boxWidth + 10.0

	financeY := 10.0 + CarLifeY + boxHeightCar
	CommentsY := 10.0 + CarLifeY + boxHeightLife // comments are below finance

	financeX := HEADER_cornerX  // should be the same as the header
	CommentsX := HEADER_cornerX // should be the same as the header

	// left box (CAR)
	pdf.Rect(CarX, CarLifeY, boxWidth, boxHeightCar, "D")
	fillCarBox(pdf, v, CarX, CarLifeY, boxWidth)

	// right box (LIFE STATUS)
	pdf.Rect(LifeX, CarLifeY, boxWidth, boxHeightLife, "D")
	fillLifeBox(pdf, v, LifeX, CarLifeY, boxWidth)

	// gæld
	pdf.Rect(financeX, financeY, boxWidth, boxHeightFinance, "D")
	fillFinanceBox(pdf, v, financeX, financeY, boxWidth)

	// commentarer
	pdf.Rect(CommentsX, CommentsY, CommentsWidth, boxHeightComments, "D")
	fillCommentsBox(pdf, v, CommentsX, CommentsY, CommentsWidth)

}

func pdfGenerate(pdf *fpdf.Fpdf, v models.Visit) error {
	pdf.SetAutoPageBreak(false, 15)
	pdf.AddUTF8Font("Roboto", "", "./static/Roboto-light.ttf")
	pdf.AddUTF8Font("Roboto", "B", "./static/Roboto-Bold.ttf")

	pdf.SetFont("Roboto", "", pdfnormalFontSize)

	pdf.AddPage()

	//tpl := gofpdi.ImportPage(pdf, "./static/Besøgsbrev bilbesøg.pdf", 1, "/MediaBox")
	//gofpdi.UseImportedTemplate(pdf, tpl, 0, 0, 210, 0)

	// Now position your fields on top, same as with the image approach
	// helper functions
	pdfHeader(pdf, v)
	// header includes top info about the case, who is involved, where and when

	pdfBody(pdf, v)
	// more descriptive about the visit

	// til slut billederne
	for _, image := range v.VisitResponse.Images {
		pdf.AddPage()
		err := addImageFit(pdf, image.ImagePath)
		if err != nil {
			return fmt.Errorf("An error occurred when putting in images: %w", err)
		}
	}
	return nil
}

func GeneratePDFVisit(visitID uint) ([]byte, error) {

	var visit models.Visit
	initializers.DB.Preload("Type").Preload("Debitors").Preload("VisitResponse").Preload("VisitResponse.Images").Preload("User").First(&visit, visitID)

	re := regexp.MustCompile(`[<>:"/\\|?*\s]`)
	sanitizedAddress := re.ReplaceAllString(visit.Address, "_")
	sanitizedAddress = strings.ReplaceAll(sanitizedAddress, "__", "_")
	filename := fmt.Sprintf("pdfs/visit_%d_%s.pdf", visitID, sanitizedAddress)
	os.MkdirAll("pdfs", os.ModePerm)

	pdfBuf := fpdf.New("P", "mm", "A4", "")
	pdfFile := fpdf.New("P", "mm", "A4", "")

	//PdfReport(pdfBuf, visit)
	//PdfReport(pdfFile, visit)
	err := pdfGenerate(pdfBuf, visit)
	if err != nil {
		return nil, err
	}
	err = pdfGenerate(pdfFile, visit)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = pdfBuf.Output(&buf)
	if err != nil {
		log.Printf("error outputting PDF to buffer: %v", err)
		return nil, err
	}

	err = pdfFile.OutputFileAndClose(filename)
	if err != nil {
		log.Printf("error outputting PDF to file: %v", err)
		return nil, err
	}

	return buf.Bytes(), nil // data , ok
}
