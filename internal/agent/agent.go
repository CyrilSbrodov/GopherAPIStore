package agent

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

type Agent struct {
	storage.Storage
	logger loggers.Logger
	client http.Client
	cfg    config.ServerConfig
}

func NewAgent(storage storage.Storage, logger loggers.Logger, cfg config.ServerConfig) *Agent {
	client := &http.Client{}
	return &Agent{
		storage,
		logger,
		*client,
		cfg,
	}
}

func (a *Agent) Start(ticker time.Ticker) {
	//запуск агента в бесконечном цикле
	for range ticker.C {
		//получение всех заказов с нужным статусом
		orders, err := a.Storage.GetAllOrders()
		if err != nil {
			a.logger.LogErr(err, "")
		}
		//если новых заказов нет, то ждем опять тикер
		if orders == nil {
			continue
		}
		//обновление заказов во внешней системе
		//if err = a.UploadOrders(orders); err != nil {
		//	a.logger.LogErr(err, "")
		//}
		//получение списка обновленных ореров из внешней системы
		updatedOrders, err := a.GetAccrual(orders)
		if err != nil {
			a.logger.LogErr(err, "")
		}
		//обновление заказов в таблице ореров
		if err = a.Storage.UpdateOrders(updatedOrders); err != nil {
			a.logger.LogErr(err, "")
		}
		//обновление суммы вознаграждения в таблице пользователей
		if err = a.Storage.UpdateUserBalance(updatedOrders); err != nil {
			a.logger.LogErr(err, "")
		}
	}
}

func (a *Agent) GetAccrual(orders []storage.Orders) ([]storage.Orders, error) {
	var updatedOrders []storage.Orders
	for _, o := range orders {
		var order storage.Orders
		req, err := http.NewRequest(http.MethodGet, a.cfg.Accrual+"/api/orders/"+o.Order, nil)

		if err != nil {
			a.logger.LogErr(err, "Failed to request")
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			a.logger.LogErr(err, "Failed to do request")
			break
		}
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			a.logger.LogErr(err, "Failed to read body")
			break
		}
		if resp.StatusCode == 429 {
			return updatedOrders, nil
		}
		if resp.StatusCode == 204 {
			continue
		}
		if err := json.Unmarshal(res, &order); err != nil {
			a.logger.LogErr(err, "Failed to read body")
			break
		}
		order.Order = o.Order
		order.UserID = o.UserID
		updatedOrders = append(updatedOrders, order)
		resp.Body.Close()
	}
	return updatedOrders, nil
}

func (a *Agent) UploadOrders(orders []storage.Orders) error {
	o := make(map[string]string)
	for _, order := range orders {
		o["order"] = order.Order
		orderJSON, err := json.Marshal(o)
		if err != nil {
			a.logger.LogErr(err, "")
		}
		req, err := http.NewRequest(http.MethodPost, a.cfg.Accrual+"/api/orders", bytes.NewBuffer(orderJSON))

		if err != nil {
			a.logger.LogErr(err, "Failed to request")
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			a.logger.LogErr(err, "Failed to do request")
			break
		}
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			a.logger.LogErr(err, "Failed to read body")
			break
		}

		resp.Body.Close()
	}
	return nil
}
