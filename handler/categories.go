package handler

import (
	"strconv"
	"strings"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListCategoriesHandler godoc
// @Summary Listar categorias
// @Description Retorna as categorias do usuário autenticado
// @Tags Categorias
// @Security Bearer
// @Produce json
// @Param status query string false "Filtra por status das categorias. Use true para apenas ativas, false para apenas inativas e deixe em branco ou use all para todas." Enums(true,false,all)
// @Success 200 {object} CategoryListSuccess
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /categories [get]
func ListCategoriesHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	status := strings.TrimSpace(ctx.Query("status"))

	query := getDB().Where("user_id = ?", user.ID)

	switch {
	case status == "" || strings.EqualFold(status, "all"):
		if includeInactiveRaw, ok := ctx.GetQuery("includeInactive"); ok {
			includeInactive, err := strconv.ParseBool(includeInactiveRaw)
			if err != nil {
				respondError(ctx, 400, "includeInactive inválido", nil)
				return
			}
			if !includeInactive {
				query = query.Where("active = ?", true)
			}
		}
	default:
		if strings.EqualFold(status, "active") || strings.EqualFold(status, "ativo") || strings.EqualFold(status, "ativa") {
			query = query.Where("active = ?", true)
		} else if strings.EqualFold(status, "inactive") || strings.EqualFold(status, "inativo") || strings.EqualFold(status, "inativa") {
			query = query.Where("active = ?", false)
		} else {
			flag, err := strconv.ParseBool(status)
			if err != nil {
				respondError(ctx, 400, "status inválido (use true, false ou all)", nil)
				return
			}
			query = query.Where("active = ?", flag)
		}
	}

	var categories []schemas.Category
	if err := query.Order("\"order\" asc, name asc").Find(&categories).Error; err != nil {
		respondError(ctx, 500, "erro ao listar categorias", err.Error())
		return
	}

	responses := make([]CategoryResponse, len(categories))
	for i, category := range categories {
		resp := toCategoryResponse(&category)
		if resp != nil {
			responses[i] = *resp
		}
	}

	respondSuccess(ctx, "categorias", responses)
}

// CreateCategoryHandler godoc
// @Summary Criar categoria
// @Description Cria uma categoria personalizada para o usuário
// @Tags Categorias
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body CategoryRequest true "Dados da categoria"
// @Success 200 {object} CategoryItemSuccess
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /categories [post]
func CreateCategoryHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	var request CategoryRequest
	if !bindJSON(ctx, &request) {
		return
	}

	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	category := schemas.Category{
		UUIDModel: schemas.UUIDModel{ID: uuid.New()},
		UserID:    user.ID,
		Name:      strings.TrimSpace(request.Name),
		Icon:      strings.TrimSpace(request.Icon),
		ColorHex:  strings.TrimSpace(request.ColorHex),
		Type:      schemas.CategoryType(request.Type),
	}

	if request.Active != nil {
		category.Active = *request.Active
	} else {
		category.Active = true // apenas se não vier no request
	}

	if request.Order != nil {
		category.Order = *request.Order
	}

	columns := []string{"id", "user_id", "name", "icon", "color_hex", "type", "order", "active"}
	if err := getDB().Select(columns).Create(&category).Error; err != nil {
		respondError(ctx, 500, "erro ao criar categoria", err.Error())
		return
	}

	if request.Active != nil {
		if err := getDB().Model(&category).Update("active", *request.Active).Error; err != nil {
			respondError(ctx, 500, "erro ao ajustar status da categoria", err.Error())
			return
		}
	}
	if err := getDB().First(&category, "id = ?", category.ID).Error; err != nil {
		respondError(ctx, 500, "erro ao carregar categoria", err.Error())
		return
	}

	respondSuccess(ctx, "categoria criada", toCategoryResponse(&category))
}

// UpdateCategoryHandler godoc
// @Summary Atualizar categoria
// @Description Atualiza dados de uma categoria existente
// @Tags Categorias
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path string true "Identificador da categoria"
// @Param body body CategoryRequest true "Campos para atualização"
// @Success 200 {object} CategoryItemSuccess
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /categories/{id} [put]
func UpdateCategoryHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	idParam := ctx.Param("id")
	categoryID, err := uuid.Parse(idParam)
	if err != nil {
		respondError(ctx, 400, "id inválido", nil)
		return
	}

	var request CategoryRequest
	if !bindJSON(ctx, &request) {
		return
	}

	updates := map[string]interface{}{}
	if request.Name != "" {
		updates["name"] = strings.TrimSpace(request.Name)
	}
	if request.Icon != "" {
		updates["icon"] = strings.TrimSpace(request.Icon)
	}
	if request.ColorHex != "" {
		if !strings.HasPrefix(request.ColorHex, "#") {
			respondError(ctx, 400, "cor deve estar no formato hexadecimal", nil)
			return
		}
		updates["color_hex"] = strings.TrimSpace(request.ColorHex)
	}
	if request.Type != "" {
		switch schemas.CategoryType(request.Type) {
		case schemas.CategoryTypeFixed, schemas.CategoryTypeVariable:
			updates["type"] = request.Type
		default:
			respondError(ctx, 400, "tipo inválido", nil)
			return
		}
	}
	if request.Order != nil {
		updates["order"] = *request.Order
	}
	if request.Active != nil {
		updates["active"] = *request.Active
	}

	if len(updates) == 0 {
		respondError(ctx, 400, "nenhum campo para atualizar", nil)
		return
	}

	var category schemas.Category
	if err := getDB().Where("id = ? AND user_id = ?", categoryID, user.ID).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(ctx, 404, "categoria não encontrada", nil)
			return
		}
		respondError(ctx, 500, "erro ao atualizar categoria", err.Error())
		return
	}

	if err := getDB().Model(&category).Updates(updates).Error; err != nil {
		respondError(ctx, 500, "erro ao persistir categoria", err.Error())
		return
	}

	if err := getDB().First(&category, "id = ?", categoryID).Error; err != nil {
		respondError(ctx, 500, "erro ao carregar categoria atualizada", err.Error())
		return
	}

	respondSuccess(ctx, "categoria atualizada", toCategoryResponse(&category))
}

// DeleteCategoryHandler godoc
// @Summary Desativar categoria
// @Description Marca uma categoria como inativa
// @Tags Categorias
// @Security Bearer
// @Param id path string true "Identificador da categoria"
// @Success 200 {object} APISuccess
// @Failure 401 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /categories/{id} [delete]
func DeleteCategoryHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	idParam := ctx.Param("id")
	categoryID, err := uuid.Parse(idParam)
	if err != nil {
		respondError(ctx, 400, "id inválido", nil)
		return
	}

	result := getDB().Model(&schemas.Category{}).
		Where("id = ? AND user_id = ?", categoryID, user.ID).
		Updates(map[string]interface{}{"active": false})

	if result.Error != nil {
		respondError(ctx, 500, "erro ao desativar categoria", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(ctx, 404, "categoria não encontrada", nil)
		return
	}

	respondSuccess(ctx, "categoria desativada", nil)
}
