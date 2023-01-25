package util

import (
	"errors"
	"os"
	"time"

	"github.com/go-chi/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

type GetOrderResponse struct {
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	Accrual   float64   `json:"accrual,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetUserBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type GetUserWithdrawalResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

var ErrIncorrectPassword error = errors.New("password is incorrect")
var ErrInsufficientAmount error = errors.New("insufficient amount of balance")

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

func GenerateToken(tokenAuth *jwtauth.JWTAuth, id string) (string, error) {
	_, token, err := tokenAuth.Encode(map[string]interface{}{"id": id})

	if err != nil {
		return "", err
	}

	return token, nil
}
