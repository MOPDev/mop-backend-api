package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MOPDev/mop-backend-api/initializers"
	"github.com/MOPDev/mop-backend-api/internal/logger"
	"github.com/MOPDev/mop-backend-api/models"
	"github.com/hypersequent/zen"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ptr is a generic helper that returns a pointer to the value passed in
func ptr[T any](v T) *T {
	return &v
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage:
  %s automigrate
  %s resetpassword <id>
  %s fullreset
  %s zod

examples:
  %s resetpassword 123
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

// go run .\migrate\migrate.go automigrate
// go run .\migrate\migrate.go resetpassword <id>
// go run .\migrate\migrate.go fullreset

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToDB()
}

func main() {
	log.SetFlags(0) // cleaner log output

	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "automigrate":
		migrateTables()

	case "resetpassword":
		if len(os.Args) < 3 {
			log.Printf("missing id for resetpassword")
			usage()
			os.Exit(2)
		}

		// Parse with platform int size to avoid overflow on 32-bit builds.
		idU, err := strconv.ParseUint(os.Args[2], 10, strconv.IntSize)
		if err != nil {
			log.Fatalf("invalid id %q: %v", os.Args[2], err)
		}
		resetPassword(uint(idU))

	case "fullreset":
		fullreset()

	case "zod":
		zodMigrate()

	default:
		log.Printf("unknown command: %s", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func migrateTables() {
	err := initializers.DB.AutoMigrate(
		&models.User{},
		&models.Debitor{},
		&models.Visit{},
		&models.VisitResponse{},
		&models.VisitStatus{},
		&models.VisitStatusLog{},
		&models.VisitResponseImage{},
		&models.Asset{},
		&models.LoginAttempt{},
		&models.AuthAttempt{},
		&models.VisitType{},
		&models.ActivityLog{},
		&models.VisitLog{},
	)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	for _, s := range statuses {
		result := initializers.DB.
			Where(models.VisitStatus{Model: gorm.Model{ID: s.ID}}).
			Assign(models.VisitStatus{Text: s.Text}).
			FirstOrCreate(&s)
		if result.Error != nil {
			logger.Error(result.Error.Error())
		}
	}

	logger.Info("Migration went well")
}

func resetPassword(id uint) {
	logger.Infof("id: %d", id)
	if id == 1 {
		logger.Info("This is the ID of the root, and should not be changed. Try another user")
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("pass"), 14)
	var user models.User
	user.ID = uint(id)
	initializers.DB.First(&user)
	logger.Infof("name: %s", user.Name)
	logger.Info("confirm y/n: ")
	var resp string
	n, err := fmt.Scanf("%s", &resp)
	if err != nil || n != 1 {
		log.Fatalf("failed to read input: %v", err)
	}
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp != "y" && resp != "yes" {
		logger.Info("aborted")
		return
	}

	initializers.DB.Model(&user).Update("password", string(hashedPassword))
	logger.Info("the new password is 'pass'")
}

func zodMigrate() {
	// Local structs without gorm.Model — use *gorm.DeletedAt so zen
	// generates DeletedAtSchema.nullable() (backend sends null when not deleted).
	type UserWithoutVisits struct {
		ID        uint              `json:"ID"`
		CreatedAt time.Time         `json:"CreatedAt"`
		UpdatedAt time.Time         `json:"UpdatedAt"`
		DeletedAt *gorm.DeletedAt   `json:"DeletedAt"`
		Initials  string            `json:"initials"`
		Name      string            `json:"name"`
		Username  string            `json:"username"`
		Rights    models.UserRights `json:"rights"`
		Email     string            `json:"email"`
		Phone     string            `json:"phone"`
		// NO Visits, NO Password
	}

	type VisitWithoutUserOrDebitors struct {
		ID        uint            `json:"ID"`
		CreatedAt time.Time       `json:"CreatedAt"`
		UpdatedAt time.Time       `json:"UpdatedAt"`
		DeletedAt *gorm.DeletedAt `json:"DeletedAt"`
		UserID    uint            `json:"user_id"`
		// NO User field - just keep UserID
		Address             string                  `json:"address"`
		Latitude            string                  `json:"latitude"`
		Longitude           string                  `json:"longitude"`
		Notes               string                  `json:"notes"`
		Sagsnr              uint                    `json:"sagsnr"`
		Stopnr              uint                    `json:"stop_nr"`
		VisitDate           time.Time               `json:"visit_date"`
		VisitTime           string                  `json:"visit_time"`
		VisitInterval       string                  `json:"visit_interval"`
		Visited             bool                    `json:"visited"`
		StatusID            uint                    `json:"status_id"`
		Status              models.VisitStatus      `json:"status"`
		VisitResponse       *models.VisitResponse   `json:"visit_response"`
		VisitStatusLogs     []models.VisitStatusLog `json:"visit_status_logs"`
		TypeID              uint                    `json:"type_id"`
		Type                models.VisitType        `json:"type"`
		AdvoproStatus       uint                    `json:"advopro__status"`
		AdvoproStatusText   string                  `json:"advopro_status_text"`
		AdvoproDeadlineDate string                  `json:"advopro_deadline_date"`
		AdvoproKlient       string                  `json:"advopro_klient"`
		GroupId             *uint                   `json:"group_id"`
		Cancelled           *bool                   `json:"cancelled"`
		// NO Debitors
	}

	type DebitorWithoutVisits struct {
		ID               uint            `json:"ID"`
		CreatedAt        time.Time       `json:"CreatedAt"`
		UpdatedAt        time.Time       `json:"UpdatedAt"`
		DeletedAt        *gorm.DeletedAt `json:"DeletedAt"`
		Name             string          `json:"name"`
		Phone            string          `json:"phone"`
		PhoneWork        string          `json:"phone_work"`
		Email            string          `json:"email"`
		Gender           models.Gender   `json:"gender"`
		Birthday         time.Time       `json:"birthday"`
		AdvoproDebitorId int             `json:"Advopro_debitor_id"`
		Risk             models.Risk     `json:"risk"`
		SSN              string          `json:"ssn"`
		Iscompany        bool            `json:"is_company"`
		Notes            string          `json:"notes"`
		// NO Visits
	}

	for _, s := range []any{UserWithoutVisits{}, VisitWithoutUserOrDebitors{}, DebitorWithoutVisits{}} {
		out := zen.StructToZodSchema(s)
		// Backend sends null for gorm.DeletedAt when not soft-deleted
		out = strings.ReplaceAll(out, "DeletedAt: DeletedAtSchema,", "DeletedAt: DeletedAtSchema.nullable(),")
		fmt.Println(out)
	}
}

func fullreset() {
	initializers.DB.Exec("PRAGMA foreign_keys = OFF;")
	initializers.DB.Exec("DROP TABLE IF EXISTS users;")

	// the many2many connection with debitor and visits need a connection table
	initializers.DB.Exec("DROP TABLE IF EXISTS visit_debitors;")
	initializers.DB.Exec("DROP TABLE IF EXISTS debitors;")
	initializers.DB.Exec("DROP TABLE IF EXISTS visits;")
	initializers.DB.Exec("DROP TABLE IF EXISTS visit_statuses;")
	initializers.DB.Exec("DROP TABLE IF EXISTS visit_status_logs;")

	initializers.DB.Exec("DROP TABLE IF EXISTS login_attempts;")
	initializers.DB.Exec("DROP TABLE IF EXISTS auth_attempt;")

	initializers.DB.Exec("DROP TABLE IF EXISTS visit_responses;")
	initializers.DB.Exec("DROP TABLE IF EXISTS visit_response_images;")

	initializers.DB.Exec("DROP TABLE IF EXISTS visit_types;")

	initializers.DB.Exec("PRAGMA foreign_keys = ON;")

	initializers.DB.AutoMigrate(
		&models.User{},
		&models.Debitor{},
		&models.Visit{},
		&models.VisitResponse{},
		&models.VisitStatus{},
		&models.VisitStatusLog{},
		&models.VisitResponseImage{},
		&models.Asset{},
		&models.LoginAttempt{},
		&models.AuthAttempt{},
		&models.VisitType{},
		&models.ActivityLog{},
	)

	for _, s := range statuses {
		initializers.DB.
			Where(models.VisitStatus{Model: gorm.Model{ID: s.ID}}).
			Assign(models.VisitStatus{Text: s.Text}).
			FirstOrCreate(&s)
	}

	// init some visit types
	for _, s := range visitTypes {
		initializers.DB.FirstOrCreate(&s)
	}

	//Hash the password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 14)
	hashedPassword1, _ := bcrypt.GenerateFromPassword([]byte(user1.Password), 14)

	user.Password = string(hashedPassword)
	user1.Password = string(hashedPassword1)
	//create some users
	initializers.DB.Create(&root)
	initializers.DB.Create(&user)  // Save the user to the database
	initializers.DB.Create(&user1) // Save the user to the database

	// init some visit types
	for _, s := range visitTypes {
		initializers.DB.Create(&s)
	}

	//create debitors
	initializers.DB.Create(&db1)
	initializers.DB.Create(&db2)
	initializers.DB.Create(&db3)

	// create some visits to the debitors
	visit1.UserID = user.ID
	visit1.TypeID = visitTypes[0].ID
	visit1.Debitors = []models.Debitor{db1, db3}

	visit2.UserID = user.ID
	visit2.TypeID = visitTypes[1].ID
	visit2.Debitors = []models.Debitor{db2, db3}

	initializers.DB.Create(&visit1) // Save the visit to the database
	initializers.DB.Create(&visit2) // Save the visit to the database

	//create some responses to the visits
	/*
		visitResponse1.VisitID = visit1.ID
		visitResponse2.VisitID = visit2.ID
		initializers.DB.Create(&visitResponse1) // Save the visit response to the database
		initializers.DB.Create(&visitResponse2) // Save the visit response to the database
	*/

	//create some visits to the debitors
	visit3.UserID = user.ID
	visit3.TypeID = visitTypes[2].ID // type3.ID

	visit4.UserID = user.ID
	visit4.TypeID = visitTypes[3].ID //4

	visit5.UserID = user.ID
	visit5.TypeID = visitTypes[0].ID

	visit6.UserID = user.ID
	visit6.TypeID = visitTypes[1].ID // 2

	// add debitors to the visits
	visit3.Debitors = []models.Debitor{db1}
	visit4.Debitors = []models.Debitor{db2}
	visit5.Debitors = []models.Debitor{db3}
	visit6.Debitors = []models.Debitor{db1, db2}

	initializers.DB.Create(&visit3) // Save the visit to the database
	initializers.DB.Create(&visit4) // Save the visit to the database
	initializers.DB.Create(&visit5) // Save the visit to the database
	initializers.DB.Create(&visit6)

	/*
		visitResponse3.VisitID = visit3.ID
		visitResponse4.VisitID = visit4.ID
		initializers.DB.Create(&visitResponse3) // Save the visit response to the database
		initializers.DB.Create(&visitResponse4) // Save the visit response to the database
	*/

}

// placeholder information
var statuses = []models.VisitStatus{
	{Model: gorm.Model{ID: 1}, Text: "Not planned"},
	{Model: gorm.Model{ID: 2}, Text: "planned"},
	{Model: gorm.Model{ID: 3}, Text: "ready to visit"},
	{Model: gorm.Model{ID: 4}, Text: "to review"},
	{Model: gorm.Model{ID: 5}, Text: "exported"},
	{Model: gorm.Model{ID: 6}, Text: "pending completion"},
}

var visitTypes = []models.VisitType{
	{
		Text: "købekontrakt",
		Description: `En købekontrakt betyder at bilen ejes af debitor.
    Debitor skylder dog penge som han har brugt på bilen.
    Det er derfor vigtigt at vide hvordan bilen har det og om han har solgt eller andet`,
	},
	{
		Text: "leasing",
		Description: `En Leasing aftale betyder at bilen ikke ejes af debitor.
    Det betyder at man godt bare må tage bilen.
    Det er derfor vigtigt at vide hvordan bilen har det, og evt. hvor den er nu`,
	},
	{
		Text: "blanco",
		Description: `En blanco aftale betyder at man bare skal have penge ud af debitor.
    Det betyder at det er vigtigt at finde ud af hvor rig personen er, og hvor godt de kan betale en gæld tilbage`,
	},
	{
		Text:        "brev",
		Description: `Dette betyder at vi bare gerne vil aflevere et brev, evt. tage et billede af postkassen eller mangel derpå`,
	},
}

var root = models.User{
	Username: "root",
	Password: "d",
	Email:    "",
	Phone:    "",
}
var user = models.User{
	Initials: "mkk",
	Name:     "Markus Kjeldsen",
	Username: "mkk",
	Password: "pass",
	Email:    "Markus@kjeldsen.dk",
	Phone:    "42480991",
	Rights:   models.RightsDeveloper,
}
var user1 = models.User{
	Initials: "po",
	Name:     "Patrick Olsen",
	Username: "po_admin",
	Password: "pass",
	Email:    "Patrick@olsen.dk",
	Phone:    "21193038",
	Rights:   models.RightsAdmin,
}
var db1 = models.Debitor{
	Name:             "Cindy Lou",
	Phone:            "1234567890",
	PhoneWork:        "0987654321",
	Email:            "cindy@example.com",
	Gender:           models.Female,
	Birthday:         time.Date(1995, 5, 10, 0, 0, 0, 0, time.UTC),
	AdvoproDebitorId: 45,
	Risk:             models.LowRisk,
	SSN:              "140599-0013",
	Notes:            "Friendly and prompt payer.",
}
var db2 = models.Debitor{
	Name:             "Ebenezer Scrooge",
	Phone:            "2223334444",
	PhoneWork:        "3334445555",
	Email:            "scrooge@example.com",
	Gender:           models.Male,
	Birthday:         time.Date(1970, 12, 25, 0, 0, 0, 0, time.UTC),
	AdvoproDebitorId: 13,
	Risk:             models.HighRisk,
	SSN:              "020202-3213",
	Notes:            "High risk, late payer.",
}
var db3 = models.Debitor{
	Name:             "Grinch",
	Phone:            "5556667777",
	PhoneWork:        "8889990000",
	Email:            "grinch@example.com",
	Gender:           models.Other,
	Birthday:         time.Date(1982, 6, 1, 0, 0, 0, 0, time.UTC),
	AdvoproDebitorId: 99,
	Risk:             models.MediumRisk,
	SSN:              "140205-0013",
	Notes:            "Sometimes cooperates.",
}
var visit1 = models.Visit{
	UserID:        user.ID,
	Address:       "123 Main St",
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "-122.4194",
	Notes:         "First visit",
	Sagsnr:        1,
	VisitDate:     time.Now(),
	VisitTime:     "10:00 AM",
	Debitors:      []models.Debitor{db1, db2},
	StatusID:      3,
	GroupId:       nil,
}
var visit2 = models.Visit{
	UserID:        user.ID,
	Address:       "123 Main St",
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "-122.4194",
	Notes:         "First visit",
	Sagsnr:        2,
	VisitDate:     time.Now(),
	VisitTime:     "12:00 AM",
	Debitors:      []models.Debitor{db2, db3},
	StatusID:      3,
	GroupId:       nil,
}
var visit3 = models.Visit{
	UserID:        user.ID,
	Address:       "1337 Main St",
	Debitors:      []models.Debitor{db3},
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "2.4194",
	Notes:         "First visit",
	Sagsnr:        3,
	VisitDate:     time.Now(),
	VisitTime:     "12:00 AM",
	StatusID:      3,
	GroupId:       ptr(uint(1)),
}
var visit4 = models.Visit{
	UserID:        user.ID,
	Address:       "1337 Main St",
	Debitors:      []models.Debitor{db2},
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "2.4194",
	Notes:         "First visit",
	Sagsnr:        4,
	VisitDate:     time.Now().AddDate(0, 0, 0),
	VisitTime:     "18:00 AM",
	StatusID:      3,
	GroupId:       ptr(uint(1)),
}
var visit5 = models.Visit{
	UserID:        user.ID,
	Address:       "1337 Main St",
	Debitors:      []models.Debitor{db1},
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "2.4194",
	Notes:         "First visit",
	Sagsnr:        4,
	VisitDate:     time.Now().AddDate(0, 0, 0),
	VisitTime:     "18:00 AM",
	Visited:       true,
	StatusID:      3,
	GroupId:       ptr(uint(1)),
}

var visit6 = models.Visit{
	UserID:        user.ID,
	Address:       "1337 Main St",
	Debitors:      []models.Debitor{db1},
	VisitInterval: "10:00-13:00",
	Latitude:      "37.7749",
	Longitude:     "2.4194",
	Notes:         "First visit",
	Sagsnr:        4,
	VisitDate:     time.Now().AddDate(0, 0, 1),
	VisitTime:     "18:00 AM",
	Visited:       false,
	StatusID:      3,
	GroupId:       ptr(uint(1)),
}

var visitResponse1 = models.VisitResponse{
	VisitID: visit2.ID,
	ActDate: time.Now(),
	ActTime: "10:00 AM",
	ActLat:  "37.7749",
	ActLong: "-122.4194",

	PosAccuracy: "0.001",
	Duration:    time.Duration(time.Duration.Minutes(3)),

	// Response data
	Contact: models.ContactQuestions{
		MailboxName:   "",
		DebitorMet:    ptr(true),
		OtherMet:      nil,
		OtherTitle:    "",
		WorkerMet:     nil,
		WorkerTitle:   "",
		CorrectedTlf:  "",
		CorrectedMail: "",
	},

	Payment: models.PaymentQuestions{
		ReceivedPayment: nil,
		PaymentAmount:   ptr(models.Money(200)), // 2 kr
		PaymentMethod:   "kontant",
	},

	Asset: models.AssetQuestions{
		AssetSeen:                ptr(true),
		AssetAccessible:          ptr(false),
		AssetStatus:              "",
		AssetStatusNote:          "",
		AssetCleanliness:         "",
		AssetCleanlinessNote:     "",
		AssetConfirmedOwner:      nil,
		AssetKeysDelivered:       nil,
		SFSigned:                 nil,
		OdometerKm:               nil,
		ContractType:             "",
		IsSeized:                 nil,
		HandoverStrategy:         "",
		HandoverStrategyNote:     "",
		TransportProvider:        "",
		FinalVehicleLocation:     "",
		FinalVehicleLocationNote: "",
	},

	Monetary: models.MonetaryQuestions{
		CivilStatus:    "Cohabiting",
		ChildrenOver18: nil,
		HasWork:        nil,
		Position:       "",
		NetSalaryMin:   ptr(models.Money(200)),
		NetSalaryMax:   ptr(models.Money(400)),

		IncomePaymentMin: ptr(models.Money(200)),
		IncomePaymentMax: ptr(models.Money(400)),

		MonthlyDisposableMin: ptr(models.Money(200)),
		MonthlyDisposableMax: ptr(models.Money(400)),

		DebtAmountPaid: ptr(models.Money(200)),
	},
	Property: models.PropertyQuestions{
		PropertyType:      ptr(models.PropertyFreestandingHouse),
		OvergrownGarden:   ptr(true),
		MailboxFull:       nil,
		BrokenWindows:     nil,
		AbandonedVehicles: nil,
		TrashOverflown:    nil,
		ForsaleSign:       nil,
	},

	Comments: "Meget grimt hus, det er nok forfaldendt",
}

var visitResponse2 = models.VisitResponse{
	VisitID: visit1.ID,
	ActDate: time.Now(),
	ActTime: "10:00 AM",
	ActLat:  "37.7749",
	ActLong: "-122.4194",

	// Response data
	Contact: models.ContactQuestions{
		MailboxName:   "",
		DebitorMet:    ptr(true), // Migrated from DebitorIsHome: ptr(true)
		OtherMet:      nil,
		OtherTitle:    "",
		WorkerMet:     nil,
		WorkerTitle:   "",
		CorrectedTlf:  "",
		CorrectedMail: "",
	},

	Payment: models.PaymentQuestions{
		ReceivedPayment: nil,
		PaymentAmount:   nil,
		PaymentMethod:   "",
	},

	Asset: models.AssetQuestions{
		AssetSeen:                ptr(true), // Migrated from AssetAtAddress: ptr(true)
		AssetAccessible:          ptr(false),
		AssetStatus:              "", // Migrated from AssetDamaged: ptr(false)
		AssetStatusNote:          "",
		AssetCleanliness:         "",
		AssetCleanlinessNote:     "",
		AssetConfirmedOwner:      nil,
		AssetKeysDelivered:       nil,
		SFSigned:                 nil,
		OdometerKm:               nil,
		ContractType:             "",
		IsSeized:                 nil,
		HandoverStrategy:         "",
		HandoverStrategyNote:     "",
		TransportProvider:        "",
		FinalVehicleLocation:     "",
		FinalVehicleLocationNote: "",
	},

	Monetary: models.MonetaryQuestions{
		CivilStatus:    string(models.Married), // Migrated from CivilStatus: ptr(models.Married)
		ChildrenOver18: ptr(uint(10)),
		HasWork:        ptr(true),
		Position:       "CEO",
		NetSalaryMin:   ptr(models.Money(5000000)), // 50.000 kr in cents
		NetSalaryMax:   ptr(models.Money(5000000)), // 50.000 kr in cents

		IncomePaymentMin:     nil,
		IncomePaymentMax:     nil,
		MonthlyDisposableMin: nil,
		MonthlyDisposableMax: nil,
		// Custom Case/Debt details migrated into MonetaryQuestions

		DebtAmountPaid: ptr(models.Money(100000000)), // 1.000.000 kr in cents,

	},

	Property: models.PropertyQuestions{
		PropertyType:      ptr(models.PropertyFreestandingHouse),
		OvergrownGarden:   nil,
		MailboxFull:       nil,
		BrokenWindows:     nil,
		AbandonedVehicles: nil,
		TrashOverflown:    nil,
		ForsaleSign:       nil,
	},

	Comments: "Meget flot hus, han er tydeligvis rig",
}

var visitResponse3 = models.VisitResponse{
	VisitID: visit1.ID,
	ActDate: time.Now(),
	ActTime: "10:00 AM",
	ActLat:  "37.7749",
	ActLong: "-122.4194",

	// Response data
	Contact: models.ContactQuestions{
		MailboxName:   "",
		DebitorMet:    ptr(true), // Migrated from DebitorIsHome: ptr(true)
		OtherMet:      nil,
		OtherTitle:    "",
		WorkerMet:     nil,
		WorkerTitle:   "",
		CorrectedTlf:  "",
		CorrectedMail: "",
	},

	Payment: models.PaymentQuestions{
		ReceivedPayment: nil,
		PaymentAmount:   nil,
		PaymentMethod:   "",
	},

	Asset: models.AssetQuestions{
		AssetSeen:                ptr(true), // Migrated from AssetAtAddress: ptr(true)
		AssetAccessible:          ptr(false),
		AssetStatus:              "", // Migrated from AssetDamaged: ptr(false)
		AssetStatusNote:          "",
		AssetCleanliness:         "",
		AssetCleanlinessNote:     "",
		AssetConfirmedOwner:      nil,
		AssetKeysDelivered:       nil,
		SFSigned:                 nil,
		OdometerKm:               nil,
		ContractType:             "",
		IsSeized:                 nil,
		HandoverStrategy:         "",
		HandoverStrategyNote:     "",
		TransportProvider:        "",
		FinalVehicleLocation:     "",
		FinalVehicleLocationNote: "",
	},

	Monetary: models.MonetaryQuestions{
		CivilStatus:    string(models.Married), // Migrated from CivilStatus: ptr(models.Married)
		ChildrenOver18: ptr(uint(0)),
		HasWork:        ptr(true),
		Position:       "janitor",
		NetSalaryMin:   ptr(models.Money(5000000)), // 50.000 kr in cents
		NetSalaryMax:   ptr(models.Money(5000000)), // 50.000 kr in cents

		IncomePaymentMin:     nil,
		IncomePaymentMax:     nil,
		MonthlyDisposableMin: nil,
		MonthlyDisposableMax: nil,
		DebtAmountPaid:       ptr(models.Money(100000000)), // 1.000.000 kr in cents,
	},

	Property: models.PropertyQuestions{
		PropertyType:      ptr(models.PropertyApartment),
		OvergrownGarden:   nil,
		MailboxFull:       nil,
		BrokenWindows:     nil,
		AbandonedVehicles: nil,
		TrashOverflown:    nil,
		ForsaleSign:       nil,
	},

	Comments: "Meget flot hus, han er tydeligvis rig",
}

var visitResponse4 = models.VisitResponse{
	VisitID: visit1.ID,
	ActDate: time.Now(),
	ActTime: "10:00 AM",
	ActLat:  "37.7749",
	ActLong: "-122.4194",

	// Response data
	Contact: models.ContactQuestions{
		MailboxName:   "",
		DebitorMet:    ptr(false), // Migrated from DebitorIsHome: ptr(false)
		OtherMet:      nil,
		OtherTitle:    "",
		WorkerMet:     nil,
		WorkerTitle:   "",
		CorrectedTlf:  "",
		CorrectedMail: "",
	},

	Payment: models.PaymentQuestions{
		ReceivedPayment: nil,
		PaymentAmount:   nil,
		PaymentMethod:   "",
	},

	Asset: models.AssetQuestions{
		AssetSeen:                ptr(false), // Migrated from AssetAtAddress: ptr(false)
		AssetAccessible:          ptr(false),
		AssetStatus:              "", // Migrated from AssetDamaged: ptr(false)
		AssetStatusNote:          "",
		AssetCleanliness:         "",
		AssetCleanlinessNote:     "",
		AssetConfirmedOwner:      nil,
		AssetKeysDelivered:       nil,
		SFSigned:                 nil,
		OdometerKm:               nil,
		ContractType:             "",
		IsSeized:                 nil,
		HandoverStrategy:         "",
		HandoverStrategyNote:     "",
		TransportProvider:        "",
		FinalVehicleLocation:     "",
		FinalVehicleLocationNote: "",
	},

	Monetary: models.MonetaryQuestions{
		CivilStatus:    string(models.Single), // Migrated from CivilStatus: ptr(models.Single)
		ChildrenOver18: ptr(uint(10)),
		HasWork:        ptr(true),
		Position:       "CEO",
		NetSalaryMin:   ptr(models.Money(5000000)), // 50.000 kr in cents
		NetSalaryMax:   ptr(models.Money(5000000)), // 50.000 kr in cents

		IncomePaymentMin:     nil,
		IncomePaymentMax:     nil,
		MonthlyDisposableMin: nil,
		MonthlyDisposableMax: nil,
		DebtAmountPaid:       ptr(models.Money(100000000)),
	},

	Property: models.PropertyQuestions{
		PropertyType:      ptr(models.PropertySummerHouse),
		OvergrownGarden:   nil,
		MailboxFull:       nil,
		BrokenWindows:     nil,
		AbandonedVehicles: nil,
		TrashOverflown:    nil,
		ForsaleSign:       nil,
	},

	Comments: "Meget flot hus, han er tydeligvis rig",
}
