package handler

import (
	"context"
	"math"
	"os"
	"strconv"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/Pmmvito/Golang-Api-Exemple/service/gemini"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	envPromptCostPer1K   = "GEMINI_PROMPT_COST_PER_1K_CENTS"
	envResponseCostPer1K = "GEMINI_RESPONSE_COST_PER_1K_CENTS"
)

func recordTokenUsage(ctx context.Context, userID uuid.UUID, requestType schemas.RequestType, usage gemini.UsageMetadata, metadata datatypes.JSONMap) (*schemas.TokenUsage, error) {
	entry := &schemas.TokenUsage{
		UserID:         userID,
		RequestType:    requestType,
		RequestID:      uuid.New(),
		PromptTokens:   usage.PromptTokenCount,
		ResponseTokens: usage.CandidatesTokenCount,
		TotalTokens:    usage.TotalTokenCount,
		CostInCents:    estimateTokenCost(usage),
	}

	if metadata != nil {
		entry.Metadata = metadata
	} else {
		entry.Metadata = datatypes.JSONMap{}
	}

	if err := getDB().WithContext(ctx).Create(entry).Error; err != nil {
		return nil, err
	}

	return entry, nil
}

// ListTokenUsageHandler godoc
// @Summary Listar consumo de tokens
// @Description Retorna o histórico de consumo de tokens do usuário autenticado com totais agregados
// @Tags Token Usage
// @Security Bearer
// @Produce json
// @Param limit query int false "Número máximo de registros por página (1-200)"
// @Param page query int false "Página a ser retornada (>=1)"
// @Success 200 {object} TokenUsageListSuccess
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /token-usage [get]
func ListTokenUsageHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	limit := parseIntDefault(ctx.Query("limit"), 50)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	page := parseIntDefault(ctx.Query("page"), 1)
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	var totalEntries int64
	if err := getDB().Model(&schemas.TokenUsage{}).
		Where("user_id = ?", user.ID).
		Count(&totalEntries).Error; err != nil {
		respondError(ctx, 500, "erro ao contar consumo de tokens", err.Error())
		return
	}

	var usages []schemas.TokenUsage
	if err := getDB().Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&usages).Error; err != nil {
		respondError(ctx, 500, "erro ao listar consumo de tokens", err.Error())
		return
	}

	totals, err := aggregateTokenUsageTotals(user.ID)
	if err != nil {
		respondError(ctx, 500, "erro ao calcular totais de tokens", err.Error())
		return
	}

	entries := make([]TokenUsageEntryResponse, len(usages))
	for i := range usages {
		entries[i] = toTokenUsageEntryResponse(&usages[i])
	}

	response := TokenUsageListResponse{
		Entries: entries,
		Summary: TokenUsageSummary{
			TotalPromptTokens:   totals.PromptSum,
			TotalResponseTokens: totals.ResponseSum,
			TotalTokens:         totals.TotalSum,
			TotalCostCents:      totals.CostSum,
		},
		Pagination: TokenUsagePagination{
			Page:         page,
			Limit:        limit,
			TotalEntries: totalEntries,
		},
	}

	respondSuccess(ctx, "consumo de tokens", response)
}

type tokenUsageTotals struct {
	PromptSum   int64 `gorm:"column:prompt_sum"`
	ResponseSum int64 `gorm:"column:response_sum"`
	TotalSum    int64 `gorm:"column:total_sum"`
	CostSum     int64 `gorm:"column:cost_sum"`
}

func aggregateTokenUsageTotals(userID uuid.UUID) (tokenUsageTotals, error) {
	totals := tokenUsageTotals{}
	err := getDB().Model(&schemas.TokenUsage{}).
		Select("COALESCE(SUM(prompt_tokens),0) AS prompt_sum, COALESCE(SUM(response_tokens),0) AS response_sum, COALESCE(SUM(total_tokens),0) AS total_sum, COALESCE(SUM(cost_in_cents),0) AS cost_sum").
		Where("user_id = ?", userID).
		Scan(&totals).Error
	return totals, err
}

func estimateTokenCost(usage gemini.UsageMetadata) int64 {
	promptRate := lookupCostRate(envPromptCostPer1K)
	responseRate := lookupCostRate(envResponseCostPer1K)

	promptCost := (float64(usage.PromptTokenCount) / 1000.0) * promptRate
	responseCost := (float64(usage.CandidatesTokenCount) / 1000.0) * responseRate

	total := promptCost + responseCost
	if total <= 0 {
		return 0
	}

	return int64(math.Round(total))
}

func lookupCostRate(envKey string) float64 {
	raw := os.Getenv(envKey)
	if raw == "" {
		return 0
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		getLogger().Warn("valor inválido para " + envKey + ": " + raw)
		return 0
	}

	return value
}
