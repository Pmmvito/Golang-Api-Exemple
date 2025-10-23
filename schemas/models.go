package schemas

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UUIDModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *UUIDModel) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type CategoryType string

const (
	CategoryTypeFixed    CategoryType = "fixa"
	CategoryTypeVariable CategoryType = "variavel"
)

type ExpenseOrigin string

const (
	ExpenseOriginManual ExpenseOrigin = "manual"
	ExpenseOriginOCR    ExpenseOrigin = "ocr"
	ExpenseOriginAI     ExpenseOrigin = "ia"
)

type Theme string

const (
	ThemeLight  Theme = "claro"
	ThemeDark   Theme = "escuro"
	ThemeSystem Theme = "sistema"
)

type TipType string

const (
	TipTypeSavings  TipType = "economia"
	TipTypePlanning TipType = "planejamento"
	TipTypeAlert    TipType = "alerta"
)

type MealDay string

const (
	MealDayMonday    MealDay = "seg"
	MealDayTuesday   MealDay = "ter"
	MealDayWednesday MealDay = "qua"
	MealDayThursday  MealDay = "qui"
	MealDayFriday    MealDay = "sex"
	MealDaySaturday  MealDay = "sab"
	MealDaySunday    MealDay = "dom"
)

type MealType string

const (
	MealTypeBreakfast MealType = "cafe"
	MealTypeLunch     MealType = "almoco"
	MealTypeDinner    MealType = "janta"
	MealTypeSnack     MealType = "lanche"
)

type SyncOrigin string

const (
	SyncOriginMobile SyncOrigin = "mobile"
	SyncOriginWeb    SyncOrigin = "web"
	SyncOriginBackup SyncOrigin = "backup"
)

type SyncStatus string

const (
	SyncStatusOK      SyncStatus = "ok"
	SyncStatusError   SyncStatus = "erro"
	SyncStatusPartial SyncStatus = "parcial"
)

type User struct {
	UUIDModel
	Name         string         `gorm:"size:120" json:"name"`
	Email        string         `gorm:"size:180;uniqueIndex" json:"email"`
	PasswordHash string         `gorm:"size:255" json:"-"`
	Active       bool           `gorm:"default:true" json:"active"`
	LastLogin    *time.Time     `json:"lastLogin,omitempty"`
	Categories   []Category     `gorm:"constraint:OnDelete:CASCADE;" json:"categories,omitempty"`
	Expenses     []Expense      `gorm:"constraint:OnDelete:CASCADE;" json:"expenses,omitempty"`
	Sessions     []Session      `gorm:"constraint:OnDelete:CASCADE;" json:"sessions,omitempty"`
	Tips         []GeneratedTip `gorm:"constraint:OnDelete:CASCADE;" json:"tips,omitempty"`
	MealPlans    []MealPlan     `gorm:"constraint:OnDelete:CASCADE;" json:"mealPlans,omitempty"`
	SyncJobs     []SyncJob      `gorm:"constraint:OnDelete:CASCADE;" json:"syncJobs,omitempty"`
	Config       *UserConfig    `gorm:"constraint:OnDelete:CASCADE;" json:"config,omitempty"`
}

type UserConfig struct {
	UserID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"userId"`
	Currency             string    `gorm:"size:3;default:'BRL'" json:"currency"`
	MonthlyLimit         float64   `gorm:"type:numeric(12,2);default:0" json:"monthlyLimit"`
	NotificationsEnabled bool      `gorm:"default:true" json:"notificationsEnabled"`
	Language             string    `gorm:"size:5;default:'pt-BR'" json:"language"`
	Theme                Theme     `gorm:"type:varchar(12);default:'sistema'" json:"theme"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	User                 *User     `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type Category struct {
	UUIDModel
	UserID   uuid.UUID    `gorm:"type:uuid;index" json:"userId"`
	Name     string       `gorm:"size:80" json:"name"`
	Icon     string       `gorm:"size:40" json:"icon"`
	ColorHex string       `gorm:"size:7" json:"colorHex"`
	Type     CategoryType `gorm:"type:varchar(12)" json:"type"`
	Order    int          `gorm:"default:0" json:"order"`
	Active   bool         `gorm:"default:true" json:"active"`
	User     *User        `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Expenses []Expense    `json:"expenses,omitempty"`
}

