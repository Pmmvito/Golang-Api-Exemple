package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/Pmmvito/Golang-Api-Exemple/service/gemini"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GetMealPlanHandler godoc
// @Summary Consultar plano de refeições da semana
// @Description Retorna o plano de refeições salvo para a semana ISO informada (padrão: semana atual)
// @Tags Refeições
// @Security Bearer
// @Produce json
// @Param week query string false "Semana no formato YYYY-Www (ex: 2024-W37)"
// @Success 200 {object} MealPlanResponse
// @Failure 401 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /meal-plans [get]
func GetMealPlanHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	isoWeek := strings.TrimSpace(ctx.Query("week"))
	var year, week int
	if isoWeek == "" {
		year, week, isoWeek = currentISOWeek()
	} else {
		var parseErr error
		year, week, parseErr = parseISOWeek(isoWeek)
		if parseErr != nil {
			respondError(ctx, 400, "semana inválida", parseErr.Error())
			return
		}
		isoWeek = formatISOWeek(year, week)
	}

	plan, err := loadMealPlan(ctx.Request.Context(), user.ID, isoWeek)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(ctx, 404, "plano não encontrado", fmt.Sprintf("nenhum plano salvo para %s", isoWeek))
			return
		}
		respondError(ctx, 500, "erro ao carregar plano", err.Error())
		return
	}

	respondSuccess(ctx, "plano", toMealPlanResponse(plan))
}

// GenerateMealPlanHandler godoc
// @Summary Gerar plano de refeições com Gemini
// @Description Cria um plano semanal com receitas baseadas nos itens de compras recentes. Usa heurísticas se o modelo não estiver disponível.
// @Tags Refeições
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body GenerateMealPlanRequest false "Preferências para geração"
// @Success 200 {object} MealPlanResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /meal-plans/generate [post]
func GenerateMealPlanHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	var request GenerateMealPlanRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		if !errors.Is(err, io.EOF) {
			respondError(ctx, 400, "payload inválido", err.Error())
			return
		}
	}

	isoWeek := strings.TrimSpace(request.Week)
	var year, week int
	if isoWeek == "" {
		year, week, isoWeek = currentISOWeek()
	} else {
		var parseErr error
		year, week, parseErr = parseISOWeek(isoWeek)
		if parseErr != nil {
			respondError(ctx, 400, "semana inválida", parseErr.Error())
			return
		}
		isoWeek = formatISOWeek(year, week)
	}

	plan, usage, modelName, aiErr := generateMealPlanWithAI(ctx, user, isoWeek, &request, year, week)
	aiUsed := aiErr == nil && plan != nil
	if !aiUsed {
		if aiErr != nil {
			getLogger().WarnF("falha ao gerar plano com gemini: %v", aiErr)
		}
		plan = generateHeuristicMealPlan(user.ID, isoWeek, &request)
	}

	if plan == nil {
		respondError(ctx, 500, "não foi possível gerar plano de refeições", nil)
		return
	}

	if err := persistMealPlan(ctx.Request.Context(), plan); err != nil {
		respondError(ctx, 500, "erro ao persistir plano", err.Error())
		return
	}

	stored, err := loadMealPlan(ctx.Request.Context(), user.ID, isoWeek)
	if err != nil {
		respondError(ctx, 500, "erro ao atualizar plano gerado", err.Error())
		return
	}

	if aiUsed && usage != nil {
		metadata := datatypes.JSONMap{
			"isoWeek":       isoWeek,
			"items":         len(plan.Items),
			"calorieGoal":   plan.CalorieGoal,
			"estimatedCost": plan.EstimatedCost,
			"model":         modelName,
		}
		recordCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
		defer cancel()
		if _, logErr := recordTokenUsage(recordCtx, user.ID, schemas.RequestTypeMealPlan, *usage, metadata); logErr != nil {
			getLogger().WarnF("não foi possível registrar uso de tokens: %v", logErr)
		}
	}

	source := "heurísticas"
	if aiUsed {
		source = "gemini"
	}

	respondSuccess(ctx, fmt.Sprintf("plano gerado via %s", source), toMealPlanResponse(stored))
}

