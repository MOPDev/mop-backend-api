package internal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/models"
	_ "github.com/denisenkom/go-mssqldb"
)

type AdvoProCaseData struct {
	Sagsnr       uint
	Status       int
	StatusText   string
	DeadlineDate time.Time
	KlientNavn   string
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}

func toTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch t := v.(type) {
	case time.Time:
		return t
	case []byte:
		s := string(t)
		tt, _ := time.Parse(time.RFC3339, s)
		return tt
	case string:
		tt, _ := time.Parse(time.RFC3339, t)
		return tt
	default:
		return time.Time{}
	}
}

func ExecuteQuery(server, database, query string, params ...interface{}) ([]map[string]interface{}, error) {
	user := os.Getenv("MSSQL_USER")
	pass := os.Getenv("MSSQL_PASS")

	// Option A: ODBC-style
	conn := fmt.Sprintf(
		"server=%s;user id=%s;password=%s;database=%s;encrypt=true;TrustServerCertificate=true;port=1433;connection timeout=5",
		server, user, pass, database,
	)

	// Option B: URL-style
	// conn := fmt.Sprintf("sqlserver://%s:%s@%s:1433?database=%s&encrypt=true&TrustServerCertificate=true",
	// 	 url.QueryEscape(user), url.QueryEscape(pass), server, database)

	db, err := sql.Open("sqlserver", conn)
	if err != nil {
		fmt.Print(err.Error())
		return nil, fmt.Errorf("server could not be opened: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(query, params...)
	if err != nil {
		fmt.Println(query)
		fmt.Println(params...)
		fmt.Print(err.Error())
		return nil, fmt.Errorf("Query could not be executed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		fmt.Print(err.Error())
		return nil, fmt.Errorf("failed to fetch column names from database: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range cols {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Println("Somthing went wrong in the data parsing")
			return nil, err
		}

		row := map[string]interface{}{}
		for i, c := range cols {
			row[c] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func FetchBulkCaseData(sagsnumre []uint) (map[uint]AdvoProCaseData, error) {
	if len(sagsnumre) == 0 {
		return nil, nil
	}

	// Create placeholders for the IN clause (@p1, @p2, @p3)
	placeholders := make([]string, len(sagsnumre))
	args := make([]interface{}, len(sagsnumre))
	for i, id := range sagsnumre {
		placeholders[i] = fmt.Sprintf("@p%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
        SELECT 
            f.Sagsnr,
            f.Status,
            f.Fristdato,
            s.KlientNavn,
            inks.Tekst
        FROM vwInkassoForlob f
        JOIN vwInkassoSag s ON s.Sagsnr = f.Sagsnr
        JOIN vwInkassoStatus inkS ON inkS.Statuskode = f.Status
        WHERE f.Sagsnr IN (%s)`, strings.Join(placeholders, ","))

	results, err := ExecuteQuery(Server, AdvoPro, query, args...)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	// TODO: WHAT IF 2 HAVE THE SAME SAGSNR?

	// Map results by Sagsnr for easy lookup
	caseMap := make(map[uint]AdvoProCaseData)
	for _, res := range results {
		sagsnr := uint(res["Sagsnr"].(int64))
		caseMap[sagsnr] = AdvoProCaseData{
			Sagsnr:       sagsnr,
			Status:       int(res["Status"].(int64)),
			StatusText:   res["Tekst"].(string),
			DeadlineDate: res["Fristdato"].(time.Time),
			KlientNavn:   res["KlientNavn"].(string),
		}
	}

	return caseMap, nil
}

func FetchDebitorData(debitorNum int64) *models.Debitor {
	debitors, err := ExecuteQuery(Server, AdvoPro, debitorQuery, debitorNum)
	if err != nil {
		fmt.Println("Something went wrong during the fetch of a debitor")
		fmt.Println(err)
		return nil
	}
	if len(debitors) > 1 {
		fmt.Println("There is more than one Debitor with this debitorID")
		return nil
	}
	if len(debitors) == 0 {
		fmt.Println("There is not any debitor with this ID")
		return nil
	}
	debitor := debitors[0]

	name, ok1 := debitor["Navn"].(string)
	birthday, ok2 := debitor["Fodselsdato"].(time.Time)
	genderNum, ok3 := debitor["Kon"].(int)
	phone, ok4 := debitor["Telefon"].(string)
	mobilePhone, ok5 := debitor["Mobiltlf"].(string)
	workPhone, ok6 := debitor["Arbejdstlf"].(string)
	email, ok7 := debitor["EPost"].(string)

	if !ok1 && !ok2 && !ok3 && !ok4 && !ok5 && !ok6 && !ok7 {
		fmt.Println("Formatting from the database has gone wrong")
		return nil
	}

	var phoneNr string
	if mobilePhone != "" {
		phoneNr = mobilePhone
	} else if phone != "" {
		phoneNr = phone
	} else if workPhone != "" {
		phoneNr = workPhone
	}

	var gender models.Gender
	switch genderNum {
	case 0:
		gender = models.Male
	case 1:
		gender = models.Female
	default:
		gender = models.Other
	}

	deb := models.Debitor{
		AdvoproDebitorId: int(debitorNum),
		Name:             name,
		Birthday:         birthday,
		Gender:           gender,
		Email:            email,
		Phone:            phoneNr,
		PhoneWork:        workPhone,
	}

	return &deb
}

type DebtRow struct {
	SumIndbetalinger      float64
	RestgeldAntaget       float64
	RestanceDato          time.Time // use appropriate type, e.g. time.Time or string
	KreditorHovedstol     float64
	RestgeldVedBrev       float64
	SumIndbetalingVedBrev float64
	Fordringsbeskrivelser string
	Sagsfremstillinger    string
}

func CurrentDebtCase(sagsnr uint) ([]DebtRow, error) {
	// 1. Error handling with context
	debts, err := ExecuteQuery(Server, AdvoPro, debtInfo, sagsnr)
	if err != nil {
		return nil, fmt.Errorf("failed to execute debt query for sagsnr %d: %w", sagsnr, err)
	}

	// 2. Pre-allocate slice capacity to avoid re-allocations in the loop
	result := make([]DebtRow, 0, len(debts))

	for _, debt := range debts {
		// 3. Use a helper to safely extract values and prevent panics
		row := DebtRow{
			SumIndbetalinger:      safeByteToFloat(debt["SumIndbetalinger"]),
			RestgeldAntaget:       safeByteToFloat(debt["restgeldAntaget"]),
			RestanceDato:          toTime(debt["RestanceDato"]),
			KreditorHovedstol:     safeByteToFloat(debt["KreditorHovedstol"]),
			RestgeldVedBrev:       safeByteToFloat(debt["restgeldVedBrev"]),
			SumIndbetalingVedBrev: safeByteToFloat(debt["SumIndbetalingVedBrev"]),
			Fordringsbeskrivelser: toString(debt["Fordringsbeskrivelser"]),
			Sagsfremstillinger:    toString(debt["Sagsfremstillinger"]),
		}

		result = append(result, row)
	}

	return result, nil
}

// safeByteToFloat prevents panics if the value is nil or not []byte
func safeByteToFloat(val interface{}) float64 {
	b, ok := val.([]byte)
	if !ok || b == nil {
		return 0.0
	}
	return byteToFloat(b)
}

// Converts []byte to float64, returns 0.0 on error
func byteToFloat(b []byte) float64 {
	s := string(b)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0 // or handle error as needed
	}
	return f
}

func GetAktivitetsrapporten(visitId uint64) (string, error) {
	// from internal db get sagsnr
	// Get visit/case info from DB
	var visit models.Visit
	result := initializers.DB.First(&visit, visitId)
	if result.Error != nil {
		return "", result.Error
	}

	query := `
    SELECT sf.Sagsnr, sf.Placering, sf.Filnavn, sf.Tekst, sf.Tidspunkt
    FROM vwKlientSagsforlob sf
    WHERE sf.Sagsnr = @p1
    AND sf.Extension = 'docx'
    AND LOWER(sf.Tekst) LIKE '%aktivitetsrapport%'
    ORDER BY sf.Sagsnr`

	// Build path to the mounted drive
	//mountPath := os.Getenv("DRIVE_MOUNT_PATH") // e.g. "/mnt/external"
	advoproResult, err := ExecuteQuery(Server, AdvoPro, query, visit.Sagsnr)
	if err != nil {
		return "", err
	}

	// Sagsnr	Placering	Filnavn	Tekst	Tidspunkt
	// 636000	\\MOPSRV01\AdvoPro\Opgaver\Jurist\MBB\636000	99999999.docx	Aktivitetsrapport	2024-05-30 09:35:14.270
	// 636001	\\MOPSRV01\AdvoPro\Opgaver\Jurist\MBB\636001	99999999.docx	Aktivitetsrapport	2024-06-06 11:11:52.383

	if len(advoproResult) == 0 {
		return "", fmt.Errorf("There was no result")
	}

	winPlacering := toString(advoproResult[0]["Placering"]) // "\\MOPSRV01\AdvoPro\Opgaver\..."
	winFilnavn := toString(advoproResult[0]["Filnavn"])     // "99999999.docx"

	// 1. If running on your local Windows machine
	if runtime.GOOS == "windows" {
		// Windows handles backslashes and UNC paths (\\Server\Share) natively
		return filepath.Join(winPlacering, winFilnavn), nil
	}

	// 2. Define the translation rules
	winPrefix := `\\MOPSRV01\AdvoPro`
	linuxMount := "/mnt/advopro"

	// 3. Translate the path
	// Remove the Windows server prefix
	relPath := strings.TrimPrefix(winPlacering, winPrefix)

	// Convert Windows backslashes (\) to Linux forward slashes (/)
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// 4. Combine into a final Linux path
	finalPath := filepath.Join(linuxMount, relPath, winFilnavn)

	// Optional: Verify file exists on disk
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist at path: %s", finalPath)
	}

	return finalPath, nil
}

func UploadDocument(srcPath string, sagsnr uint64, title string) error {
	// check if file exists
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("cant find file at %s: %w", srcPath, err)
	}

	// sagsnr = 430415 // jens advokado (test case)

	//title := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))

	res, err := ImportDocument(
		srcPath,
		title,
		sagsnr,
		0,     // empID -> default 185
		"",    // user  -> default AUTO_IMPORT
		"",    // destFolder -> auto-detect
		false, // dryRun
	)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("Uploaded document: dok_id=%d version_id=%d forsendelse_id=%d\n",
		res.DokID, res.VersionID, res.ForsendelseID)
	return nil
}
