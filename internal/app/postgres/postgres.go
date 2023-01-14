package postgres

import (
	"context"

	"github.com/go-chi/jwtauth"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/gophermart/internal/app/util"
)

type DbStorager interface {
	Ping(ctx context.Context) error
	RegisterUser(ctx context.Context, login, password string) error
	AuthenticateUser(ctx context.Context, tokenAuth *jwtauth.JWTAuth, login, password string) (string, error)
}

type dbStorage struct {
	dbpool *pgxpool.Pool
}

func NewDBStorage(conn *pgxpool.Pool) *dbStorage {
	return &dbStorage{dbpool: conn}
}

func (s *dbStorage) Ping(ctx context.Context) error {
	err := s.dbpool.Ping(ctx)

	if err != nil {
		return err
	}
	return nil
}

func (s *dbStorage) RegisterUser(ctx context.Context, login, password string) error {
	hp, err := util.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.dbpool.Exec(ctx, "INSERT INTO users (login, password) values ($1, $2)", login, hp)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) AuthenticateUser(ctx context.Context, tokenAuth *jwtauth.JWTAuth, login, password string) (string, error) {
	var hash string

	err := s.dbpool.QueryRow(ctx, "SELECT password from users WHERE login = $1", login).Scan(&hash)
	if err != nil {
		return "", err
	}

	ok := util.CheckPasswordHash(password, hash)

	if !ok {
		return "", util.ErrIncorrectPassword
	}

	token, err := util.GenerateToken(tokenAuth, login)
	if err != nil {
		return "", err
	}

	return token, nil
}
