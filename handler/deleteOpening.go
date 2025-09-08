package handler

import (
	"fmt"
	"net/http"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
)
// @BasePath /api/v1

// @Sumary Delete opening
// @Description Delete a job Opening
// @Tags Openings
// @Accept json
// @Produce json
// @Param id query string true "Opening Identification"
// @Success 200 {object} DeleteOpeningResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /opening [delete]
func DeleteOpeningHandler(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		sendError(ctx, http.StatusBadRequest,errParamIsRequired("id",
		"queryParameter").Error())
		return
	}
	opening := schemas.Opening{}

	if err := db.First(&opening, id).Error; err != nil{
		sendError(ctx, http.StatusNotFound, fmt.Sprintf("opening whit id: %s not found", id ))
		return
	}

	if err := db.Delete(&opening, id).Error; err != nil {
		sendError(ctx, http.StatusInternalServerError,fmt.Sprintf("error deleting opening whih id: %s", id))
		return
	}
	sendSucces(ctx,"delete-opening",opening)
}