func generateMealPlanWithAI(ctx *gin.Context, user *schemas.User, isoWeek string, request *GenerateMealPlanRequest, year, week int) (*schemas.MealPlan, *gemini.UsageMetadata, string, error) {
	client, err := gemini.NewClientFromEnv()
	if err != nil {
		return nil, nil, "", err
	}

	startOfWeek := isoWeekStartDate(year, week)
	endOfWeek := startOfWeek.AddDate(0, 0, 7)

	expenses := fetchRecentExpenses(ctx.Request.Context(), user.ID, 12)
	items := fetchRecentItems(ctx.Request.Context(), user.ID, 20)
	topCategories := fetchTopCategories(user.ID, startOfWeek, endOfWeek)

	currency := "BRL"
	language := "pt-BR"
	if user.Config != nil {
		if user.Config.Currency != "" {
			currency = strings.ToUpper(strings.TrimSpace(user.Config.Currency))
		}
		if user.Config.Language != "" {
			language = strings.TrimSpace(user.Config.Language)
		}
	}

	prompt := buildMealPlanPrompt(user.Name, isoWeek, startOfWeek, currency, language, request, expenses, items, topCategories)
	modelName := detectModelName()

	req := gemini.GenerateContentRequest{
		Contents: []gemini.Content{
			{
				Role: "user",
				Parts: []gemini.ContentPart{
					gemini.NewTextPart(prompt),
				},
			},
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 45*time.Second)
	defer cancel()

	result, err := client.GenerateContent(ctxTimeout, req)
	if err != nil {
		return nil, nil, modelName, err
	}

	sanitized := gemini.SanitizeJSON(result.Text)
	payload, err := parseMealPlanPayload(sanitized)
	if err != nil {
		return nil, nil, modelName, err
	}

	plan := convertToMealPlan(user.ID, isoWeek, request, payload)
	if plan == nil {
		return nil, nil, modelName, fmt.Errorf("modelo não retornou refeições válidas")
	}

	usage := result.Usage
	return plan, &usage, modelName, nil
}

type aiMeal struct {
	Day           string   `json:"day"`
	MealType      string   `json:"mealType"`
	Title         string   `json:"title"`
	Ingredients   []string `json:"ingredients"`
	Instructions  string   `json:"instructions"`
	EstimatedCost float64  `json:"estimatedCost"`
}

type aiMealPlanPayload struct {
	EstimatedCost float64  `json:"estimatedCost"`
	CalorieGoal   int      `json:"calorieGoal"`
	Meals         []aiMeal `json:"meals"`
}

func parseMealPlanPayload(raw string) (*aiMealPlanPayload, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("resposta vazia do modelo")
	}

	var payload aiMealPlanPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if len(payload.Meals) == 0 {
		return nil, fmt.Errorf("modelo retornou 0 receitas")
	}
	return &payload, nil
}

func convertToMealPlan(userID uuid.UUID, isoWeek string, request *GenerateMealPlanRequest, payload *aiMealPlanPayload) *schemas.MealPlan {
	plan := &schemas.MealPlan{
		UserID:        userID,
		IsoWeek:       isoWeek,
		GeneratedByAI: true,
		EstimatedCost: roundFloat(payload.EstimatedCost),
	}

	if payload.CalorieGoal > 0 {
		plan.CalorieGoal = payload.CalorieGoal
	} else if request.CalorieGoal != nil {
		plan.CalorieGoal = *request.CalorieGoal
	} else {
		plan.CalorieGoal = 2000
	}

	for _, meal := range payload.Meals {
		day, ok := normalizeMealDay(meal.Day)
		if !ok {
			continue
		}
		mealType, ok := normalizeMealType(meal.MealType)
		if !ok {
			continue
		}
		title := strings.TrimSpace(meal.Title)
		if title == "" {
			continue
		}
		ingredients := sanitizeStringSlice(meal.Ingredients)
		ingredientsJSON, _ := json.Marshal(ingredients)

		plan.Items = append(plan.Items, schemas.MealItem{
			DayOfWeek:     day,
			MealType:      mealType,
			Title:         title,
			EstimatedCost: roundFloat(meal.EstimatedCost),
			Ingredients:   datatypes.JSON(ingredientsJSON),
			Instructions:  strings.TrimSpace(meal.Instructions),
		})
	}

	if len(plan.Items) == 0 {
		return nil
	}

	if plan.EstimatedCost <= 0 {
		plan.EstimatedCost = roundFloat(float64(len(plan.Items)) * 18)
	}

	return plan
}

