package handler

import (
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" {
			respondError(ctx, 401, "token ausente", nil)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondError(ctx, 401, "formato de token inválido", nil)
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			respondError(ctx, 401, "token vazio", nil)
			return
		}

		session := schemas.Session{}
		err := getDB().Preload("User.Config").First(&session, "token = ?", token).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				respondError(ctx, 401, "sessão não encontrada", nil)
				return
			}
			respondError(ctx, 500, "erro ao validar sessão", err.Error())
			return
		}

		if !session.Valid || session.ExpiresAt.Before(time.Now()) {
			respondError(ctx, 401, "sessão expirada", nil)
			return
		}

		user := session.User
		if user == nil {
			respondError(ctx, 401, "usuário não associado à sessão", nil)
			return
		}

		setAuthenticatedUser(ctx, user)
		setCurrentSession(ctx, &session)
		ctx.Next()
	}
}
