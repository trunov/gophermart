package util

import (
	"errors"
	"os"
	"time"

	"github.com/go-chi/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

var OrderStatusesMap = map[int]string{
	1: "NEW",
	2: "PROCESSING",
	3: "INVALID",
	4: "PROCESSED",
}

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
var ErrNoKeyPresented error = errors.New("key was not found in the map")
var ErrOrderLoadedByOtherUser error = errors.New("order already loaded by other user")
var ErrOrderLoadedByUser error = errors.New("order already loaded by user")

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

func FindKeyByValue(value string) (int, error) {
	for k, v := range OrderStatusesMap {
		if v == value {
			return k, nil
		}
	}
	return 0, ErrNoKeyPresented
}
