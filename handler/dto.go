package handler

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	Currency     string   `json:"currency"`
	MonthlyLimit *float64 `json:"monthlyLimit,omitempty"`
	Language     string   `json:"language"`
	Theme        string   `json:"theme"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogoutRequest struct {
	AllDevices bool `json:"allDevices"`
}

type CategoryRequest struct {
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	ColorHex string `json:"colorHex"`
	Type     string `json:"type"`
	Order    *int   `json:"order,omitempty"`
	Active   *bool  `json:"active,omitempty"`
}

type ExpenseRequest struct {
	CategoryID  string        `json:"categoryId"`
	Description string        `json:"description"`
	Amount      float64       `json:"amount"`
	Date        string        `json:"date"`
	Recurring   bool          `json:"recurring"`
	Origin      string        `json:"origin"`
	Receipt     *ReceiptInput `json:"receipt,omitempty"`
}

type UpdateExpenseRequest struct {
	CategoryID    *string       `json:"categoryId,omitempty"`
	Description   *string       `json:"description,omitempty"`
	Amount        *float64      `json:"amount,omitempty"`
	Date          *string       `json:"date,omitempty"`
	Recurring     *bool         `json:"recurring,omitempty"`
	Origin        *string       `json:"origin,omitempty"`
	Receipt       *ReceiptInput `json:"receipt,omitempty"`
	RemoveReceipt bool          `json:"removeReceipt,omitempty"`
}

type ReceiptInput struct {
	FilePath      string   `json:"filePath"`
	ExtractedText string   `json:"extractedText"`
	OcrConfidence *float64 `json:"ocrConfidence,omitempty"`
}

type ReceiptScanRequest struct {
	ImageBase64 string   `json:"imageBase64"`
	Currency    string   `json:"currency"`
	AmountHint  *float64 `json:"amountHint,omitempty"`
	Locale      string   `json:"locale,omitempty"`
	ReturnRaw   bool     `json:"returnRaw,omitempty"`
}

type GenerateMealPlanRequest struct {
	Week              string   `json:"week,omitempty"`
	CalorieGoal       *int     `json:"calorieGoal,omitempty"`
	Servings          *int     `json:"servings,omitempty"`
	DietaryPreference string   `json:"dietaryPreference,omitempty"`
	Exclusions        []string `json:"exclusions,omitempty"`
	Budget            *float64 `json:"budget,omitempty"`
}

type ReceiptItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Total       float64 `json:"total"`
}

type ExpenseFilter struct {
	Month      int
	Year       int
	CategoryID *uuid.UUID
	Origin     *schemas.ExpenseOrigin
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expiresAt"`
	User      UserResponse `json:"user"`
}

type UserResponse struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Email     string              `json:"email"`
	Active    bool                `json:"active"`
	LastLogin *time.Time          `json:"lastLogin,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Config    *UserConfigResponse `json:"config,omitempty"`
}

type UserConfigResponse struct {
	Currency             string  `json:"currency"`
	MonthlyLimit         float64 `json:"monthlyLimit"`
	NotificationsEnabled bool    `json:"notificationsEnabled"`
	Language             string  `json:"language"`
	Theme                string  `json:"theme"`
}

type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	ColorHex  string    `json:"colorHex"`
	Type      string    `json:"type"`
	Order     int       `json:"order"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ReceiptResponse struct {
	ID            string  `json:"id"`
	FilePath      string  `json:"filePath"`
	ExtractedText string  `json:"extractedText"`
	OcrConfidence float64 `json:"ocrConfidence"`
}

