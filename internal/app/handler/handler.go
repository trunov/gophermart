package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"github.com/trunov/gophermart/internal/app/luhn"
	"github.com/trunov/gophermart/internal/app/postgres"
	"github.com/trunov/gophermart/internal/app/util"
)

var tokenAuth *jwtauth.JWTAuth

func init() {
	secret := util.Getenv("secret", "secretKey")
	tokenAuth = jwtauth.New("HS256", []byte(secret), nil)
}

type RegRequest struct {
	Login    string `validate:"required" json:"login"`
	Password string `validate:"required" json:"password"`
}

type LoginRequest struct {
	Login    string `validate:"required" json:"login"`
	Password string `validate:"required" json:"password"`
}

type WithdrawRequest struct {
	Order string  `validate:"required" json:"order"`
	Sum   float64 `validate:"required" json:"sum"`
}

type Handler struct {
	dbStorage postgres.DBStorager
	logger    zerolog.Logger
}

func NewHandler(dbStorage postgres.DBStorager, logger zerolog.Logger) *Handler {
	return &Handler{dbStorage: dbStorage, logger: logger}
}

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var regRequest RegRequest

	if err := json.NewDecoder(r.Body).Decode(&regRequest); err != nil {
		h.logger.Err(err).Msg("Could not deserialize JSON into struct")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := validator.New().Struct(regRequest)
	if err != nil {
		h.logger.Err(err).Msg("Validation during registration error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, err := h.dbStorage.RegisterUser(ctx, regRequest.Login, regRequest.Password)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			h.logger.Info().Msg("User tried to register but this login is already in use: " + regRequest.Login)
			w.WriteHeader(http.StatusConflict)
			return
		}

		h.logger.Err(err).Msg("Something is wrong with db")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	token, err := util.GenerateToken(tokenAuth, userID)
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with jwt token generation")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(10 * time.Minute),
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    token,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) AuthenticateUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var loginReq LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		h.logger.Err(err).Msg("Could not deserialize JSON into struct")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := validator.New().Struct(loginReq)
	if err != nil {
		h.logger.Err(err).Msg("Validation during registration error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.dbStorage.AuthenticateUser(ctx, tokenAuth, loginReq.Login, loginReq.Password)
	if err != nil {
		if err == util.ErrIncorrectPassword || err == pgx.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.logger.Err(err).Msg("Something went wrong")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(10 * time.Minute),
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    token,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	orderNumber, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Err(err).Msg("CreateOrder. Something went wrong while reading body.")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	ok := luhn.Valid(string(orderNumber))
	if !ok {
		h.logger.Warn().Msg("CreateOrder. Order number is not valid")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	id, ok := token.Get("id")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	err = h.dbStorage.CreateOrder(ctx, string(orderNumber), id.(string))
	if err != nil {
		if err == util.ErrOrderLoadedByOtherUser {
			w.WriteHeader(http.StatusConflict)
			return
		}

		if err == util.ErrOrderLoadedByUser {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusAccepted)
}

// TODO: fix error when user balance is null
func (h *Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	userID, ok := token.Get("id")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	balance, err := h.dbStorage.GetUserBalance(ctx, userID.(string))
	if err != nil {
		h.logger.Err(err).Msg("GetUserBalance. Database failed to process sequel.")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(balance); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	userID, ok := token.Get("id")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	var wReq WithdrawRequest

	if err := json.NewDecoder(r.Body).Decode(&wReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.New().Struct(wReq)
	if err != nil {
		h.logger.Err(err).Msg("Validation during registration error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ok = luhn.Valid(wReq.Order)
	if !ok {
		h.logger.Warn().Msg("Withdraw. Order number is not valid")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err = h.dbStorage.Withdraw(ctx, wReq.Sum, userID.(string), wReq.Order)

	if err != nil {
		if err == util.ErrInsufficientAmount {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		h.logger.Err(err).Msg("Something went wrong")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	userID, ok := token.Get("id")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	orders, err := h.dbStorage.GetOrdersByUser(ctx, userID.(string))
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		h.logger.Err(err).Msg("Get orders. Something went wrong with database.")
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	userID, ok := token.Get("id")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	withdrawals, err := h.dbStorage.GetUserWithdrawals(ctx, userID.(string))
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		h.logger.Err(err).Msg("Get orders. Something went wrong with database.")
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if err := h.dbStorage.Ping(ctx); err != nil {
		h.logger.Err(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Info().Msg("Ping to db was successful")
	w.WriteHeader(http.StatusOK)
}

func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		// TODO: refactor; group into /api/user and /balance
		r.Post("/api/user/orders", h.CreateOrder)
		r.Get("/api/user/orders", h.GetOrders)
		r.Get("/api/user/balance", h.GetUserBalance)
		r.Post("/api/user/balance/withdraw", h.Withdraw)
		r.Get("/api/user/withdrawals", h.GetWithdrawals)
	})

	r.Get("/ping", h.PingDB)
	r.Post("/api/user/register", h.RegisterUser)
	r.Post("/api/user/login", h.AuthenticateUser)

	return r
}
