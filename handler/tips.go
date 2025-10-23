package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/Pmmvito/Golang-Api-Exemple/service/gemini"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ListTipsHandler godoc
// @Summary Listar dicas financeiras
// @Description Retorna dicas ordenadas por relevância para o usuário autenticado
// @Tags Dicas
// @Security Bearer
// @Produce json
// @Param month query int false "Mês (1-12)"
// @Param year query int false "Ano"
// @Param refresh query bool false "true para recalcular dicas antes de responder"
// @Success 200 {array} TipResponse
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /tips [get]
func ListTipsHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	now := time.Now()
	month := parseIntDefault(ctx.Query("month"), int(now.Month()))
	year := parseIntDefault(ctx.Query("year"), now.Year())
	refresh := strings.EqualFold(ctx.Query("refresh"), "true")

	tips, err := loadTips(ctx.Request.Context(), user.ID)
	if err != nil {
		respondError(ctx, 500, "erro ao carregar dicas", err.Error())
		return
	}

	if refresh || len(tips) == 0 {
		generated, _, genErr := regenerateTips(ctx, user, month, year)
		if genErr == nil {
			tips = generated
		} else if len(tips) == 0 {
			respondError(ctx, 500, "não foi possível gerar dicas", genErr.Error())
			return
		}
	}

	responses := make([]TipResponse, 0, len(tips))
	for i := range tips {
		responses = append(responses, toTipResponse(&tips[i]))
	}

	respondSuccess(ctx, "dicas", responses)
}

// GenerateTipsHandler godoc
// @Summary Gerar novas dicas financeiras
// @Description Recalcula dicas com base nos gastos recentes e Gemini. Usa heurísticas caso o modelo não esteja disponível.
// @Tags Dicas
// @Security Bearer
// @Produce json
// @Param month query int false "Mês (1-12)"
// @Param year query int false "Ano"
// @Success 200 {array} TipResponse
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /tips/generate [post]
func GenerateTipsHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	now := time.Now()
	month := parseIntDefault(ctx.Query("month"), int(now.Month()))
	year := parseIntDefault(ctx.Query("year"), now.Year())

	generated, aiUsed, err := regenerateTips(ctx, user, month, year)
	if err != nil {
		respondError(ctx, 500, "não foi possível gerar dicas", err.Error())
		return
	}

	responses := make([]TipResponse, 0, len(generated))
	for i := range generated {
		responses = append(responses, toTipResponse(&generated[i]))
	}

	source := "heurísticas"
	if aiUsed {
		source = "gemini"
	}
	respondSuccess(ctx, fmt.Sprintf("dicas geradas via %s", source), responses)
}

func regenerateTips(ctx *gin.Context, user *schemas.User, month, year int) ([]schemas.GeneratedTip, bool, error) {
	aiTips, usage, modelName, err := generateTipsWithAI(ctx, user, month, year)
	if err == nil && len(aiTips) > 0 {
		if err := persistTips(ctx.Request.Context(), user.ID, aiTips); err != nil {
			return nil, true, err
		}

		metadata := datatypes.JSONMap{
			"month":         month,
			"year":          year,
			"tipsGenerated": len(aiTips),
			"model":         modelName,
		}

		recordCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Second)
		defer cancel()
		if usage != nil {
			if _, logErr := recordTokenUsage(recordCtx, user.ID, schemas.RequestTypeInsight, *usage, metadata); logErr != nil {
				getLogger().WarnF("não foi possível registrar uso de tokens: %v", logErr)
			}
		}

		stored, loadErr := loadTips(ctx.Request.Context(), user.ID)
		if loadErr != nil {
			return nil, true, loadErr
		}
		return stored, true, nil
	}

	if err != nil {
		getLogger().WarnF("falha ao gerar dicas com gemini: %v", err)
	}

	if clearErr := clearTips(ctx.Request.Context(), user.ID); clearErr != nil {
		return nil, false, clearErr
	}

	generated, heurErr := generateHeuristicTips(user)
	if heurErr != nil {
		return nil, false, heurErr
	}

	return generated, false, nil
}

