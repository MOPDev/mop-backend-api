package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PropertyType string

const (
	PropertyFreestandingHouse PropertyType = "Fritlæggende hus"
	PropertyTownhouse         PropertyType = "Byhus"
	PropertyTerracedHouse     PropertyType = "Rækkehus"
	PropertySummerHouse       PropertyType = "Sommerhus"
	PropertyGardenColony      PropertyType = "Kolonihave"
	PropertyApartment         PropertyType = "Lejlighed"
)

type MaintenanceStatus string

const (
	WellMaintained MaintenanceStatus = "Velholdt"
	Deteriorated   MaintenanceStatus = "Forfalden"
)

type UserRights string

const (
	RightsAdmin        UserRights = "admin"
	RightsDeveloper    UserRights = "developer"
	RightsOfficeWorker UserRights = "office"
	RightsAuditor      UserRights = "auditor"
	RightsUser         UserRights = "user"
	RightsNone         UserRights = "none"
)

type CivilStatus string

const (
	Married    CivilStatus = "Married"
	Single     CivilStatus = "Single"
	Cohabiting CivilStatus = "Cohabiting"
)

type Gender string

const (
	Male   Gender = "Male"
	Female Gender = "Female"
	Other  Gender = "Other"
)

type Risk string

const (
	LowRisk    Risk = "Low"
	MediumRisk Risk = "Medium"
	HighRisk   Risk = "High"
)

