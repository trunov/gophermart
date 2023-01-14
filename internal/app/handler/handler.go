package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
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

	err = h.dbStorage.RegisterUser(ctx, regRequest.Login, regRequest.Password)
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

	token, err := util.GenerateToken(tokenAuth, regRequest.Login)
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with jwt token generation")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(1 * time.Minute),
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		HttpOnly: true,
		Expires:  time.Now().Add(1 * time.Minute),
		SameSite: http.SameSiteLaxMode,
		Name:     "jwt",
		Value:    token,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	token, _, err := jwtauth.FromContext(r.Context())
	if err != nil {
		h.logger.Err(err).Msg("Something is wrong with reading jwt token from the context")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	login, ok := token.Get("login")
	if !ok {
		h.logger.Error().Msg("Login data can not be found in token")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}

	fmt.Println(login)

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

		r.Get("/ping", h.PingDB)
	})

	r.Post("/api/user/register", h.RegisterUser)
	r.Post("/api/user/login", h.AuthenticateUser)

	return r
}