type Expense struct {
	UUIDModel
	UserID      uuid.UUID     `gorm:"type:uuid;index" json:"userId"`
	CategoryID  uuid.UUID     `gorm:"type:uuid;index" json:"categoryId"`
	Description string        `gorm:"size:200" json:"description"`
	Amount      float64       `gorm:"type:numeric(12,2)" json:"amount"`
	Date        time.Time     `gorm:"index" json:"date"`
	Recurring   bool          `gorm:"default:false" json:"recurring"`
	Origin      ExpenseOrigin `gorm:"type:varchar(10);default:'manual'" json:"origin"`
	Receipt     *Receipt      `json:"receipt,omitempty"`
	User        *User         `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Category    *Category     `gorm:"constraint:OnDelete:SET NULL" json:"category,omitempty"`
	Items       []ExpenseItem `gorm:"constraint:OnDelete:CASCADE;" json:"items"`
}

type Receipt struct {
	UUIDModel
	ExpenseID     uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"expenseId"`
	FilePath      string    `gorm:"size:255" json:"filePath"`
	ExtractedText string    `gorm:"type:text" json:"extractedText"`
	OcrConfidence float64   `gorm:"type:numeric(5,2)" json:"ocrConfidence"`
	Expense       *Expense  `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type GeneratedTip struct {
	UUIDModel
	UserID      uuid.UUID `gorm:"type:uuid;index" json:"userId"`
	Type        TipType   `gorm:"type:varchar(20)" json:"type"`
	Text        string    `gorm:"type:text" json:"text"`
	ModelSource string    `gorm:"size:80" json:"modelSource"`
	Relevance   int       `json:"relevance"`
	User        *User     `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type MealPlan struct {
	UUIDModel
	UserID        uuid.UUID  `gorm:"type:uuid;index" json:"userId"`
	IsoWeek       string     `gorm:"size:8" json:"isoWeek"`
	CalorieGoal   int        `json:"calorieGoal"`
	EstimatedCost float64    `gorm:"type:numeric(12,2)" json:"estimatedCost"`
	GeneratedByAI bool       `gorm:"default:false" json:"generatedByAi"`
	Items         []MealItem `gorm:"constraint:OnDelete:CASCADE;" json:"items"`
	User          *User      `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type MealItem struct {
	UUIDModel
	MealPlanID    uuid.UUID      `gorm:"type:uuid;index" json:"mealPlanId"`
	DayOfWeek     MealDay        `gorm:"type:varchar(5)" json:"dayOfWeek"`
	MealType      MealType       `gorm:"type:varchar(10)" json:"mealType"`
	Title         string         `gorm:"size:120" json:"title"`
	EstimatedCost float64        `gorm:"type:numeric(12,2)" json:"estimatedCost"`
	Ingredients   datatypes.JSON `gorm:"type:jsonb" json:"ingredients"`
	Instructions  string         `gorm:"type:text" json:"instructions"`
	MealPlan      *MealPlan      `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type Session struct {
	Token     string    `gorm:"size:64;primaryKey" json:"token"`
	UserID    uuid.UUID `gorm:"type:uuid;index" json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `gorm:"index" json:"expiresAt"`
	Valid     bool      `gorm:"default:true" json:"valid"`
	User      *User     `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type SyncJob struct {
	UUIDModel
	UserID     uuid.UUID  `gorm:"type:uuid;index" json:"userId"`
	Origin     SyncOrigin `gorm:"type:varchar(10)" json:"origin"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Status     SyncStatus `gorm:"type:varchar(10)" json:"status"`
	User       *User      `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type RequestType string

const (
	RequestTypeReceipt  RequestType = "receipt"
	RequestTypeInsight  RequestType = "insight"
	RequestTypeMealPlan RequestType = "meal_plan"
)

type TokenUsage struct {
	UUIDModel
	UserID         uuid.UUID         `gorm:"type:uuid;index;not null" json:"userId"`
	RequestType    RequestType       `gorm:"type:varchar(20)" json:"requestType"`
	RequestID      uuid.UUID         `gorm:"type:uuid;index" json:"requestId"`
	PromptTokens   int64             `json:"promptTokens"`
	ResponseTokens int64             `json:"responseTokens"`
	TotalTokens    int64             `json:"totalTokens"`
	CostInCents    int64             `json:"costInCents"`
	Metadata       datatypes.JSONMap `gorm:"serializer:json" json:"metadata"`
	User           *User             `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type ExpenseItem struct {
	UUIDModel
	ExpenseID   uuid.UUID `gorm:"type:uuid;index" json:"expenseId"`
	Name        string    `gorm:"size:180" json:"name"`
	Quantity    float64   `gorm:"type:numeric(12,3)" json:"quantity"`
	UnitPrice   float64   `gorm:"type:numeric(12,2)" json:"unitPrice"`
	TotalPrice  float64   `gorm:"type:numeric(12,2)" json:"totalPrice"`
	CategoryTag string    `gorm:"size:80" json:"category"`
	Expense     *Expense  `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}
