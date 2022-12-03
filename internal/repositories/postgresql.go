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

	q := `CREATE TABLE if not exists users (
    		id BIGINT PRIMARY KEY generated always as identity,
    		login VARCHAR(200) NOT NULL unique,
    		hashed_password VARCHAR(200) NOT NULL,
    		balance DOUBLE PRECISION
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
	hashPassword := p.hashPassword(u.Password)
	q := `INSERT INTO users (login, hashed_password)
	   						VALUES ($1, $2)`
	if _, err := p.client.Exec(context.Background(), q, u.Login, hashPassword); err != nil {
		p.logger.LogErr(err, "Failure to insert object into table")
		return err
	}
	return nil
}

func (p *PGSStore) GetOrder(login string) ([]storage.Orders, error) {
	q := `SELECT users.id FROM users WHERE login = $1`
	id := ""
	if err := p.client.QueryRow(context.Background(), q, login).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return nil, fmt.Errorf("wrong login %s", login)
		}
		p.logger.LogErr(err, "")
		return nil, err
	}

	var orders []storage.Orders
	q = `SELECT orders.orders, status, accrual, uploaded_at FROM orders WHERE orders.user_id = $1`
	rows, err := p.client.Query(context.Background(), q, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return nil, fmt.Errorf("wrong login %s", login)
		}
		p.logger.LogErr(err, "")
		return nil, err
	}
	for rows.Next() {
		var order storage.Orders
		err = rows.Scan(&order.Order, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			fmt.Println("nen")
			fmt.Println(err)
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (p *PGSStore) CollectOrder(login string, order int) (int, error) {
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

	//сверяем id того, кто загрузил ордер с id того, кто пытается загрузить
	if id != userIDFromDB {
		//если id не совпадают, то возвращаем 409 — номер заказа уже был загружен другим пользователем
		return 409, fmt.Errorf("order is already upload from another user")
	} else if id == userIDFromDB {
		//если id совпадают, то возвращаем 200 — номер заказа уже был загружен этим пользователем
		return 200, fmt.Errorf("order is already upload")
	}
	return 0, nil
}

func (p *PGSStore) Login(u *storage.AcceptUser) error {
	var password string
	hashPassword := p.hashPassword(u.Password)
	q := `SELECT hashed_password FROM users WHERE login = $1`
	if err := p.client.QueryRow(context.Background(), q, u.Login).Scan(&password); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p.logger.LogErr(err, "Failure to select object from table")
			return fmt.Errorf("wrong login %s", u.Login)
		}
		p.logger.LogErr(err, "Wrong login")
		return fmt.Errorf("wrong login %s", u.Login)
	}
	if hashPassword != password {
		return fmt.Errorf("wrong password to %s", u.Login)
	}
	return nil
}

//func (p *PGSStore) GetMetric(metric storage.Metrics) (storage.Metrics, error) {
//	var m storage.Metrics
//	if metric.MType == "counter" || metric.MType == "gauge" {
//		q := `SELECT id, mType, delta, value, hash FROM metrics WHERE id = $1 AND mType = $2`
//		if err := p.client.QueryRow(context.Background(), q, metric.ID, metric.MType).Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash); err != nil {
//			if errors.Is(err, pgx.ErrNoRows) {
//				p.logger.LogErr(err, "Failure to select object from table")
//				return m, err
//			}
//			p.logger.LogErr(err, "wrong metric")
//			return m, fmt.Errorf("missing metric %s", metric.ID)
//		}
//		m.Hash, _ = hashing(p.Hash, &m, &p.logger)
//		return m, nil
//	} else {
//		p.logger.LogErr(fmt.Errorf("wrong type"), "wrong type")
//		return m, fmt.Errorf("wrong type")
//	}
//}
//
//func (p *PGSStore) GetAll() (string, error) {
//	var metrics []storage.Metrics
//	q := `SELECT id, mType, delta, value, hash FROM metrics`
//	rows, err := p.client.Query(context.Background(), q)
//	if err != nil {
//		p.logger.LogErr(err, "Failure to select object from table")
//		return "", err
//	}
//	for rows.Next() {
//		var m storage.Metrics
//		err = rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash)
//		if err != nil {
//			p.logger.LogErr(err, "Failure to convert object from table")
//			return "", err
//		}
//		metrics = append(metrics, m)
//	}
//
//	result := ""
//	for _, f := range metrics {
//		if f.MType == "gauge" {
//			if f.Value != nil {
//				result += fmt.Sprintf("%s : %f\n", f.ID, *f.Value)
//			}
//			continue
//		} else if f.MType == "counter" {
//			result += fmt.Sprintf("%s : %d\n", f.ID, *f.Delta)
//		}
//	}
//	return result, nil
//}
//
//func (p *PGSStore) CollectMetric(m storage.Metrics) error {
//
//	if m.Hash != "" {
//		_, ok := hashing(p.Hash, &m, &p.logger)
//		if !ok {
//			err := fmt.Errorf("hash is wrong")
//			p.logger.LogErr(err, "hash is wrong")
//			return err
//		}
//	}
//	if m.MType == "counter" || m.MType == "gauge" {
//		q := `INSERT INTO metrics (id, mType, delta, value, hash)
//    						VALUES ($1, $2, $3, $4, $5)
//							ON CONFLICT (id) DO UPDATE SET
//    							delta = metrics.delta + EXCLUDED.delta,
//    							value = $4,
//    							hash = EXCLUDED.hash`
//		if _, err := p.client.Exec(context.Background(), q, m.ID, m.MType, m.Delta, m.Value, m.Hash); err != nil {
//			p.logger.LogErr(err, "Failure to insert object into table")
//			return err
//		}
//	} else {
//		p.logger.LogErr(fmt.Errorf("wrong type"), "wrong type")
//		return fmt.Errorf("wrong type")
//	}
//	return nil
//}
//
//func (p *PGSStore) CollectOrChangeGauge(id string, value float64) error {
//	mType := "gauge"
//	hash := ""
//	q := `INSERT INTO metrics (id, mType, value, hash)
//    						VALUES ($1, $2, $3, $4)
//							ON CONFLICT (id) DO UPDATE SET
//    							value = EXCLUDED.value,
//							    mType = EXCLUDED.mType`
//	if _, err := p.client.Exec(context.Background(), q, id, mType, value, hash); err != nil {
//		p.logger.LogErr(err, "Failure to insert object into table")
//		return err
//	}
//	return nil
//}
//
//func (p *PGSStore) CollectOrIncreaseCounter(id string, delta int64) error {
//	mType := "counter"
//	hash := ""
//	q := `INSERT INTO metrics (id, mType, delta, hash)
//    						VALUES ($1, $2, $3, $4)
//							ON CONFLICT (id) DO UPDATE SET
//    							delta = metrics.delta + EXCLUDED.delta,
// 								mType = EXCLUDED.mType`
//	if _, err := p.client.Exec(context.Background(), q, id, mType, delta, hash); err != nil {
//		p.logger.LogErr(err, "Failure to insert object into table")
//		return err
//	}
//	return nil
//}
//
//func (p *PGSStore) GetGauge(id string) (float64, error) {
//	var value float64
//	mType := "gauge"
//	q := `SELECT value FROM metrics WHERE id = $1 AND mType = $2`
//	if err := p.client.QueryRow(context.Background(), q, id, mType).Scan(&value); err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			p.logger.LogErr(err, "Failure to select object from table")
//			return 0, err
//		}
//		p.logger.LogErr(err, "Wrong metric")
//		return 0, fmt.Errorf("missing metric %s", id)
//	}
//	return value, nil
//}
//
//func (p *PGSStore) GetCounter(id string) (int64, error) {
//	var delta int64
//	mType := "counter"
//	q := `SELECT delta FROM metrics WHERE id = $1 AND mType = $2`
//	if err := p.client.QueryRow(context.Background(), q, id, mType).Scan(&delta); err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			p.logger.LogErr(err, "Failure to select object from table")
//			return 0, err
//		}
//		p.logger.LogErr(err, "Wrong metric")
//		return 0, fmt.Errorf("missing metric %s", id)
//	}
//	return delta, nil
//}
//
//func (p *PGSStore) PingClient() error {
//	return p.client.Ping(context.Background())
//}
//
//func (p *PGSStore) CollectMetrics(metrics []storage.Metrics) error {
//
//	tx, err := p.client.BeginTx(context.Background(), pgx.TxOptions{})
//	if err != nil {
//		p.logger.LogErr(err, "failed to begin transaction")
//		return err
//	}
//	defer tx.Rollback(context.Background())
//	q := `INSERT INTO metrics (id, mType, delta, value, hash)
//    						VALUES ($1, $2, $3, $4, $5)
//							ON CONFLICT (id) DO UPDATE SET
//    							delta = metrics.delta + EXCLUDED.delta,
//    							value = $4,
//    							hash = EXCLUDED.hash`
//
//	for _, m := range metrics {
//		if _, err = tx.Exec(context.Background(), q, m.ID, m.MType, m.Delta, m.Value, m.Hash); err != nil {
//			p.logger.LogErr(err, "failed transaction")
//			return err
//		}
//	}
//
//	return tx.Commit(context.Background())
//}

func (p *PGSStore) hashPassword(pass string) string {
	h := hmac.New(sha256.New, []byte("password"))
	h.Write([]byte(pass))
	return fmt.Sprintf("%x", h.Sum(nil))
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
func (p *PGSStore) Valid(order int) bool {
	return (order%10+checksum(order/10))%10 == 0
}