type ExpenseResponse struct {
	ID          string            `json:"id"`
	CategoryID  string            `json:"categoryId"`
	Description string            `json:"description"`
	Amount      float64           `json:"amount"`
	Date        time.Time         `json:"date"`
	Recurring   bool              `json:"recurring"`
	Origin      string            `json:"origin"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Category    *CategoryResponse `json:"category,omitempty"`
	Receipt     *ReceiptResponse  `json:"receipt,omitempty"`
}

type ExpenseSummary struct {
	TotalCount   int     `json:"totalCount"`
	TotalAmount  float64 `json:"totalAmount"`
	AverageValue float64 `json:"averageValue"`
}

type ExpensesListResponse struct {
	Expenses []ExpenseResponse `json:"expenses"`
	Summary  ExpenseSummary    `json:"summary"`
}

type CategoryAggregate struct {
	Category CategoryResponse `json:"category"`
	Total    float64          `json:"total"`
}

type DashboardSummaryResponse struct {
	Month         int                 `json:"month"`
	Year          int                 `json:"year"`
	TotalSpent    float64             `json:"totalSpent"`
	PreviousTotal float64             `json:"previousTotal"`
	VariationPct  float64             `json:"variationPct"`
	TopCategories []CategoryAggregate `json:"topCategories"`
}

type ReceiptScanResponse struct {
	SuggestedAmount float64          `json:"suggestedAmount"`
	SuggestedDate   string           `json:"suggestedDate"`
	Currency        string           `json:"currency"`
	ExtractedText   string           `json:"extractedText"`
	Items           []ReceiptItem    `json:"items"`
	Confidence      float64          `json:"confidence"`
	TokensUsed      int64            `json:"tokensUsed"`
	TokenCostCents  int64            `json:"tokenCostCents"`
	Model           string           `json:"model"`
	RawModelOutput  string           `json:"rawModelOutput,omitempty"`
	SavedExpense    *ExpenseResponse `json:"savedExpense,omitempty"`
}

type TokenUsageEntryResponse struct {
	ID             string                 `json:"id"`
	RequestType    string                 `json:"requestType"`
	RequestID      string                 `json:"requestId,omitempty"`
	PromptTokens   int64                  `json:"promptTokens"`
	ResponseTokens int64                  `json:"responseTokens"`
	TotalTokens    int64                  `json:"totalTokens"`
	CostInCents    int64                  `json:"costInCents"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
}

type TokenUsageSummary struct {
	TotalPromptTokens   int64 `json:"totalPromptTokens"`
	TotalResponseTokens int64 `json:"totalResponseTokens"`
	TotalTokens         int64 `json:"totalTokens"`
	TotalCostCents      int64 `json:"totalCostCents"`
}

type TokenUsagePagination struct {
	Page         int   `json:"page"`
	Limit        int   `json:"limit"`
	TotalEntries int64 `json:"totalEntries"`
}

type TokenUsageListResponse struct {
	Entries    []TokenUsageEntryResponse `json:"entries"`
	Summary    TokenUsageSummary         `json:"summary"`
	Pagination TokenUsagePagination      `json:"pagination"`
}

