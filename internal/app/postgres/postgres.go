package postgres

import (
	"context"

	"github.com/go-chi/jwtauth"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/gophermart/internal/app/util"
)

var OrderStatusesMap = map[int]string{
	1: "NEW",
	2: "PROCCESSING",
	3: "INVALID",
	4: "PROCESSED",
}

type DBStorager interface {
	Ping(ctx context.Context) error
	RegisterUser(ctx context.Context, login, password string) (string, error)
	AuthenticateUser(ctx context.Context, tokenAuth *jwtauth.JWTAuth, login, password string) (string, error)
	CreateOrder(ctx context.Context, number, userID string) error
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

func (s *dbStorage) RegisterUser(ctx context.Context, login, password string) (string, error) {
	hp, err := util.HashPassword(password)
	if err != nil {
		return "", err
	}

	_, err = s.dbpool.Exec(ctx, "INSERT INTO users (login, password) values ($1, $2)", login, hp)
	if err != nil {
		return "", err
	}

	var userID string
	err = s.dbpool.QueryRow(ctx, "SELECT id from users WHERE login = $1", login).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (s *dbStorage) AuthenticateUser(ctx context.Context, tokenAuth *jwtauth.JWTAuth, login, password string) (string, error) {
	var User struct {
		Hash   string
		UserID string
	}

	err := s.dbpool.QueryRow(ctx, "SELECT password, id from users WHERE login = $1", login).Scan(&User.Hash, &User.UserID)
	if err != nil {
		return "", err
	}

	ok := util.CheckPasswordHash(password, User.Hash)

	if !ok {
		return "", util.ErrIncorrectPassword
	}

	token, err := util.GenerateToken(tokenAuth, User.UserID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *dbStorage) CreateOrder(ctx context.Context, number, userID string) error {
	// "NEW" order status corresponds to 1
	_, err := s.dbpool.Exec(ctx, "INSERT INTO orders (number, user_id, status) values ($1, $2, $3)", number, userID, 1)
	if err != nil {
		return err
	}

	return nil
}