// models/models.go
type User struct {
	gorm.Model
	Initials string     `json:"initials" gorm:"not null,default:''"`
	Name     string     `json:"name" binding:"required" gorm:"not null;uniqueIndex:ux_users_name_active,where:deleted_at IS NULL"`
	Username string     `json:"username" binding:"required" gorm:"not null;uniqueIndex:ux_users_username_active,where:deleted_at IS NULL"`
	Password string     `json:"-" binding:"required" gorm:"not null"`
	Rights   UserRights `json:"rights" gorm:"default:user"`
	Email    string     `json:"email"`
	Phone    string     `json:"phone"`
	Visits   []Visit    `json:"visits" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type AuthAttempt struct {
	gorm.Model
	IP            string `gorm:"size:45;not null"`
	FailureReason string `json:"failure_reason"`
}

type LoginAttempt struct {
	ID            uint      `gorm:"primarykey"`
	CreatedAt     time.Time `gorm:"index:idx_created"`
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	UserID        uint           `json:"user_id" gorm:"not null,index:idx_user_created,priority:1"`
	Username      string         `gorm:"not null;index:idx_username_created,priority:1"`
	IP            string         `gorm:"size:45;not null;index:idx_ip_created,priority:1"`
	Successful    bool           `gorm:"not null"`
	FailureReason string
}

type Debitor struct {
	gorm.Model
	Name             string    `json:"name" gorm:"not null"`
	Phone            string    `json:"phone"`
	PhoneWork        string    `json:"phone_work"`
	Email            string    `json:"email"`
	Gender           Gender    `json:"gender" gorm:"not null"` // Male, Female, Other
	Birthday         time.Time `json:"birthday"`
	AdvoproDebitorId int       `json:"Advopro_debitor_id"`
	Risk             Risk      `json:"risk"` // Low, Medium, High
	SSN              string    `json:"ssn"`
	Iscompany        bool      `json:"is_company"`

	Notes  string  `json:"notes"`
	Visits []Visit `gorm:"many2many:visit_debitors;"`
}

// skal jeg lave en tabel som hedder sager
// også have debitorer knyttet til en sag?
// eller vil vi gerne have mulighed for at kunne besøge kun en debitor
// skal jeg så have forlobid med.
type VisitStatus struct {
	gorm.Model
	Text        string `json:"text"`
	Description string `json:"description"`
}

type VisitStatusLog struct {
	gorm.Model
	VisitID     uint      `json:"visit_id" gorm:"not null"`
	OldStatusID uint      `json:"old_status_id"`
	NewStatusID uint      `json:"new_status_id"`
	ChangedAt   time.Time `json:"changed_at" gorm:"autoCreateTime"`
	ChangedByID uint      `json:"changed_by_id"` // Optionally, reference User.ID
}

type VisitLog struct {
	gorm.Model
	VisitID     uint      `json:"visit_id" gorm:"not null"`
	PreviousVal string    `json:"old_val"`
	NewVal      string    `json:"new_val"`
	ValType     string    `json:"val_type"`
	ChangedAt   time.Time `json:"changed_at" gorm:"autoCreateTime"`
	ChangedByID uint      `json:"changed_by_id"`
}

type VisitType struct {
	gorm.Model
	Text        string `json:"text"`
	Description string `json:"description"`
}

type Visit struct {
	gorm.Model
	UserID          uint             `json:"user_id"`
	User            User             `json:"user"`
	Address         string           `json:"address"`
	Latitude        string           `json:"latitude"`
	Longitude       string           `json:"longitude"`
	Notes           string           `json:"notes"`
	Sagsnr          uint             `json:"sagsnr"`
	Stopnr          uint             `json:"stop_nr"`
	VisitDate       time.Time        `json:"visit_date" gorm:"type:date"`
	VisitTime       string           `json:"visit_time"`
	VisitInterval   string           `json:"visit_interval"`
	Visited         bool             `json:"visited"`
	StatusID        uint             `json:"status_id" gorm:"not null;default:1"` // <-- Add this
	Status          VisitStatus      `json:"status" gorm:"foreignKey:StatusID"`   // <-- Keep this for relation
	Debitors        []Debitor        `json:"debitors" gorm:"many2many:visit_debitors;"`
	VisitResponse   *VisitResponse   `json:"visit_response" gorm:"foreignKey:VisitID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	VisitStatusLogs []VisitStatusLog `json:"visit_status_logs" gorm:"foreignKey:VisitID"`
	TypeID          uint             `json:"type_id"`
	Type            VisitType        `json:"type" gorm:"foreignKey:TypeID"`
	// these are for data in the excel sheet
	AdvoproStatus       uint   `json:"advopro__status"`
	AdvoproStatusText   string `json:"advopro_status_text"`
	AdvoproDeadlineDate string `json:"advopro_deadline_date"`
	AdvoproKlient       string `json:"advopro_klient"`
	// a new type of ID for grouping
	GroupId *uint `json:"group_id"`
}

type VisitResponse struct {
	gorm.Model
	VisitID uint `json:"visit_id" binding:"required" gorm:"not null;unique"`

	// actual data
	ActDate     time.Time     `json:"actual_date" binding:"required"`
	ActTime     string        `json:"actual_time" binding:"required"`
	ActLat      string        `json:"actual_latitude" binding:"required"`
	ActLong     string        `json:"actual_longitude" binding:"required"`
	PosAccuracy string        `json:"pos_accuracy" binding:"required"`
	Duration    time.Duration `json:"duration"`

	// nested questions
	Contact  ContactQuestions  `json:"contact" gorm:"embedded;embeddedPrefix:contact_"`
	Payment  PaymentQuestions  `json:"payment" gorm:"embedded;embeddedPrefix:payment_"`
	Asset    AssetQuestions    `json:"asset" gorm:"embedded;embeddedPrefix:asset_"`
	Property PropertyQuestions `json:"property" gorm:"embedded;embeddedPrefix:property_"`
	Monetary MonetaryQuestions `json:"monetary" gorm:"embedded;embeddedPrefix:monetary_"`

	// one to many
	OtherAssets []Asset              `json:"other_assets" gorm:"foreignKey:VisitResponseID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Images      []VisitResponseImage `json:"images" gorm:"foreignKey:VisitResponseID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Comments string `json:"comments"` // free text field for comments, Property and such
}