func generateTipsWithAI(ctx *gin.Context, user *schemas.User, month, year int) ([]schemas.GeneratedTip, *gemini.UsageMetadata, string, error) {
	client, err := gemini.NewClientFromEnv()
	if err != nil {
		return nil, nil, "", err
	}

	start, end := monthInterval(month, year)
	total := aggregateTotal(user.ID, start, end)
	topCategories := fetchTopCategories(user.ID, start, end)
	recentExpenses := fetchRecentExpenses(ctx.Request.Context(), user.ID, 6)

	currency := "BRL"
	monthlyLimit := 0.0
	language := "pt-BR"
	if user.Config != nil {
		if user.Config.Currency != "" {
			currency = strings.ToUpper(strings.TrimSpace(user.Config.Currency))
		}
		if user.Config.MonthlyLimit > 0 {
			monthlyLimit = user.Config.MonthlyLimit
		}
		if user.Config.Language != "" {
			language = strings.TrimSpace(user.Config.Language)
		}
	}

	prompt := buildTipsPrompt(user.Name, currency, language, month, year, total, monthlyLimit, topCategories, recentExpenses)
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

	ctxTimeout, cancel := context.WithTimeout(ctx.Request.Context(), 40*time.Second)
	defer cancel()

	result, err := client.GenerateContent(ctxTimeout, req)
	if err != nil {
		return nil, nil, modelName, err
	}

	sanitized := gemini.SanitizeJSON(result.Text)
	payload, err := parseTipsPayload(sanitized)
	if err != nil {
		return nil, nil, modelName, err
	}

	tips := convertToGeneratedTips(user.ID, payload, modelName)
	if len(tips) == 0 {
		return nil, nil, modelName, fmt.Errorf("modelo não retornou dicas válidas")
	}

	usage := result.Usage
	return tips, &usage, modelName, nil
}

func loadTips(ctx context.Context, userID uuid.UUID) ([]schemas.GeneratedTip, error) {
	tips := []schemas.GeneratedTip{}
	if err := getDB().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("relevance DESC, created_at DESC").
		Limit(5).
		Find(&tips).Error; err != nil {
		return nil, err
	}
	return tips, nil
}

func persistTips(ctx context.Context, userID uuid.UUID, tips []schemas.GeneratedTip) error {
	if len(tips) == 0 {
		return fmt.Errorf("nenhuma dica para persistir")
	}
	return getDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&schemas.GeneratedTip{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&tips).Error; err != nil {
			return err
		}
		return nil
	})
}

func clearTips(ctx context.Context, userID uuid.UUID) error {
	return getDB().WithContext(ctx).Where("user_id = ?", userID).Delete(&schemas.GeneratedTip{}).Error
}

func fetchRecentExpenses(ctx context.Context, userID uuid.UUID, limit int) []schemas.Expense {
	expenses := []schemas.Expense{}
	if limit <= 0 {
		limit = 5
	}
	getDB().WithContext(ctx).Preload("Category").
		Where("user_id = ?", userID).
		Order("date DESC").
		Limit(limit).
		Find(&expenses)
	return expenses
}

type aiTip struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Relevance int    `json:"relevance"`
}

type aiTipPayload struct {
	Tips []aiTip `json:"tips"`
}

func parseTipsPayload(raw string) (*aiTipPayload, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("resposta vazia do modelo")
	}
	var payload aiTipPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if len(payload.Tips) == 0 {
		return nil, fmt.Errorf("modelo retornou 0 dicas")
	}
	return &payload, nil
}

func convertToGeneratedTips(userID uuid.UUID, payload *aiTipPayload, model string) []schemas.GeneratedTip {
	tips := make([]schemas.GeneratedTip, 0, len(payload.Tips))
	for _, item := range payload.Tips {
		text := strings.TrimSpace(item.Message)
		if text == "" {
			continue
		}
		tipType := normalizeTipType(item.Type)
		relevance := clampRelevance(item.Relevance)
		tips = append(tips, schemas.GeneratedTip{
			UserID:      userID,
			Type:        tipType,
			Text:        text,
			ModelSource: model,
			Relevance:   relevance,
		})
	}
	return tips
}

func normalizeTipType(value string) schemas.TipType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alerta", "alert", "warning":
		return schemas.TipTypeAlert
	case "economia", "saving", "savings":
		return schemas.TipTypeSavings
	default:
		return schemas.TipTypePlanning
	}
}

