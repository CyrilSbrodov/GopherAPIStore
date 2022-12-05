package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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
	for range ticker.C {
		orders, err := a.Storage.GetAllOrders()
		if err != nil {
			a.logger.LogErr(err, "")
		}
		updatedOrders, err := a.GetAccrual(orders)
		if err != nil {
			a.logger.LogErr(err, "")
		}
		if err := a.UpdateOrders(updatedOrders); err != nil {
			a.logger.LogErr(err, "")
		}
	}
}

func (a *Agent) GetAccrual(orders []storage.Orders) ([]storage.Orders, error) {
	var updatedOrders []storage.Orders
	for _, o := range orders {
		var order storage.Orders
		orderPath := strconv.Itoa(o.Order)
		req, err := http.NewRequest(http.MethodGet, "http://"+a.cfg.Accrual+"/api/orders/"+orderPath, nil)

		if err != nil {
			a.logger.LogErr(err, "Failed to request")
			fmt.Println(err)
			break
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			a.logger.LogErr(err, "Failed to do request")
			break
		}
		res, err := ioutil.ReadAll(resp.Body)
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
		updatedOrders = append(updatedOrders, order)
		resp.Body.Close()
	}
	return updatedOrders, nil
}

func (a *Agent) UpdateOrders(orders []storage.Orders) error {
	return nil
}
