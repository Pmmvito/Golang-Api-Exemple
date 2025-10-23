package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterHandler godoc
// @Summary Registrar novo usuário
// @Description Cria um usuário, configura dados padrão e retorna a sessão autenticada
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Dados de registro"
// @Success 200 {object} AuthSuccessResponse
// @Failure 400 {object} APIError
// @Failure 409 {object} APIError
// @Failure 500 {object} APIError
// @Router /auth/register [post]
func RegisterHandler(ctx *gin.Context) {
	var request RegisterRequest
	if !bindJSON(ctx, &request) {
		return
	}

	request.Normalize()
	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	var count int64
	if err := getDB().Model(&schemas.User{}).Where("email = ?", request.Email).Count(&count).Error; err != nil {
		respondError(ctx, 500, "erro ao verificar email", err.Error())
		return
	}
	if count > 0 {
		respondError(ctx, 409, "email já cadastrado", nil)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(ctx, 500, "erro ao gerar hash de senha", err.Error())
		return
	}

	var createdUser schemas.User
	var createdSession schemas.Session

	err = getDB().Transaction(func(tx *gorm.DB) error {
		user := schemas.User{
			Name:         request.Name,
			Email:        request.Email,
			PasswordHash: string(passwordHash),
			Active:       true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		monthlyLimit := 0.0
		if request.MonthlyLimit != nil {
			monthlyLimit = *request.MonthlyLimit
		}

		config := schemas.UserConfig{
			UserID:               user.ID,
			Currency:             strings.ToUpper(request.Currency),
			MonthlyLimit:         monthlyLimit,
			NotificationsEnabled: true,
			Language:             request.Language,
			Theme:                schemas.Theme(request.Theme),
		}
		if err := tx.Create(&config).Error; err != nil {
			return err
		}

		if err := seedDefaultCategories(tx, user.ID); err != nil {
			return err
		}

		session, err := createSession(tx, user.ID)
		if err != nil {
			return err
		}

		createdUser = user
		createdUser.Config = &config
		createdSession = *session
		return nil
	})

	if err != nil {
		respondError(ctx, 500, "erro ao registrar usuário", err.Error())
		return
	}

	respondSuccess(ctx, "cadastro realizado", AuthResponse{
		Token:     createdSession.Token,
		ExpiresAt: createdSession.ExpiresAt,
		User:      toUserResponse(&createdUser),
	})
}

// LoginHandler godoc
// @Summary Autenticar usuário
// @Description Valida credenciais e emite um novo token de sessão
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Credenciais"
// @Success 200 {object} AuthSuccessResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /auth/login [post]
func LoginHandler(ctx *gin.Context) {
	var request LoginRequest
	if !bindJSON(ctx, &request) {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		respondError(ctx, 400, err.Error(), nil)
		return
	}

	user := schemas.User{}
	if err := getDB().Preload("Config").Where("email = ?", request.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(ctx, 401, "credenciais inválidas", nil)
			return
		}
		respondError(ctx, 500, "erro ao buscar usuário", err.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		respondError(ctx, 401, "credenciais inválidas", nil)
		return
	}

	var session schemas.Session
	err := getDB().Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		user.LastLogin = &now
		if err := tx.Model(&user).Updates(map[string]interface{}{"last_login": now}).Error; err != nil {
			return err
		}

		newSession, err := createSession(tx, user.ID)
		if err != nil {
			return err
		}
		session = *newSession
		return nil
	})
	if err != nil {
		respondError(ctx, 500, "erro ao criar sessão", err.Error())
		return
	}

	respondSuccess(ctx, "login realizado", AuthResponse{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		User:      toUserResponse(&user),
	})
}

// LogoutHandler godoc
// @Summary Encerrar sessão
// @Description Invalida a sessão atual ou todas as sessões do usuário autenticado
// @Tags Auth
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body LogoutRequest false "Opções de encerramento"
// @Success 200 {object} APISuccess
// @Failure 401 {object} APIError
// @Failure 500 {object} APIError
// @Router /auth/logout [post]
func LogoutHandler(ctx *gin.Context) {
	var request LogoutRequest
	if ctx.Request.ContentLength > 0 {
		if !bindJSON(ctx, &request) {
			return
		}
	}

	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	if err := getDB().Transaction(func(tx *gorm.DB) error {
		if request.AllDevices {
			return tx.Model(&schemas.Session{}).
				Where("user_id = ?", user.ID).
				Updates(map[string]interface{}{"valid": false}).Error
		}

		session, ok := getCurrentSession(ctx)
		if !ok {
			return errors.New("sessão atual não encontrada")
		}
		return tx.Model(&schemas.Session{}).
			Where("token = ?", session.Token).
			Updates(map[string]interface{}{"valid": false}).Error
	}); err != nil {
		respondError(ctx, 500, "erro ao encerrar sessão", err.Error())
		return
	}

	respondSuccess(ctx, "logout realizado", nil)
}

// MeHandler godoc
// @Summary Perfil do usuário
// @Description Retorna os dados do usuário autenticado
// @Tags Auth
// @Security Bearer
// @Produce json
// @Success 200 {object} UserProfileSuccess
// @Failure 401 {object} APIError
// @Router /auth/me [get]
func MeHandler(ctx *gin.Context) {
	user, err := getAuthenticatedUser(ctx)
	if err != nil {
		respondError(ctx, 401, "não autenticado", nil)
		return
	}

	respondSuccess(ctx, "perfil", toUserResponse(user))
}

func createSession(tx *gorm.DB, userID uuid.UUID) (*schemas.Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	session := schemas.Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(getSessionTTL()),
		Valid:     true,
	}

	if err := tx.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func seedDefaultCategories(tx *gorm.DB, userID uuid.UUID) error {
	defaults := []struct {
		Name  string
		Icon  string
		Color string
		Type  schemas.CategoryType
	}{
		{Name: "Alimentação", Icon: "utensils", Color: "#EF4444", Type: schemas.CategoryTypeVariable},
		{Name: "Transporte", Icon: "bus", Color: "#3B82F6", Type: schemas.CategoryTypeVariable},
		{Name: "Moradia", Icon: "home", Color: "#8B5CF6", Type: schemas.CategoryTypeFixed},
		{Name: "Saúde", Icon: "heart", Color: "#10B981", Type: schemas.CategoryTypeVariable},
		{Name: "Educação", Icon: "book", Color: "#F59E0B", Type: schemas.CategoryTypeVariable},
		{Name: "Lazer", Icon: "music", Color: "#6366F1", Type: schemas.CategoryTypeVariable},
	}

	for index, item := range defaults {
		category := schemas.Category{
			UserID:   userID,
			Name:     item.Name,
			Icon:     item.Icon,
			ColorHex: item.Color,
			Type:     item.Type,
			Order:    index,
			Active:   true,
		}
		if err := tx.Create(&category).Error; err != nil {
			return err
		}
	}
	return nil
}