func generateHeuristicMealPlan(userID uuid.UUID, isoWeek string, request *GenerateMealPlanRequest) *schemas.MealPlan {
	plan := &schemas.MealPlan{
		UserID:        userID,
		IsoWeek:       isoWeek,
		GeneratedByAI: false,
		CalorieGoal:   2000,
		EstimatedCost: 210,
	}

	if request.CalorieGoal != nil && *request.CalorieGoal > 0 {
		plan.CalorieGoal = *request.CalorieGoal
	}

	breakfasts := []string{
		"Iogurte natural com granola e frutas",
		"Ovos mexidos com torradas integrais",
		"Vitamina de banana com aveia",
	}
	lunches := []string{
		"Peito de frango grelhado com legumes assados",
		"Tilápia ao forno com salada de quinoa",
		"Carne magra ensopada com batata-doce",
		"Arroz integral com feijão e legumes salteados",
	}
	dinners := []string{
		"Sopa de legumes com torradas integrais",
		"Omelete de espinafre e queijo branco",
		"Macarrão integral ao pesto com frango desfiado",
	}

	days := []schemas.MealDay{
		schemas.MealDayMonday,
		schemas.MealDayTuesday,
		schemas.MealDayWednesday,
		schemas.MealDayThursday,
		schemas.MealDayFriday,
		schemas.MealDaySaturday,
		schemas.MealDaySunday,
	}

	for i, day := range days {
		breakfastTitle := breakfasts[i%len(breakfasts)]
		lunchTitle := lunches[i%len(lunches)]
		dinnerTitle := dinners[i%len(dinners)]

		plan.Items = append(plan.Items, buildHeuristicMeal(day, schemas.MealTypeBreakfast, breakfastTitle, []string{"Iogurte natural", "Granola", "Frutas da estação"}, "Monte o bowl com iogurte, adicione a granola e finalize com frutas frescas."))
		plan.Items = append(plan.Items, buildHeuristicMeal(day, schemas.MealTypeLunch, lunchTitle, []string{"Proteína magra", "Legumes variados", "Azeite"}, "Tempere a proteína e os legumes com azeite, asse até dourar e sirva quente."))
		plan.Items = append(plan.Items, buildHeuristicMeal(day, schemas.MealTypeDinner, dinnerTitle, []string{"Legumes frescos", "Caldo de legumes", "Pão integral"}, "Cozinhe os legumes no caldo até ficarem macios e sirva acompanhados de torradas integrais."))
	}

	return plan
}

func buildHeuristicMeal(day schemas.MealDay, mealType schemas.MealType, title string, ingredients []string, instructions string) schemas.MealItem {
	ingredientsJSON, _ := json.Marshal(ingredients)
	cost := 18.0
	switch mealType {
	case schemas.MealTypeBreakfast:
		cost = 12
	case schemas.MealTypeDinner:
		cost = 20
	}

	return schemas.MealItem{
		DayOfWeek:     day,
		MealType:      mealType,
		Title:         title,
		EstimatedCost: roundFloat(cost),
		Ingredients:   datatypes.JSON(ingredientsJSON),
		Instructions:  instructions,
	}
}

func persistMealPlan(ctx context.Context, plan *schemas.MealPlan) error {
	if plan == nil {
		return fmt.Errorf("plano inválido")
	}

	return getDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND iso_week = ?", plan.UserID, plan.IsoWeek).Delete(&schemas.MealPlan{}).Error; err != nil {
			return err
		}
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		return nil
	})
}

func loadMealPlan(ctx context.Context, userID uuid.UUID, isoWeek string) (*schemas.MealPlan, error) {
	plan := schemas.MealPlan{}
	err := getDB().WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("day_of_week ASC, meal_type ASC")
		}).
		Where("user_id = ? AND iso_week = ?", userID, isoWeek).
		First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func fetchRecentItems(ctx context.Context, userID uuid.UUID, limit int) []schemas.ExpenseItem {
	items := []schemas.ExpenseItem{}
	if limit <= 0 {
		limit = 15
	}
	getDB().WithContext(ctx).
		Joins("JOIN expenses ON expenses.id = expense_items.expense_id").
		Where("expenses.user_id = ?", userID).
		Order("expense_items.created_at DESC").
		Limit(limit).
		Find(&items)
	return items
}