func clampRelevance(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func buildTipsPrompt(name, currency, language string, month, year int, total, limit float64, categories []CategoryAggregate, expenses []schemas.Expense) string {
	var builder strings.Builder
	builder.WriteString("Você é um assistente financeiro pessoal.\n")
	builder.WriteString("Use os dados fornecidos para criar de 3 a 5 dicas práticas e motivacionais.\n")
	builder.WriteString("Responda apenas em JSON no formato {\"tips\":[{\"type\":\"...\",\"message\":\"...\",\"relevance\":int}]} sem comentários adicionais.\n")
	builder.WriteString("Tipos permitidos: alerta, planejamento, economia. O campo relevance deve estar entre 0 e 100.\n")
	builder.WriteString("Dados do usuário:\n")
	builder.WriteString(fmt.Sprintf("- Nome: %s\n", strings.TrimSpace(name)))
	builder.WriteString(fmt.Sprintf("- Mês analisado: %02d/%d\n", month, year))
	builder.WriteString(fmt.Sprintf("- Total gasto no período: %.2f %s\n", total, currency))
	if limit > 0 {
		builder.WriteString(fmt.Sprintf("- Limite mensal configurado: %.2f %s\n", limit, currency))
	}
	if len(categories) > 0 {
		builder.WriteString("- Principais categorias:\n")
		for i, cat := range categories {
			builder.WriteString(fmt.Sprintf("  %d. %s — %.2f %s\n", i+1, cat.Category.Name, cat.Total, currency))
		}
	}
	if len(expenses) > 0 {
		builder.WriteString("- Despesas recentes:\n")
		for i, exp := range expenses {
			if i >= 8 {
				break
			}
			catName := ""
			if exp.Category != nil {
				catName = exp.Category.Name
			}
			builder.WriteString(fmt.Sprintf("  - %s: %s em %s (%.2f %s)\n", exp.Date.Format("02/01"), exp.Description, catName, exp.Amount, currency))
		}
	}
	builder.WriteString("Considera que o idioma preferido do usuário é " + language + ". Sempre inclua orientações acionáveis, curtas e claras.\n")
	builder.WriteString("Se o usuário estiver perto ou acima do limite, priorize dicas de alerta e planejamento.\n")
	builder.WriteString("Garanta que cada dica esteja adaptada ao contexto apresentado.\n")
	return builder.String()
}

func generateHeuristicTips(user *schemas.User) ([]schemas.GeneratedTip, error) {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()
	start, end := monthInterval(month, year)

	total := aggregateTotal(user.ID, start, end)
	tops := fetchTopCategories(user.ID, start, end)

	tips := []schemas.GeneratedTip{}

	if user.Config != nil && user.Config.MonthlyLimit > 0 {
		limit := user.Config.MonthlyLimit
		if total > limit {
			tips = append(tips, schemas.GeneratedTip{
				UserID:      user.ID,
				Type:        schemas.TipTypeAlert,
				Text:        fmt.Sprintf("Você já ultrapassou seu limite mensal de R$ %.2f. Revise seus gastos das últimas semanas.", limit),
				ModelSource: "heuristic",
				Relevance:   95,
			})
		} else if total > limit*0.85 {
			tips = append(tips, schemas.GeneratedTip{
				UserID:      user.ID,
				Type:        schemas.TipTypePlanning,
				Text:        fmt.Sprintf("Atingiu %d%% do limite mensal. Considere pausar compras não essenciais para evitar surpresas.", int((total/limit)*100)),
				ModelSource: "heuristic",
				Relevance:   80,
			})
		}
	}

	if len(tops) > 0 {
		top := tops[0]
		tips = append(tips, schemas.GeneratedTip{
			UserID:      user.ID,
			Type:        schemas.TipTypeSavings,
			Text:        fmt.Sprintf("Categoria %s representa R$ %.2f neste mês. Avalie trocas ou renegociações para reduzir esse custo.", top.Category.Name, top.Total),
			ModelSource: "heuristic",
			Relevance:   75,
		})
	}

	tips = append(tips, schemas.GeneratedTip{
		UserID:      user.ID,
		Type:        schemas.TipTypePlanning,
		Text:        "Reserve 10 minutos para revisar seu fluxo de caixa e planejar a próxima semana.",
		ModelSource: "heuristic",
		Relevance:   60,
	})

	if len(tips) == 0 {
		return nil, nil
	}

	if err := getDB().Create(&tips).Error; err != nil {
		if err == gorm.ErrInvalidData {
			return nil, nil
		}
		return nil, err
	}

	return tips, nil
}
