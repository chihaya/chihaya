package authentication

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/ProtocolONE/chihaya/frontend/cord/config"
	"github.com/ProtocolONE/chihaya/frontend/cord/database"
	"github.com/ProtocolONE/chihaya/frontend/cord/models"
	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"os"
	"time"
)

type JWTAuthenticationBackend struct {
	privateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

const (
	tokenDuration = 72
	expireOffset  = 3600
)

var authBackendInstance *JWTAuthenticationBackend

func InitJWTAuthenticationBackend() *JWTAuthenticationBackend {

	if authBackendInstance == nil {
		authBackendInstance = &JWTAuthenticationBackend{
			privateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}
	return authBackendInstance
}

func (backend *JWTAuthenticationBackend) GenerateToken(clientID string, userUUID string, refreshToken bool) (string, error) {

	token := jwt.New(jwt.SigningMethodRS512)
	cfg := config.Get().Service

	if refreshToken {

		token.Claims = jwt.MapClaims{
			"exp":       time.Now().Add(time.Hour * time.Duration(cfg.JwtRefExpDelta)).Unix(),
			"iat":       time.Now().Unix(),
			"sub":       userUUID,
			"client_id": clientID,
			"refresh":   true,
		}
	} else {

		token.Claims = jwt.MapClaims{
			"exp":       time.Now().Add(time.Second * time.Duration(cfg.JwtExpDelta)).Unix(),
			"iat":       time.Now().Unix(),
			"sub":       userUUID,
			"client_id": clientID,
			"access":    true,
		}
	}

	tokenString, err := token.SignedString(backend.privateKey)
	if err != nil {
		return "", fmt.Errorf("Cannot generate token, error: %s", err)
	}

	return tokenString, nil
}

func (backend *JWTAuthenticationBackend) Authenticate(user *models.Authorization) bool {

	manager := database.NewUserManager()
	users, err := manager.FindByName(user.Username)
	if err != nil {
		return false
	}

	return len(users) == 1 && user.Username == users[0].Username && bcrypt.CompareHashAndPassword([]byte(users[0].Password), []byte(user.Password)) == nil
}

func (backend *JWTAuthenticationBackend) GetTokenRemainingValidity(token *jwt.Token) int64 {

	claims := token.Claims.(jwt.MapClaims)
	exp, _ := claims["exp"]

	if validity, ok := exp.(float64); ok {

		return int64(validity) - time.Now().Unix()
	}

	return 0
}

func (backend *JWTAuthenticationBackend) Logout(tokenStr string, token *jwt.Token) error {

	return nil
}

func (backend *JWTAuthenticationBackend) IsInBlacklist(tokenStr string) bool {

	return false
}

func getPrivateKey() *rsa.PrivateKey {

	cfg := config.Get().Service
	privateKeyFile, err := os.Open(cfg.PrivateKeyPath)
	if err != nil {
		panic(fmt.Sprintf("Cannot open file \"%s\"", cfg.PrivateKeyPath))
	}
	pemfileinfo, _ := privateKeyFile.Stat()
	size := pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	privateKeyFile.Close()

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	return privateKeyImported
}

func getPublicKey() *rsa.PublicKey {

	cfg := config.Get().Service
	publicKeyFile, err := os.Open(cfg.PublicKeyPath)
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := publicKeyFile.Stat()
	size := pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	publicKeyFile.Close()

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}

	return rsaPub
}