type SyncJobResponse struct {
	ID         string     `json:"id"`
	Origin     string     `json:"origin"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type TipResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Source    string    `json:"source"`
	Relevance int       `json:"relevance"`
	CreatedAt time.Time `json:"createdAt"`
}

type MealPlanResponse struct {
	ID            string             `json:"id"`
	IsoWeek       string             `json:"isoWeek"`
	CalorieGoal   int                `json:"calorieGoal"`
	EstimatedCost float64            `json:"estimatedCost"`
	GeneratedByAI bool               `json:"generatedByAi"`
	CreatedAt     time.Time          `json:"createdAt"`
	Items         []MealItemResponse `json:"items"`
}

type MealItemResponse struct {
	ID            string   `json:"id"`
	DayOfWeek     string   `json:"dayOfWeek"`
	MealType      string   `json:"mealType"`
	Title         string   `json:"title"`
	EstimatedCost float64  `json:"estimatedCost"`
	Ingredients   []string `json:"ingredients"`
	Instructions  string   `json:"instructions"`
}

type SyncRequest struct {
	Origin string `json:"origin"`
}

func (r *RegisterRequest) Normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Name = strings.TrimSpace(r.Name)

	currency := strings.ToUpper(strings.TrimSpace(r.Currency))
	if currency == "" {
		currency = "BRL"
	}
	r.Currency = currency

	language := strings.TrimSpace(r.Language)
	if language == "" {
		language = "pt-BR"
	}
	r.Language = language

	theme := strings.TrimSpace(r.Theme)
	if theme == "" {
		theme = string(schemas.ThemeSystem)
	}
	r.Theme = theme
}

func (r *RegisterRequest) Validate() error {
	if r.Name == "" {
		return errors.New("nome é obrigatório")
	}
	if r.Email == "" {
		return errors.New("email é obrigatório")
	}
	if !strings.Contains(r.Email, "@") {
		return errors.New("email inválido")
	}
	if len(r.Password) < 6 {
		return errors.New("senha deve ter pelo menos 6 caracteres")
	}

	if len(r.Currency) != 3 {
		return errors.New("currency deve conter exatamente 3 letras (ex: BRL)")
	}
	for _, ch := range r.Currency {
		if ch < 'A' || ch > 'Z' {
			return errors.New("currency deve conter apenas letras A-Z")
		}
	}

	if len(r.Language) > 5 {
		return errors.New("language deve ter no máximo 5 caracteres (ex: pt-BR)")
	}

	switch schemas.Theme(r.Theme) {
	case schemas.ThemeLight, schemas.ThemeDark, schemas.ThemeSystem:
	default:
		return errors.New("theme inválido")
	}

	return nil
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" || r.Password == "" {
		return errors.New("email e senha são obrigatórios")
	}
	return nil
}

func (r *CategoryRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("nome é obrigatório")
	}
	if strings.TrimSpace(r.Type) == "" {
		return errors.New("tipo é obrigatório")
	}
	switch schemas.CategoryType(r.Type) {
	case schemas.CategoryTypeFixed, schemas.CategoryTypeVariable:
	default:
		return errors.New("tipo inválido")
	}
	if r.ColorHex != "" && !strings.HasPrefix(r.ColorHex, "#") {
		return errors.New("cor deve estar em formato hexadecimal (#RRGGBB)")
	}
	return nil
}

func (r *ExpenseRequest) Validate() error {
	if r.CategoryID == "" {
		return errors.New("categoryId é obrigatório")
	}
	if r.Description == "" {
		return errors.New("descrição é obrigatória")
	}
	if r.Amount <= 0 {
		return errors.New("valor deve ser maior que zero")
	}
	if r.Date == "" {
		return errors.New("data é obrigatória")
	}
	if r.Origin == "" {
		r.Origin = string(schemas.ExpenseOriginManual)
	}
	switch schemas.ExpenseOrigin(r.Origin) {
	case schemas.ExpenseOriginManual, schemas.ExpenseOriginOCR, schemas.ExpenseOriginAI:
	default:
		return errors.New("origem inválida")
	}
	return nil
}

func (r *UpdateExpenseRequest) Validate() error {
	if r.CategoryID == nil && r.Description == nil && r.Amount == nil && r.Date == nil && r.Recurring == nil && r.Origin == nil && r.Receipt == nil && !r.RemoveReceipt {
		return errors.New("nenhum campo para atualizar")
	}
	if r.CategoryID != nil && *r.CategoryID == "" {
		return errors.New("categoryId inválido")
	}
	if r.Amount != nil && *r.Amount <= 0 {
		return errors.New("valor deve ser maior que zero")
	}
	if r.Origin != nil {
		switch schemas.ExpenseOrigin(*r.Origin) {
		case schemas.ExpenseOriginManual, schemas.ExpenseOriginOCR, schemas.ExpenseOriginAI:
		default:
			return errors.New("origem inválida")
		}
	}
	return nil
}

func (r *SyncRequest) Validate() error {
	if r.Origin == "" {
		r.Origin = string(schemas.SyncOriginMobile)
	}
	switch schemas.SyncOrigin(r.Origin) {
	case schemas.SyncOriginMobile, schemas.SyncOriginWeb, schemas.SyncOriginBackup:
		return nil
	default:
		return errors.New("origem de sincronização inválida")
	}
}

func toUserResponse(user *schemas.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}

	response := UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		Active:    user.Active,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.Config != nil {
		response.Config = &UserConfigResponse{
			Currency:             user.Config.Currency,
			MonthlyLimit:         user.Config.MonthlyLimit,
			NotificationsEnabled: user.Config.NotificationsEnabled,
			Language:             user.Config.Language,
			Theme:                string(user.Config.Theme),
		}
	}

	return response
}

func toCategoryResponse(category *schemas.Category) *CategoryResponse {
	if category == nil {
		return nil
	}
	return &CategoryResponse{
		ID:        category.ID.String(),
		Name:      category.Name,
		Icon:      category.Icon,
		ColorHex:  category.ColorHex,
		Type:      string(category.Type),
		Order:     category.Order,
		Active:    category.Active,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func toExpenseResponse(expense *schemas.Expense) *ExpenseResponse {
	if expense == nil {
		return nil
	}

	resp := &ExpenseResponse{
		ID:          expense.ID.String(),
		CategoryID:  expense.CategoryID.String(),
		Description: expense.Description,
		Amount:      expense.Amount,
		Date:        expense.Date,
		Recurring:   expense.Recurring,
		Origin:      string(expense.Origin),
		CreatedAt:   expense.CreatedAt,
		UpdatedAt:   expense.UpdatedAt,
	}

	if expense.Category != nil {
		resp.Category = toCategoryResponse(expense.Category)
	}
	if expense.Receipt != nil {
		resp.Receipt = &ReceiptResponse{
			ID:            expense.Receipt.ID.String(),
			FilePath:      expense.Receipt.FilePath,
			ExtractedText: expense.Receipt.ExtractedText,
			OcrConfidence: expense.Receipt.OcrConfidence,
		}
	}
	return resp
}

func toTipResponse(tip *schemas.GeneratedTip) TipResponse {
	return TipResponse{
		ID:        tip.ID.String(),
		Type:      string(tip.Type),
		Text:      tip.Text,
		Source:    tip.ModelSource,
		Relevance: tip.Relevance,
		CreatedAt: tip.CreatedAt,
	}
}

func toMealPlanResponse(plan *schemas.MealPlan) MealPlanResponse {
	if plan == nil {
		return MealPlanResponse{}
	}

	items := make([]MealItemResponse, 0, len(plan.Items))
	for _, item := range plan.Items {
		var ingredients []string
		if len(item.Ingredients) > 0 {
			if err := json.Unmarshal(item.Ingredients, &ingredients); err != nil {
				ingredients = []string{}
			}
		}
		items = append(items, MealItemResponse{
			ID:            item.ID.String(),
			DayOfWeek:     string(item.DayOfWeek),
			MealType:      string(item.MealType),
			Title:         item.Title,
			EstimatedCost: item.EstimatedCost,
			Ingredients:   ingredients,
			Instructions:  item.Instructions,
		})
	}

	return MealPlanResponse{
		ID:            plan.ID.String(),
		IsoWeek:       plan.IsoWeek,
		CalorieGoal:   plan.CalorieGoal,
		EstimatedCost: plan.EstimatedCost,
		GeneratedByAI: plan.GeneratedByAI,
		CreatedAt:     plan.CreatedAt,
		Items:         items,
	}
}

func toSyncJobResponse(job *schemas.SyncJob) SyncJobResponse {
	return SyncJobResponse{
		ID:         job.ID.String(),
		Origin:     string(job.Origin),
		Status:     string(job.Status),
		StartedAt:  job.StartedAt,
		FinishedAt: job.FinishedAt,
	}
}

func toTokenUsageEntryResponse(usage *schemas.TokenUsage) TokenUsageEntryResponse {
	if usage == nil {
		return TokenUsageEntryResponse{}
	}

	var metadata map[string]interface{}
	if usage.Metadata != nil {
		metadata = map[string]interface{}(usage.Metadata)
	}

	requestID := ""
	if usage.RequestID != uuid.Nil {
		requestID = usage.RequestID.String()
	}

	return TokenUsageEntryResponse{
		ID:             usage.ID.String(),
		RequestType:    string(usage.RequestType),
		RequestID:      requestID,
		PromptTokens:   usage.PromptTokens,
		ResponseTokens: usage.ResponseTokens,
		TotalTokens:    usage.TotalTokens,
		CostInCents:    usage.CostInCents,
		Metadata:       metadata,
		CreatedAt:      usage.CreatedAt,
	}
}
