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
	GetOrders(ctx context.Context, userID string) ([]util.GetOrderResponse, error)
	GetUserBalance(ctx context.Context, userID string) (util.GetUserBalanceResponse, error)
	Withdraw(ctx context.Context, sum float64, orderID, userID string) error
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

func (s *dbStorage) GetOrders(ctx context.Context, userID string) ([]util.GetOrderResponse, error) {
	orders := []util.GetOrderResponse{}

	rows, err := s.dbpool.Query(ctx, "SELECT number, status, accrual, updated_at from orders where user_id = $1", userID)
	if err != nil {
		return orders, err
	}

	defer rows.Close()

	for rows.Next() {
		var order util.GetOrderResponse

		var status int
		err = rows.Scan(&order.Number, &status, &order.Accrual, &order.UpdatedAt)
		if err != nil {
			return orders, err
		}
		order.Status = OrderStatusesMap[status]

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	return orders, nil
}

func (s *dbStorage) GetUserBalance(ctx context.Context, userID string) (util.GetUserBalanceResponse, error) {
	var userBalance util.GetUserBalanceResponse

	err := s.dbpool.QueryRow(ctx, "SELECT balance from users WHERE id = $1", userID).Scan(&userBalance.Current)
	if err != nil {
		return userBalance, err
	}

	err = s.dbpool.QueryRow(ctx, "SELECT SUM(amount) from withdrawal WHERE user_id = $1", userID).Scan(&userBalance.Withdrawn)
	if err != nil {
		return userBalance, err
	}

	return userBalance, nil
}

func (s *dbStorage) Withdraw(ctx context.Context, sum float64, userID, orderID string) error {
	var userBalance float64
	err := s.dbpool.QueryRow(ctx, "SELECT balance from users WHERE id = $1", userID).Scan(&userBalance)
	if err != nil {
		return err
	}

	amountAfterWithdrawal := userBalance - sum
	if amountAfterWithdrawal < 0 {
		return util.ErrInsufficientAmount
	}

	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "UPDATE users SET balance = $1 WHERE id = $2", amountAfterWithdrawal, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "INSERT INTO withdrawal (user_id, order_id, amount) values ($1, $2,$3)", userID, orderID, sum); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
