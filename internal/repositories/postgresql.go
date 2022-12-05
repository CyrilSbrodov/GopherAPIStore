package repositories

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"

	//"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
	"github.com/CyrilSbrodov/GopherAPIStore/pkg/client/postgresql"
)

type PGSStore struct {
	client postgresql.Client
	logger loggers.Logger
}

func createTable(ctx context.Context, client postgresql.Client, logger *loggers.Logger) error {
	tx, err := client.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		logger.LogErr(err, "failed to begin transaction")
		return err
	}
	defer tx.Rollback(ctx)

	//создание таблиц
	q := `CREATE TABLE if not exists users (
    		id BIGINT PRIMARY KEY generated always as identity,
    		login VARCHAR(200) NOT NULL unique,
    		hashed_password VARCHAR(200) NOT NULL,
    		balance_current DOUBLE PRECISION,
            balance_withdrawn DOUBLE PRECISION
		);
		CREATE UNIQUE INDEX if not exists users_login_uindex on users (login);
		CREATE TABLE if not exists orders (
    		user_id BIGINT,
    		FOREIGN KEY (user_id) REFERENCES users(id),
    		orders BIGINT PRIMARY KEY NOT NULL,
    		status VARCHAR(200),
    		accrual DOUBLE PRECISION,
    		uploaded_at TIMESTAMPTZ(0)
	);`

	_, err = tx.Exec(ctx, q)
	if err != nil {
		logger.LogErr(err, "failed to create table")
		return err
	}
	return tx.Commit(ctx)
}

func NewPGSStore(client postgresql.Client, cfg *config.ServerConfig, logger *loggers.Logger) (*PGSStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := createTable(ctx, client, logger); err != nil {
		logger.LogErr(err, "failed to create table")
		return nil, err
	}
	return &PGSStore{
		client: client,
	}, nil
}

func (p *PGSStore) Register(u *storage.AcceptUser) error {
	//хэширование пароля
	hashPassword := p.hashPassword(u.Password)
	//добавление пользователя в базу
	q := `INSERT INTO users (login, hashed_password)
	   						VALUES ($1, $2)`
	if _, err := p.client.Exec(context.Background(), q, u.Login, hashPassword); err != nil {
		p.logger.LogErr(err, "Failure to insert object into table")
		return err
	}
	return nil
}

func (p *PGSStore) Login(u *storage.AcceptUser) error {
	var password string
	//хэширование полученного пароля
	hashPassword := p.hashPassword(u.Password)
	//получение хэш пароля, хранящегося в базе
	q := `SELECT hashed_password FROM users WHERE login = $1`
	if err := p.client.QueryRow(context.Background(), q, u.Login).Scan(&password); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return fmt.Errorf("wrong login %s", u.Login)
		}
		p.logger.LogErr(err, "Wrong login")
		return fmt.Errorf("wrong login %s", u.Login)
	}
	//сравнение пхэш пароля полученного и хэш пароля из базы
	if hashPassword != password {
		return fmt.Errorf("wrong password to %s", u.Login)
	}
	return nil
}

func (p *PGSStore) CollectOrder(login string, order int) (int, error) {
	//проверка номера ордера на валидность по алгоритсу Луны
	if !p.Valid(order) {
		return 422, fmt.Errorf("wrong orders number %v", order)
	}
	id := ""
	//получаем id пользователя по логину
	q := `SELECT users.id FROM users WHERE login = $1`
	if err := p.client.QueryRow(context.Background(), q, login).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return 400, fmt.Errorf("wrong login %s", login)
		}
		p.logger.LogErr(err, "")
		return 500, err
	}
	userIDFromDB := ""
	//проверка есть ли ордер в базе, если ордер есть, то получаем id того, кто его загрузил
	q = `SELECT orders.user_id FROM orders WHERE orders.orders = $1`
	if err := p.client.QueryRow(context.Background(), q, order).Scan(&userIDFromDB); err != nil {
		//если ордера нет, то заносим его в базу
		if errors.Is(err, pgx.ErrNoRows) {
			q = `INSERT INTO orders (user_id, orders, status, accrual, uploaded_at) VALUES ($1, $2, $3, $4, current_timestamp)`
			if _, err := p.client.Exec(context.Background(), q, id, order, "NEW", 0); err != nil {
				p.logger.LogErr(err, "Failure to insert object into table")
				return 500, err
			}
			//возвращаем 202 — новый номер заказа принят в обработку
			return 202, nil
		}
		p.logger.LogErr(err, "")
		return 500, err
	}

	//сверяем id того, кто загрузил ордер с id тем, кто пытается загрузить
	if id != userIDFromDB {
		//если id не совпадают, то возвращаем 409 — номер заказа уже был загружен другим пользователем
		return 409, fmt.Errorf("order is already upload from another user")
	} else if id == userIDFromDB {
		//если id совпадают, то возвращаем 200 — номер заказа уже был загружен этим пользователем
		return 200, fmt.Errorf("order is already upload")
	}
	return 0, nil
}

