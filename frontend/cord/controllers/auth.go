package controllers

import (
	"github.com/ProtocolONE/chihaya/frontend/cord/core/authentication"
	"github.com/ProtocolONE/chihaya/frontend/cord/database"
	"github.com/ProtocolONE/chihaya/frontend/cord/models"

	"fmt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	request "github.com/dgrijalva/jwt-go/request"
	"github.com/pborman/uuid"

	"github.com/labstack/echo"
)

func CreateUser(context echo.Context) error {

	reqUser := &models.Authorization{}
	err := context.Bind(reqUser)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorInvalidJSONFormat, "Invalid JSON format: " + err.Error()})
	}

	manager := database.NewUserManager()
	users, err := manager.FindByName(reqUser.Username)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorReadDataBase, fmt.Sprintf("Cannot read from database, error: %s", err.Error())})
	}

	if len(users) != 0 {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorUserAlreadyExists, fmt.Sprintf("User %s already exists", reqUser.Username)})
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(reqUser.Password), 10)

	err = manager.Insert(&models.User{reqUser.Username, string(hashedPassword)})
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorCreateUser, fmt.Sprintf("Cannot create user %s, error: %s", reqUser.Username, err.Error())})
	}

	zap.S().Infow("Created new user", zap.String("username", reqUser.Username))

	return context.NoContent(http.StatusCreated)
}

func DeleteUser(context echo.Context) error {

	reqUser := &models.Authorization{}
	err := context.Bind(reqUser)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorInvalidJSONFormat, "Invalid JSON format: " + err.Error()})
	}

	manager := database.NewUserManager()
	err = manager.RemoveByName(reqUser.Username)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorDeleteUser, fmt.Sprintf("Cannot delete user %s, error: %s", reqUser.Username, err.Error())})
	}

	zap.S().Infow("Removed user", zap.String("username", reqUser.Username))
	return context.NoContent(http.StatusOK)
}

func Login(context echo.Context) error {

	reqUser := &models.Authorization{}
	err := context.Bind(reqUser)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorInvalidJSONFormat, "Invalid JSON format: " + err.Error()})
	}

	zap.S().Infow("Login", zap.String("username", reqUser.Username), zap.String("password", reqUser.Password))

	authBackend := authentication.InitJWTAuthenticationBackend()
	if !authBackend.Authenticate(reqUser) {
		return context.JSON(http.StatusUnauthorized, models.Error{models.ErrorInvalidUsernameOrPassword, "Invalid username or password"})
	}

	userUUID := uuid.New()
	token, err := authBackend.GenerateToken(reqUser.Username, userUUID, false)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, models.Error{models.ErrorGenToken, fmt.Sprintf("Cannot generate access-token for user %s, error: %s", reqUser.Username, err.Error())})
	}

	userUUID = uuid.New()
	refreshToken, err := authBackend.GenerateToken(reqUser.Username, userUUID, true)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, models.Error{models.ErrorGenToken, fmt.Sprintf("Cannot generate refresh-token for user %s, error: %s", reqUser.Username, err.Error())})
	}

	zap.S().Infow("Login", zap.String("token", token))
	return context.JSON(http.StatusOK, models.AuthToken{reqUser.Username, token, refreshToken})
}

func RefreshToken(context echo.Context) error {

	username := context.Request().Header.Get("ClientID")
	if username == "" {
		return context.NoContent(http.StatusBadRequest)
	}

	authBackend := authentication.InitJWTAuthenticationBackend()

	userUUID := uuid.New()
	token, err := authBackend.GenerateToken(username, userUUID, false)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, models.Error{models.ErrorGenToken, fmt.Sprintf("Cannot generate access-token for user %s, error: %s", username, err.Error())})
	}

	userUUID = uuid.New()
	refreshToken, err := authBackend.GenerateToken(username, userUUID, true)
	if err != nil {
		return context.JSON(http.StatusInternalServerError, models.Error{models.ErrorGenToken, fmt.Sprintf("Cannot generate refresh-token for user %s, error: %s", username, err.Error())})
	}

	return context.JSON(http.StatusOK, models.AuthRefresh{token, refreshToken})
}

func Logout(context echo.Context) error {

	authBackend := authentication.InitJWTAuthenticationBackend()
	tokenRequest, err := request.ParseFromRequest(context.Request(), request.OAuth2Extractor, func(token *jwt.Token) (interface{}, error) {
		return authBackend.PublicKey, nil
	})

	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorLogout, fmt.Sprintf("Logout failed, error: %s", err.Error())})
	}

	tokenString := context.Request().Header.Get("Authorization")

	err = authBackend.Logout(tokenString, tokenRequest)
	if err != nil {
		return context.JSON(http.StatusBadRequest, models.Error{models.ErrorLogout, fmt.Sprintf("Logout failed, error: %s", err.Error())})
	}

	return context.NoContent(http.StatusOK)
}