type ContactQuestions struct {
	// quick option to choose the expected name
	MailboxName string `json:"mailbox_name"`
	// letter left to prove auditor visited when nobody answered
	LetterDelivered *bool `json:"letter_delivered"`

	// hvis CPR og kobekontrakt
	DebitorMet *bool `json:"debitor_met"` // Pointer so it can be null, not just false
	OtherMet   *bool `json:"other_met"`
	// including but not limited to "ægtefælle", "Partner", "kæreste", "Barn", "Nabo"
	OtherTitle string `json:"other_title"`

	// hvis CVR og kobekontrakt
	WorkerMet *bool `json:"worker_met"`
	// including but not limited to "Direktør", "Håndværker", "Lønmodtager/arbejder", "receptionist"
	WorkerTitle string `json:"worker_title"`

	CorrectedTlf  string `json:"corrected_tlf"`
	CorrectedMail string `json:"corrected_mail"`
}

type PaymentQuestions struct {
	// betaling af restancen, enten delvist eller fuldt via bankoverførsel eller i kontanter
	ReceivedPayment *bool
	PaymentAmount   *Money // 150,95 => 15095, always in ore
	PaymentMethod   string // including but not limited to "Kontant", "Bankoverførsel", etc.
}

type AssetQuestions struct {
	// asset meaning the car, truck, boat, RV, etc.
	// if located then prompt to take picture
	AssetSeen *bool `json:"asset_seen"`
	// can it be taken?
	AssetAccessible *bool `json:"asset_accessible"`
	// "Perfect", "minor scratches", "beaten up", "Totaled"
	AssetStatus string `json:"asset_status"`
	// "wheels missing" etc.
	AssetStatusNote string `json:"asset_status_note"`
	// "lige ud af vaskehal", "generelt ren", "jord og mudder/beskidt"
	AssetCleanliness string `json:"asset_cleanliness"`
	// burned seats
	AssetCleanlinessNote string `json:"asset_cleanliness_note"`

	// if debitor or other is there then ask
	AssetConfirmedOwner *bool `json:"asset_confirmed_owner"`
	// only if owner is there
	AssetKeysDelivered *bool `json:"asset_keys_delivered"`
	SFSigned           *bool `json:"sf_signed"` // Salgs Fuldmagt

	// only if IsSeized == true
	OdometerKm *uint `json:"odometer_km"`

	// if keys are given what is the take home strategy,
	// when kobekontrakt debitor owns the car
	// when blanco they can both own and not own
	// when leasing they dont own so it will be taken, if we can, unless they pay

	// if they dont own
	// Asset left for later pickup (tomorrow or just later)
	// Asset is getting picked up by grube (sjælland) or ???? (jylland)
	// auditor moves it themselves to local dealership for later pickup
	// most important, where is the vehicle after it has been moved, the result of the visit
	// --- Contract Type ---
	// Options: "Købekontrakt" (Reservation of title / Ejendomsforbehold)
	//          "Blanco" (Unsecured)
	//          "Leasing" (Creditor owns the asset)
	ContractType string `json:"contract_type"`

	// --- Seizure and Strategy ---
	// Has received the car keys / vehicle taken
	IsSeized *bool `json:"is_seized" gorm:"index"`

	// Take Home / Handover Strategy (Mandatory if IsSeized is true)
	// Options: "Auditor Drive-Away", "Immediate Towing", "Leave On Site Locked", "Other"
	HandoverStrategy string `json:"handover_strategy"`
	// if other then describe, hvad er aftalen? og med hvem
	HandoverStrategyNote string `json:"handover_strategy_note"`

	// Towing / Transport Provider -- display the contact information to the auditor.
	// Options: "Grube (Sjælland)", "Jens (Jylland)", "Auditor", "Other" (skriv i final loca note), "None"
	TransportProvider string `json:"transport_provider"`

	// --- THE RESULT OF THE VISIT (The ultimate source of truth) ---
	// Options:
	// 			"Towing Storage Yard (Sjælland)"
	//          "Towing Storage Yard (Jylland)"
	//          "Local Dealership" (Often for leasing returns)
	//          "At Debtor Address" (If left behind locked on-site)
	//          "Other"
	FinalVehicleLocation string `json:"final_vehicle_location" gorm:"index"`

	// Crucial free-text note: Exact GPS, address, bay number, or key locker location
	// e.g., "Parked in bay 42, keys dropped in Grube's night box" or "Left on driveway, locked, pending Jens pickup tomorrow"
	FinalVehicleLocationNote string `json:"final_vehicle_location_note" gorm:"type:text"`
}