func (p *PGSStore) GetOrder(login string) (int, []storage.Orders, error) {
	id := ""
	//поиск id пользователя по логину
	q := `SELECT users.id FROM users WHERE login = $1`
	if err := p.client.QueryRow(context.Background(), q, login).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return 204, nil, fmt.Errorf("wrong login %s", login)
		}
		p.logger.LogErr(err, "")
		return 500, nil, err
	}

	var orders []storage.Orders
	//получение списка ордеров по id пользователя
	q = `SELECT orders.orders, status, accrual, uploaded_at FROM orders WHERE orders.user_id = $1`
	rows, err := p.client.Query(context.Background(), q, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return 204, nil, fmt.Errorf("no one orders")
		}
		p.logger.LogErr(err, "")
		return 500, nil, err
	}
	//добавление всех ордеров в слайс
	for rows.Next() {
		var order storage.Orders
		err = rows.Scan(&order.Order, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			p.logger.LogErr(err, "Failure to scan object from table")
			return 500, nil, err
		}
		orders = append(orders, order)
	}
	return 200, orders, nil
}

func (p *PGSStore) GetBalance(login string) (storage.Balance, error) {
	var balance storage.Balance
	//получение баланса из базы по логину пользователя
	q := `SELECT balance_current, balance_withdrawn FROM users WHERE login = $1`
	if err := p.client.QueryRow(context.Background(), q, login).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return balance, fmt.Errorf("wrong login %s", login)
		}
		p.logger.LogErr(err, "Wrong login")
		return balance, err
	}
	return balance, nil
}

func (p *PGSStore) GetAllOrders() ([]storage.Orders, error) {

	var orders []storage.Orders
	q := `SELECT orders.orders FROM orders WHERE status = 'REGISTERED' or status = 'PROCESSING' or status = 'NEW'`
	rows, err := p.client.Query(context.Background(), q)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return nil, fmt.Errorf("no one orders")
		}
		p.logger.LogErr(err, "")
		return nil, err
	}
	//добавление всех ордеров в слайс
	for rows.Next() {
		var order storage.Orders
		err = rows.Scan(&order.Order)
		if err != nil {
			p.logger.LogErr(err, "Failure to scan object from table")
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (p *PGSStore) UpdateOrders(orders []storage.Orders) error {
	tx, err := p.client.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		p.logger.LogErr(err, "failed to begin transaction")
		return err
	}
	defer tx.Rollback(context.Background())
	q := `UPDATE orders SET status = $1, accrual = $2 WHERE orders.orders = $3`
	for _, o := range orders {
		if _, err = tx.Exec(context.Background(), q, o.Status, o.Accrual, o.Order); err != nil {

			p.logger.LogErr(err, "failed transaction")
			return err
		}
	}
	return tx.Commit(context.Background())
}

func (p *PGSStore) hashPassword(pass string) string {
	h := hmac.New(sha256.New, []byte("password"))
	h.Write([]byte(pass))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p *PGSStore) Valid(order int) bool {
	return (order%10+checksum(order/10))%10 == 0
}

func checksum(order int) int {
	var luhn int
	for i := 0; order > 0; i++ {
		cur := order % 10
		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}
		luhn += cur
		order = order / 10
	}
	return luhn % 10
}
