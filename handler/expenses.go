package handler

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateExpenseHandler godoc
// @Summary Criar despesa
// @Description Registra uma nova despesa para o usuário autenticado
// @Tags Despesas
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body ExpenseRequest true "Dados da despesa"
// @Success 200 {object} ExpenseResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 500 {object} APIError
// @Router /expenses [post]
func CreateExpenseHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	var request ExpenseRequest
	if !bindJSON(ctx, &request) {
		return
	}
	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	categoryID, err := uuid.Parse(request.CategoryID)
	if err != nil {
		respondError(ctx, 400, "categoryId inválido", nil)
		return
	}

	if !categoryBelongsToUser(getDB(), user.ID, categoryID) {
		respondError(ctx, 403, "categoria não pertence ao usuário", nil)
		return
	}

	date, err := parseDate(request.Date)
	if err != nil {
		respondError(ctx, 400, "data inválida", err.Error())
		return
	}

	var createdExpense schemas.Expense

	if err := getDB().Transaction(func(tx *gorm.DB) error {
		expense := schemas.Expense{
			UserID:      user.ID,
			CategoryID:  categoryID,
			Description: request.Description,
			Amount:      request.Amount,
			Date:        date,
			Recurring:   request.Recurring,
			Origin:      schemas.ExpenseOrigin(request.Origin),
		}
		if err := tx.Create(&expense).Error; err != nil {
			return err
		}

		if request.Receipt != nil {
			confidence := 0.0
			if request.Receipt.OcrConfidence != nil {
				confidence = *request.Receipt.OcrConfidence
			}
			receipt := schemas.Receipt{
				ExpenseID:     expense.ID,
				FilePath:      request.Receipt.FilePath,
				ExtractedText: request.Receipt.ExtractedText,
				OcrConfidence: confidence,
			}
			if err := tx.Create(&receipt).Error; err != nil {
				return err
			}
			expense.Receipt = &receipt
		}

		createdExpense = expense
		return nil
	}); err != nil {
		respondError(ctx, 500, "erro ao criar despesa", err.Error())
		return
	}

	if err := getDB().Preload("Category").Preload("Receipt").First(&createdExpense, "id = ?", createdExpense.ID).Error; err != nil {
		respondError(ctx, 500, "erro ao carregar despesa criada", err.Error())
		return
	}

	respondSuccess(ctx, "despesa criada", toExpenseResponse(&createdExpense))
}

// ListExpensesHandler godoc
// @Summary Listar despesas
// @Description Lista despesas filtradas por mês e ano
// @Tags Despesas
// @Security Bearer
// @Produce json
// @Param month query int false "Mês (1-12)"
// @Param year query int false "Ano"
// @Param categoryId query string false "Filtro por categoria"
// @Param origin query string false "Origem: manual|ocr|ia"
// @Success 200 {object} ExpensesListResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Router /expenses [get]
func ListExpensesHandler(ctx *gin.Context) {
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

	start, end := monthInterval(filter.Month, filter.Year)

	query := getDB().Preload("Category").Preload("Receipt").
		Where("user_id = ?", user.ID).
		Where("date >= ? AND date < ?", start, end)
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.Origin != nil {
		query = query.Where("origin = ?", *filter.Origin)
	}

	var expenses []schemas.Expense
	if err := query.Order("date DESC").Find(&expenses).Error; err != nil {
		respondError(ctx, 500, "erro ao listar despesas", err.Error())
		return
	}

	responses := make([]ExpenseResponse, len(expenses))
	total := 0.0
	for i := range expenses {
		responses[i] = *toExpenseResponse(&expenses[i])
		total += expenses[i].Amount
	}

	summary := ExpenseSummary{TotalCount: len(expenses), TotalAmount: total}
	if len(expenses) > 0 {
		summary.AverageValue = roundFloat(total / float64(len(expenses)))
	}

	respondSuccess(ctx, "despesas", ExpensesListResponse{Expenses: responses, Summary: summary})
}

// GetExpenseHandler godoc
// @Summary Buscar despesa
// @Description Retorna os detalhes de uma despesa específica
// @Tags Despesas
// @Security Bearer
// @Produce json
// @Param id path string true "Identificador da despesa"
// @Success 200 {object} ExpenseResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 404 {object} APIError
// @Router /expenses/{id} [get]
func GetExpenseHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	expenseID, err := parseUUIDParam(ctx.Param("id"))
	if err != nil {
		respondError(ctx, 400, "id inválido", nil)
		return
	}

	expense := schemas.Expense{}
	if err := getDB().Preload("Category").Preload("Receipt").
		Where("id = ? AND user_id = ?", expenseID, user.ID).
		First(&expense).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(ctx, 404, "despesa não encontrada", nil)
			return
		}
		respondError(ctx, 500, "erro ao carregar despesa", err.Error())
		return
	}

	respondSuccess(ctx, "despesa", toExpenseResponse(&expense))
}

