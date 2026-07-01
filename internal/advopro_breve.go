package internal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
)

// ConvertDocxToPdf converts a .docx file to .pdf using LibreOffice headless.
// Returns the path to the generated PDF file.
// ponytail: global lock — LibreOffice single-instance limit. Per-user locks if throughput matters.
var libreOfficeMu sync.Mutex

func ConvertDocxToPdf(docxPath string) (string, error) {
	outDir := filepath.Dir(docxPath)

	libreOfficeBin := "libreoffice"
	if runtime.GOOS == "windows" {
		libreOfficeBin = `C:\Program Files\LibreOffice\program\soffice.exe`
	}

	libreOfficeMu.Lock()
	cmd := exec.Command(libreOfficeBin, "--headless", "--convert-to", "pdf", "--outdir", outDir, docxPath)
	output, err := cmd.CombinedOutput()
	libreOfficeMu.Unlock()

	if err != nil {
		return "", fmt.Errorf("libreoffice conversion failed: %w, Output: %s", err, string(output))
	}

	base := filepath.Base(docxPath)
	ext := filepath.Ext(base)
	pdfName := base[:len(base)-len(ext)] + ".pdf"
	pdfPath := filepath.Join(outDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return "", fmt.Errorf("expected PDF not found at: %s", pdfPath)
	}

	return pdfPath, nil
}

func MergePDFs(pdfs [][]byte) ([]byte, error) {
	readers := make([]io.ReadSeeker, len(pdfs))
	for i, b := range pdfs {
		readers[i] = bytes.NewReader(b)
	}
	var out bytes.Buffer
	if err := api.MergeRaw(readers, &out, false, nil); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ExtractPDFPage extracts a single page from a PDF and returns the bytes.
// pageNum is 1-indexed.
func ExtractPDFPage(pdfBytes []byte, pageNum int) ([]byte, error) {
	// Write input bytes to a temp file
	tmpIn, err := os.CreateTemp("", "input-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err := tmpIn.Write(pdfBytes); err != nil {
		return nil, fmt.Errorf("failed to write temp input file: %w", err)
	}
	tmpIn.Close()

	// Create a temp file for output
	tmpOut, err := os.CreateTemp("", "output-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	tmpOut.Close()
	defer os.Remove(tmpOut.Name())

	// Extract the page using pdfcpu
	selectedPages := []string{fmt.Sprintf("%d", pageNum)}
	if err := api.CollectFile(tmpIn.Name(), tmpOut.Name(), selectedPages, nil); err != nil {
		return nil, fmt.Errorf("failed to extract page %d: %w", pageNum, err)
	}

	outBytes, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read output PDF: %w", err)
	}

	return outBytes, nil
}

// internal/documents.go

// ponytail: shared fetch+convert, page param selects the slice
func getDocPage(visitId uint64, requireTypeID *uint, page int) ([]byte, error) {
	var visit models.Visit
	if err := initializers.DB.First(&visit, visitId).Error; err != nil {
		return nil, err
	}
	if requireTypeID != nil && visit.TypeID != *requireTypeID {
		return nil, fmt.Errorf("visit %d is not type %d", visitId, *requireTypeID)
	}

	query := `SELECT sf.Sagsnr, sf.Placering, sf.Filnavn, sf.Tekst, sf.Tidspunkt
    FROM vwKlientSagsforlob sf
    WHERE sf.Sagsnr = @p1
    AND sf.Extension = 'docx'
    AND (
        LOWER(sf.Tekst) LIKE '%besøgsbrev blanco sendt%' OR
        LOWER(sf.Tekst) LIKE '%besøgsbrev bil sendt%'
    )
    ORDER BY sf.Tidspunkt desc`

	rows, err := ExecuteQuery(Server, AdvoPro, query, visit.Sagsnr)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no document found for case %s", visit.Sagsnr)
	}

	letterPath := resolveDocPath(toString(rows[0]["Placering"]), toString(rows[0]["Filnavn"]))
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(letterPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist at path: %s", letterPath)
		}
	}

	pdfPath, err := ConvertDocxToPdf(letterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document to PDF: %w", err)
	}
	defer os.Remove(pdfPath)

	fileBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	return ExtractPDFPage(fileBytes, page)
}

func resolveDocPath(winPlacering, winFilnavn string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(winPlacering, winFilnavn)
	}
	rel := strings.TrimPrefix(winPlacering, `\\MOPSRV01\AdvoPro`)
	rel = strings.ReplaceAll(rel, "\\", "/")
	return filepath.Join("/mnt/advopro", rel, winFilnavn)
}

func GetBesogsbrev(visitId uint64) ([]byte, error) {
	return getDocPage(visitId, nil, 2)
}

func GetSF(visitId uint64) ([]byte, error) {
	typeID := uint(1)
	return getDocPage(visitId, &typeID, 3)
}

// skyldnererklæring
func GetSE(visitId uint64) ([]byte, error) {

	var visit models.Visit
	result := initializers.DB.First(&visit, visitId)
	if result.Error != nil {
		return nil, result.Error
	}

	query := `SELECT sf.Sagsnr, sf.Placering, sf.Filnavn, sf.Tekst, sf.Tidspunkt
	FROM vwKlientSagsforlob sf
	WHERE sf.Sagsnr = @p1
	AND sf.Extension = 'docx'
	AND (
		LOWER(sf.Tekst) LIKE '%besøgsbrev blanco sendt%' OR 
		LOWER(sf.Tekst) LIKE '%besøgsbrev bil sendt%'
		)
	ORDER BY sf.Tidspunkt desc`

	advoproResult, err := ExecuteQuery(Server, AdvoPro, query, visit.Sagsnr)
	if err != nil {
		return nil, err
	}

	if len(advoproResult) == 0 {
		return nil, fmt.Errorf("There was no result from the database")
	}
	//if len(advoproResult) > 0 { this warning dosnt matter
	// if more then one they should be the same, but just take the most recent which is the top one
	//	logger.Info("More then one besøgsbrev file for this case, using the latest")
	//}

	winPlacering := toString(advoproResult[0]["Placering"]) // "\\MOPSRV01\AdvoPro\Opgaver\..."
	winFilnavn := toString(advoproResult[0]["Filnavn"])     // "99999999.docx"
	letterPath := ""
	// 1. If running on your local Windows machine
	if runtime.GOOS == "windows" {
		// Windows handles backslashes and UNC paths (\\Server\Share) natively
		letterPath = filepath.Join(winPlacering, winFilnavn)
	} else {
		// 2. Define the translation rules
		winPrefix := `\\MOPSRV01\AdvoPro`
		linuxMount := "/mnt/advopro"

		// 3. Translate the path
		// Remove the Windows server prefix
		relPath := strings.TrimPrefix(winPlacering, winPrefix)

		// Convert Windows backslashes (\) to Linux forward slashes (/)
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// 4. Combine into a final Linux path
		letterPath = filepath.Join(linuxMount, relPath, winFilnavn)

		// Optional: Verify file exists on disk
		if _, err := os.Stat(letterPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist at path: %s", letterPath)
		}
	}

	// Convert the .docx to .pdf
	pdfPath, err := ConvertDocxToPdf(letterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document to PDF: %w", err)
	}
	// Clean up the temporary PDF after reading (optional but tidy)
	defer os.Remove(pdfPath)

	fileBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Extract only page 3
	fileBytes, err = ExtractPDFPage(fileBytes, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to extract page: %w", err)
	}

	return fileBytes, nil
}