type Asset struct {
	gorm.Model
	VisitResponseID uint   `json:"visit_response_id"`
	Regnr           string `json:"regnr"`
	ImagePath       string `json:"image_path"`
	OriginalName    string `json:"original_name"`
}

type PropertyQuestions struct {
	PropertyType *PropertyType `json:"property_type"`
	// if below are not descriptive enough, write in the comments
	OvergrownGarden   *bool `json:"overgrown_garden"`
	MailboxFull       *bool `json:"mailbox_full"`
	BrokenWindows     *bool `json:"broken_windows"`
	AbandonedVehicles *bool `json:"abandoned_vehicles"`
	TrashOverflown    *bool `json:"trash_overflown"`
	ForsaleSign       *bool `json:"forsale_sign"`

	Note string `json:"note"`
}

type MonetaryQuestions struct {
	// standard answers are: "Gift", "enlig", "samboende", other
	CivilStatus string `json:"civil_status"`

	//children At home
	ChildrenOver18  *uint `json:"children_over_18"`
	ChildrenUnder18 *uint `json:"children_under_18"`
	//ChildSupport   *float32 `json:"child_support"`

	//work - rough amounts
	HasWork  *bool  `json:"has_work"`
	Position string `json:"position"`

	// Split NetSalary into Min and Max (using custom Money pointers)
	NetSalaryMin *Money `json:"net_salary_min" binding:"omitempty,ltefield=NetSalaryMax" gorm:"type:bigint"`
	NetSalaryMax *Money `json:"net_salary_max" binding:"omitempty,gtefield=NetSalaryMin" gorm:"type:bigint"`

	// Other income rough amounts - offentlige ydelser
	IncomePaymentMin *Money `json:"income_payment_min" binding:"omitempty,ltefield=IncomePaymentMax" gorm:"type:bigint"`
	IncomePaymentMax *Money `json:"income_payment_max" binding:"omitempty,gtefield=IncomePaymentMin" gorm:"type:bigint"`

	// Disposable Income Range
	MonthlyDisposableMin *Money `json:"monthly_disposable_min" binding:"omitempty,ltefield=MonthlyDisposableMax" gorm:"type:bigint"`
	MonthlyDisposableMax *Money `json:"monthly_disposable_max" binding:"omitempty,gtefield=MonthlyDisposableMin" gorm:"type:bigint"`

	// Other debt being paid per month roughly
	DebtAmountPaid *Money `json:"debt_amount_paid" gorm:"type:bigint"`
}

type VisitResponseImage struct {
	gorm.Model
	VisitResponseID uint   `json:"visit_response_id"`
	ImagePath       string `json:"image_path"`
	OriginalName    string `json:"original_name"`
}

type ActivityLog struct {
	gorm.Model
	ActingUserID uint           `json:"acting_user_id"`
	TargetID     uint           `json:"target_id"`
	TargetIDType string         `json:"target_id_type"`
	ActionType   string         `json:"action_type"`
	ColumnType   string         `json:"column_type"`
	PrevVal      datatypes.JSON `json:"prev_val"`
	CurrentVal   datatypes.JSON `json:"current_val"`
}
