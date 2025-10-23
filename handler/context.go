package handler

import (
	"errors"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
)

const (
	contextUserKey    = "currentUser"
	contextSessionKey = "currentSession"
)

var errUserNotFound = errors.New("usuário não encontrado no contexto")

func setAuthenticatedUser(ctx *gin.Context, user *schemas.User) {
	ctx.Set(contextUserKey, user)
}

func getAuthenticatedUser(ctx *gin.Context) (*schemas.User, error) {
	value, exists := ctx.Get(contextUserKey)
	if !exists {
		return nil, errUserNotFound
	}

	user, ok := value.(*schemas.User)
	if !ok {
		return nil, errUserNotFound
	}
	return user, nil
}

func setCurrentSession(ctx *gin.Context, session *schemas.Session) {
	ctx.Set(contextSessionKey, session)
}

func getCurrentSession(ctx *gin.Context) (*schemas.Session, bool) {
	value, exists := ctx.Get(contextSessionKey)
	if !exists {
		return nil, false
	}
	session, ok := value.(*schemas.Session)
	if !ok {
		return nil, false
	}
	return session, true
}
