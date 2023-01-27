package postgres

import (
	"context"
	"database/sql"

	"github.com/go-chi/jwtauth"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/gophermart/internal/app/util"
)

type DBStorager interface {
	Ping(ctx context.Context) error
	RegisterUser(ctx context.Context, login, password string) (string, error)
	AuthenticateUser(ctx context.Context, tokenAuth *jwtauth.JWTAuth, login, password string) (string, error)
	CreateOrder(ctx context.Context, number, userID string) error
	GetOrdersByUser(ctx context.Context, userID string) ([]util.GetOrderResponse, error)
	GetOrders(ctx context.Context) ([]util.GetOrderResponse, error)
	UpdateOrder(ctx context.Context, orderNumber string, orderStatus int, accrual float64) error
	GetUserBalance(ctx context.Context, userID string) (util.GetUserBalanceResponse, error)
	Withdraw(ctx context.Context, sum float64, orderID, userID string) error
	GetUserWithdrawals(ctx context.Context, userID string) ([]util.GetUserWithdrawalResponse, error)
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

func (s *dbStorage) UpdateOrder(ctx context.Context, orderNumber string, orderStatus int, accrual float64) error {
	if accrual != 0 {
		var userID string
		err := s.dbpool.QueryRow(ctx, "SELECT user_id from orders WHERE number = $1", orderNumber).Scan(&userID)
		if err != nil {
			return err
		}

		tx, err := s.dbpool.Begin(ctx)
		if err != nil {
			return err
		}

		defer tx.Rollback(ctx)

		if _, err := tx.Exec(ctx, "UPDATE orders SET status = $1, accrual = $2 WHERE number = $3", orderStatus, accrual, orderNumber); err != nil {
			return err
		}

		if _, err = tx.Exec(ctx, "UPDATE users SET balance = balance + $1 WHERE id = $2", accrual, userID); err != nil {
			return err
		}

		err = tx.Commit(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	if _, err := s.dbpool.Exec(ctx, "UPDATE orders SET status = $1 WHERE number = $2", orderStatus, orderNumber); err != nil {
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
	var queryUserID string
	err := s.dbpool.QueryRow(ctx, "SELECT user_id from orders WHERE number = $1", number).Scan(&queryUserID)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	if queryUserID == userID {
		return util.ErrOrderLoadedByUser
	}

	if queryUserID != "" {
		return util.ErrOrderLoadedByOtherUser
	}

	_, err = s.dbpool.Exec(ctx, "INSERT INTO orders (number, user_id, status) values ($1, $2, $3)", number, userID, 1)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) GetOrdersByUser(ctx context.Context, userID string) ([]util.GetOrderResponse, error) {
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
		order.Status = util.OrderStatusesMap[status]

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	return orders, nil
}

func (s *dbStorage) GetOrders(ctx context.Context) ([]util.GetOrderResponse, error) {
	orders := []util.GetOrderResponse{}

	rows, err := s.dbpool.Query(ctx, "SELECT number, status, accrual, updated_at from orders")
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
		order.Status = util.OrderStatusesMap[status]

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

	var nullFloat sql.NullFloat64

	err = s.dbpool.QueryRow(ctx, "SELECT SUM(amount) from withdrawal WHERE user_id = $1", userID).Scan(&nullFloat)
	if err != nil {
		return userBalance, err
	}

	if nullFloat.Valid {
		userBalance.Withdrawn = nullFloat.Float64
	} else {
		userBalance.Withdrawn = 0
	}

	return userBalance, nil
}

func (s *dbStorage) GetUserWithdrawals(ctx context.Context, userID string) ([]util.GetUserWithdrawalResponse, error) {
	var userWithdrawals []util.GetUserWithdrawalResponse

	rows, err := s.dbpool.Query(ctx, "SELECT order_id, amount, processed_at from withdrawal WHERE user_id = $1", userID)
	if err != nil {
		return userWithdrawals, err
	}

	defer rows.Close()

	for rows.Next() {
		var userWithdrawal util.GetUserWithdrawalResponse
		err = rows.Scan(&userWithdrawal.Order, &userWithdrawal.Sum, &userWithdrawal.ProcessedAt)
		if err != nil {
			return userWithdrawals, err
		}

		userWithdrawals = append(userWithdrawals, userWithdrawal)
	}

	return userWithdrawals, nil
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
