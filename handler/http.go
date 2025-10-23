package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type APISuccess struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func respondError(ctx *gin.Context, status int, message string, details interface{}) {
	ctx.Header("Content-Type", "application/json")
	ctx.AbortWithStatusJSON(status, APIError{Message: message, Details: details})
}

func respondSuccess(ctx *gin.Context, message string, data interface{}) {
	ctx.Header("Content-Type", "application/json")
	ctx.JSON(http.StatusOK, APISuccess{Message: message, Data: data})
}

func bindJSON(ctx *gin.Context, dest interface{}) bool {
	if err := ctx.ShouldBindJSON(dest); err != nil {
		respondError(ctx, http.StatusBadRequest, "payload inválido", err.Error())
		return false
	}
	return true
}
