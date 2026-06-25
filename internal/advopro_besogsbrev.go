package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
)

// IMPLEMENT BESOGSBREV RETRIVAL
func GetBesogsbrev(visitId uint64) ([]byte, error) {

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
	if len(advoproResult) > 0 {
		// if more then one they should be the same, but just take the most recent which is the top one
		fmt.Print("More then one besøgsbrev file for this case, using the latest")
	}

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
		letterPath := filepath.Join(linuxMount, relPath, winFilnavn)

		// Optional: Verify file exists on disk
		if _, err := os.Stat(letterPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist at path: %s", letterPath)
		}
	}

	fileBytes, err := os.ReadFile(letterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	return fileBytes, nil
}
