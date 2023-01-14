package util

import (
	"errors"
	"os"

	"github.com/go-chi/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

var ErrIncorrectPassword error = errors.New("password is incorrect")

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GenerateToken(tokenAuth *jwtauth.JWTAuth, login string) (string, error) {
	_, token, err := tokenAuth.Encode(map[string]interface{}{"login": login})
	if err != nil {
		return "", err
	}

	return token, nil
}