func buildMealPlanPrompt(name, isoWeek string, start time.Time, currency, language string, request *GenerateMealPlanRequest, expenses []schemas.Expense, items []schemas.ExpenseItem, topCategories []CategoryAggregate) string {
	var builder strings.Builder
	builder.WriteString("Você é um nutricionista financeiro que cria planos de refeições realistas.\n")
	builder.WriteString("Entregue receitas práticas usando ingredientes do histórico de compras.\n")
	builder.WriteString("Retorne apenas JSON com este formato:\n")
	builder.WriteString(`{"estimatedCost":number,"calorieGoal":number,"meals":[{"day":"seg|ter|...","mealType":"cafe|almoco|janta|lanche","title":"...","ingredients":["ingredient"],"instructions":"passo a passo","estimatedCost":number}]}` + "\n")
	builder.WriteString("Use ponto como separador decimal e idioma " + language + ".\n")
	builder.WriteString(fmt.Sprintf("Planeje a semana ISO %s iniciando em %s.\n", isoWeek, start.Format("02/01/2006")))
	builder.WriteString(fmt.Sprintf("Moeda preferida: %s.\n", currency))

	if request != nil {
		if request.CalorieGoal != nil && *request.CalorieGoal > 0 {
			builder.WriteString(fmt.Sprintf("Objetivo calórico diário: %d kcal.\n", *request.CalorieGoal))
		}
		if request.Servings != nil && *request.Servings > 0 {
			builder.WriteString(fmt.Sprintf("Número de porções por refeição: %d.\n", *request.Servings))
		}
		if request.DietaryPreference != "" {
			builder.WriteString(fmt.Sprintf("Preferência alimentar: %s.\n", request.DietaryPreference))
		}
		if len(request.Exclusions) > 0 {
			builder.WriteString("Evite ingredientes: " + strings.Join(request.Exclusions, ", ") + ".\n")
		}
		if request.Budget != nil && *request.Budget > 0 {
			builder.WriteString(fmt.Sprintf("Orçamento semanal máximo: %.2f %s.\n", *request.Budget, currency))
		}
	}

	if len(topCategories) > 0 {
		builder.WriteString("Categorias com mais gastos recentes:\n")
		for i, cat := range topCategories {
			builder.WriteString(fmt.Sprintf("  %d. %s — %.2f %s\n", i+1, cat.Category.Name, cat.Total, currency))
		}
	}

	if len(items) > 0 {
		builder.WriteString("Itens de mercado recentes:\n")
		for i, item := range items {
			if i >= 20 {
				break
			}
			builder.WriteString(fmt.Sprintf("  - %s (%.2f unidades) — total %.2f %s\n", item.Name, item.Quantity, item.TotalPrice, currency))
		}
	} else if len(expenses) > 0 {
		builder.WriteString("Despesas recentes relevantes:\n")
		for i, expense := range expenses {
			if i >= 10 {
				break
			}
			catName := ""
			if expense.Category != nil {
				catName = expense.Category.Name
			}
			builder.WriteString(fmt.Sprintf("  - %s (%s) em %s — %.2f %s\n", expense.Description, expense.Date.Format("02/01"), catName, expense.Amount, currency))
		}
	}

	builder.WriteString("Inclua instruções passo a passo curtas (máx 3 frases) para cada refeição.\n")
	builder.WriteString("Garanta que os dias usem a sigla em português (seg, ter, qua, qui, sex, sab, dom).\n")
	builder.WriteString("Se possível, reutilize ingredientes para reduzir custos e destaque vibrações positivas.\n")

	return builder.String()
}

func normalizeMealDay(value string) (schemas.MealDay, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "seg", "segunda", "segunda-feira", "monday":
		return schemas.MealDayMonday, true
	case "ter", "terca", "terça", "terça-feira", "tuesday":
		return schemas.MealDayTuesday, true
	case "qua", "quarta", "quarta-feira", "wednesday":
		return schemas.MealDayWednesday, true
	case "qui", "quinta", "quinta-feira", "thursday":
		return schemas.MealDayThursday, true
	case "sex", "sexta", "sexta-feira", "friday":
		return schemas.MealDayFriday, true
	case "sab", "sábado", "sabado", "saturday":
		return schemas.MealDaySaturday, true
	case "dom", "domingo", "sunday":
		return schemas.MealDaySunday, true
	default:
		return schemas.MealDay(""), false
	}
}

func normalizeMealType(value string) (schemas.MealType, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "cafe", "café", "café da manhã", "breakfast":
		return schemas.MealTypeBreakfast, true
	case "almoco", "almoço", "lunch":
		return schemas.MealTypeLunch, true
	case "janta", "jantar", "dinner":
		return schemas.MealTypeDinner, true
	case "lanche", "snack":
		return schemas.MealTypeSnack, true
	default:
		return schemas.MealType(""), false
	}
}

func sanitizeStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func currentISOWeek() (int, int, string) {
	now := time.Now()
	year, week := now.ISOWeek()
	return year, week, formatISOWeek(year, week)
}

func parseISOWeek(value string) (int, int, error) {
	var year, week int
	if _, err := fmt.Sscanf(value, "%d-W%d", &year, &week); err != nil {
		return 0, 0, err
	}
	if week < 1 || week > 53 {
		return 0, 0, fmt.Errorf("semana fora do intervalo (1-53)")
	}
	return year, week, nil
}

func formatISOWeek(year, week int) string {
	return fmt.Sprintf("%d-W%02d", year, week)
}

func isoWeekStartDate(year, week int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -(weekday - 1))
	return monday.AddDate(0, 0, (week-1)*7)
}
