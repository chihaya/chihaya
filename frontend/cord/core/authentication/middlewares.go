package authentication

import (
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	request "github.com/dgrijalva/jwt-go/request"

	"github.com/labstack/echo"
	"go.uber.org/zap"

	"github.com/ProtocolONE/chihaya/frontend/cord/models"
)

func RequireTokenAuthentication(next echo.HandlerFunc) echo.HandlerFunc {

	return requireTokenAuthentication(next, false)
}

func RequireRefreshTokenAuthentication(next echo.HandlerFunc) echo.HandlerFunc {

	return requireTokenAuthentication(next, true)
}

func requireTokenAuthentication(next echo.HandlerFunc, refreshToken bool) echo.HandlerFunc {

	return func(context echo.Context) error {

		authBackend := InitJWTAuthenticationBackend()
		token, err := request.ParseFromRequest(context.Request(), request.OAuth2Extractor, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			} else {
				return authBackend.PublicKey, nil
			}
		})

		if err != nil || !token.Valid || authBackend.IsInBlacklist(context.Request().Header.Get("Authorization")) {

			if err != nil {
				zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", err.Error()))
				return context.JSON(http.StatusUnauthorized, models.Error{Code: models.ErrorUnauthorized, Message: "requireTokenAuthentication failed, error: " + err.Error()})

			} else {
				zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", "Authorization failed"))
				return context.JSON(http.StatusUnauthorized, models.Error{Code: models.ErrorUnauthorized, Message: "requireTokenAuthentication failed, error: Authorization failed"})
			}
		}

		rem := authBackend.GetTokenRemainingValidity(token)
		if rem <= 0 {

			zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", "Token is expired"))
			return context.JSON(http.StatusUnauthorized, models.Error{Code: models.ErrorTokenExpired, Message: "Token is expired"})
		}

		claims := token.Claims.(jwt.MapClaims)

		if refreshToken {

			refresh, ok := claims["refresh"].(bool)
			if !ok || !refresh {

				zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", "Invalid refresh token"))
				return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorInvalidToken, Message: "Invalid refresh token"})
			}

		} else {

			access, ok := claims["access"].(bool)
			if !ok || !access {

				zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", "Invalid access token"))
				return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorInvalidToken, Message: "Invalid access token"})
			}
		}

		clientID, ok := claims["client_id"].(string)
		if !ok || clientID == "" {

			zap.S().Errorw("requireTokenAuthentication failed", zap.String("error", "Invalid token"))
			return context.JSON(http.StatusBadRequest, models.Error{Code: models.ErrorInvalidToken, Message: "Invalid token"})
		}

		context.Request().Header.Set("ClientID", clientID)
		return next(context)
	}
}
