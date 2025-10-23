package handler

import (
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type categoryTotal struct {
	CategoryID uuid.UUID
	Total      float64
}

// DashboardSummaryHandler godoc
// @Summary Resumo do dashboard
// @Description Retorna totais do mês atual e comparação com o mês anterior
// @Tags Dashboard
// @Security Bearer
// @Produce json
// @Param month query int false "Mês (1-12)"
// @Param year query int false "Ano"
// @Success 200 {object} DashboardSummaryResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Router /dashboard/summary [get]
func DashboardSummaryHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	filter, err := buildExpenseFilter(ctx)
	if err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	currentStart, currentEnd := monthInterval(filter.Month, filter.Year)
	previousStart := currentStart.AddDate(0, -1, 0)
	previousEnd := currentStart

	currentTotal := aggregateTotal(user.ID, currentStart, currentEnd)
	previousTotal := aggregateTotal(user.ID, previousStart, previousEnd)

	topCategories := fetchTopCategories(user.ID, currentStart, currentEnd)

	variation := 0.0
	if previousTotal > 0 {
		variation = ((currentTotal - previousTotal) / previousTotal) * 100
	}

	response := DashboardSummaryResponse{
		Month:         filter.Month,
		Year:          filter.Year,
		TotalSpent:    roundFloat(currentTotal),
		PreviousTotal: roundFloat(previousTotal),
		VariationPct:  roundFloat(variation),
		TopCategories: topCategories,
	}

	respondSuccess(ctx, "dashboard", response)
}

func aggregateTotal(userID uuid.UUID, start, end time.Time) float64 {
	var total float64
	getDB().Model(&schemas.Expense{}).
		Select("COALESCE(SUM(amount),0)").
		Where("user_id = ? AND date >= ? AND date < ?", userID, start, end).
		Scan(&total)
	return total
}

func fetchTopCategories(userID uuid.UUID, start, end time.Time) []CategoryAggregate {
	totals := []categoryTotal{}
	getDB().Model(&schemas.Expense{}).
		Select("category_id, SUM(amount) as total").
		Where("user_id = ? AND date >= ? AND date < ?", userID, start, end).
		Group("category_id").
		Order("total DESC").
		Limit(5).
		Scan(&totals)

	responses := make([]CategoryAggregate, 0, len(totals))
	for _, total := range totals {
		category := schemas.Category{}
		if err := getDB().First(&category, "id = ?", total.CategoryID).Error; err != nil {
			continue
		}
		responses = append(responses, CategoryAggregate{
			Category: *toCategoryResponse(&category),
			Total:    roundFloat(total.Total),
		})
	}
	return responses
}
