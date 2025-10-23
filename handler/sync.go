package handler

import (
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
)

// TriggerSyncHandler godoc
// @Summary Registrar sincronização
// @Description Salva o histórico de uma sincronização concluída
// @Tags Sincronização
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body SyncRequest true "Origens"
// @Success 200 {object} SyncJobResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /sync/jobs [post]
func TriggerSyncHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	var request SyncRequest
	if !bindJSON(ctx, &request) {
		return
	}
	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	job := schemas.SyncJob{
		UserID:     user.ID,
		Origin:     schemas.SyncOrigin(request.Origin),
		StartedAt:  time.Now(),
		FinishedAt: ptrTime(time.Now()),
		Status:     schemas.SyncStatusOK,
	}

	if err := getDB().Create(&job).Error; err != nil {
		respondError(ctx, 500, "erro ao registrar sincronização", err.Error())
		return
	}

	respondSuccess(ctx, "sincronização concluída", toSyncJobResponse(&job))
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