// UpdateExpenseHandler godoc
// @Summary Atualizar despesa
// @Description Atualiza campos de uma despesa existente
// @Tags Despesas
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path string true "Identificador da despesa"
// @Param body body UpdateExpenseRequest true "Campos para atualização"
// @Success 200 {object} ExpenseResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 404 {object} APIError
// @Router /expenses/{id} [put]
func UpdateExpenseHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	expenseID, err := parseUUIDParam(ctx.Param("id"))
	if err != nil {
		respondError(ctx, 400, "id inválido", nil)
		return
	}

	var request UpdateExpenseRequest
	if !bindJSON(ctx, &request) {
		return
	}
	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	err = getDB().Transaction(func(tx *gorm.DB) error {
		expense := schemas.Expense{}
		if err := tx.Where("id = ? AND user_id = ?", expenseID, user.ID).First(&expense).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{}
		if request.CategoryID != nil {
			categoryUUID, err := uuid.Parse(*request.CategoryID)
			if err != nil {
				return err
			}
			if !categoryBelongsToUser(tx, user.ID, categoryUUID) {
				return gorm.ErrInvalidData
			}
			updates["category_id"] = categoryUUID
		}
		if request.Description != nil {
			updates["description"] = *request.Description
		}
		if request.Amount != nil {
			updates["amount"] = *request.Amount
		}
		if request.Date != nil {
			parsedDate, err := parseDate(*request.Date)
			if err != nil {
				return err
			}
			updates["date"] = parsedDate
		}
		if request.Recurring != nil {
			updates["recurring"] = *request.Recurring
		}
		if request.Origin != nil {
			updates["origin"] = schemas.ExpenseOrigin(*request.Origin)
		}

		if len(updates) > 0 {
			if err := tx.Model(&expense).Updates(updates).Error; err != nil {
				return err
			}
		}

		if request.RemoveReceipt {
			if err := tx.Where("expense_id = ?", expense.ID).Delete(&schemas.Receipt{}).Error; err != nil {
				return err
			}
		} else if request.Receipt != nil {
			confidence := 0.0
			if request.Receipt.OcrConfidence != nil {
				confidence = *request.Receipt.OcrConfidence
			}
			receipt := schemas.Receipt{}
			if err := tx.Where("expense_id = ?", expense.ID).First(&receipt).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					receipt = schemas.Receipt{
						ExpenseID:     expense.ID,
						FilePath:      request.Receipt.FilePath,
						ExtractedText: request.Receipt.ExtractedText,
						OcrConfidence: confidence,
					}
					if err := tx.Create(&receipt).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				if err := tx.Model(&receipt).Updates(map[string]interface{}{
					"file_path":      request.Receipt.FilePath,
					"extracted_text": request.Receipt.ExtractedText,
					"ocr_confidence": confidence,
				}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(ctx, 404, "despesa não encontrada", nil)
			return
		}
		if err == gorm.ErrInvalidData {
			respondError(ctx, 403, "categoria não pertence ao usuário", nil)
			return
		}
		respondError(ctx, 400, "erro ao atualizar despesa", err.Error())
		return
	}

	updated := schemas.Expense{}
	if err := getDB().Preload("Category").Preload("Receipt").
		Where("id = ? AND user_id = ?", expenseID, user.ID).
		First(&updated).Error; err != nil {
		respondError(ctx, 500, "erro ao recarregar despesa", err.Error())
		return
	}

	respondSuccess(ctx, "despesa atualizada", toExpenseResponse(&updated))
}

// DeleteExpenseHandler godoc
// @Summary Remover despesa
// @Description Exclui uma despesa definitivamente
// @Tags Despesas
// @Security Bearer
// @Param id path string true "Identificador da despesa"
// @Success 200 {object} APISuccess
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 404 {object} APIError
// @Router /expenses/{id} [delete]
func DeleteExpenseHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	expenseID, err := parseUUIDParam(ctx.Param("id"))
	if err != nil {
		respondError(ctx, 400, "id inválido", nil)
		return
	}

	result := getDB().Where("id = ? AND user_id = ?", expenseID, user.ID).Delete(&schemas.Expense{})
	if result.Error != nil {
		respondError(ctx, 500, "erro ao remover despesa", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(ctx, 404, "despesa não encontrada", nil)
		return
	}

	respondSuccess(ctx, "despesa removida", nil)
}

func buildExpenseFilter(ctx *gin.Context) (ExpenseFilter, error) {
	now := time.Now()
	month := parseIntDefault(ctx.Query("month"), int(now.Month()))
	year := parseIntDefault(ctx.Query("year"), now.Year())

	filter := ExpenseFilter{Month: month, Year: year}

	if categoryParam := ctx.Query("categoryId"); categoryParam != "" {
		categoryID, err := uuid.Parse(categoryParam)
		if err != nil {
			return filter, err
		}
		filter.CategoryID = &categoryID
	}

	if originParam := ctx.Query("origin"); originParam != "" {
		origin := schemas.ExpenseOrigin(originParam)
		switch origin {
		case schemas.ExpenseOriginManual, schemas.ExpenseOriginOCR, schemas.ExpenseOriginAI:
			filter.Origin = &origin
		default:
			return filter, gorm.ErrInvalidData
		}
	}

	return filter, nil
}

func monthInterval(month, year int) (time.Time, time.Time) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end
}

func parseDate(value string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("data inválida")
}

func parseUUIDParam(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return fallback
}

func categoryBelongsToUser(db *gorm.DB, userID uuid.UUID, categoryID uuid.UUID) bool {
	var count int64
	if err := db.Model(&schemas.Category{}).
		Where("id = ? AND user_id = ?", categoryID, userID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}
