package repositories

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

var (
	CFG config.ServerConfig
)

func TestMain(m *testing.M) {
	cfg := config.ServerConfigInit()
	CFG.DatabaseURI = cfg.DatabaseURI
	os.Exit(m.Run())
}

func TestPGSStore_Register(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")

	err := s.Register(&storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	})
	assert.NoError(t, err)
}

func TestPGSStore_CollectOrder(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")

	login := "test"
	order := "12345678903"
	_, err := s.CollectOrder(login, order)
	assert.Error(t, err)

	err = s.Register(&storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	})
	assert.NoError(t, err)
	_, err = s.CollectOrder(login, order)
	assert.NoError(t, err)
}

func TestPGSStore_Login(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")
	var u = storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	}
	err := s.Login(&u)
	assert.Error(t, err)
	err = s.Register(&u)
	assert.NoError(t, err)
	err = s.Login(&u)
	assert.NoError(t, err)
}

func TestStoreGopher_GetOrder(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")
	var u = storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	}
	order := "12345678903"
	// error no rows
	statusCode, orders, err := s.GetOrder(u.Login)
	assert.Error(t, err)
	assert.NotNil(t, statusCode)
	assert.Nil(t, orders)

	// add user and orders
	err = s.Register(&u)
	assert.NoError(t, err)
	statusCode, err = s.CollectOrder(u.Login, order)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)

	//check orders
	statusCode, orders, err = s.GetOrder(u.Login)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)
	assert.NotNil(t, orders)
}

func TestPGSStore_GetAllOrders(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")
	var u = storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	}
	orderFirst := "12345678903"
	orderSecond := "12345678911"

	orders, err := s.GetAllOrders()
	assert.Nil(t, orders)
	assert.NoError(t, err)

	// add user and orders
	err = s.Register(&u)
	assert.NoError(t, err)
	statusCode, err := s.CollectOrder(u.Login, orderFirst)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)
	statusCode, err = s.CollectOrder(u.Login, orderSecond)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)

	orders, err = s.GetAllOrders()
	assert.NotNil(t, orders)
	assert.NoError(t, err)

}

func TestPGSStore_GetBalance(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")
	var u = storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	}
	order := "12345678903"

	balance, err := s.GetBalance(u.Login)
	assert.Error(t, err)
	assert.Nil(t, balance)

	err = s.Register(&u)
	assert.NoError(t, err)

	statusCode, err := s.CollectOrder(u.Login, order)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)

	var newOrders = []storage.Orders{
		{
			UserID:  0,
			Order:   "12345678903",
			Status:  "PROCESSED",
			Accrual: 500,
			Sum:     0,
		},
	}

	err = s.UpdateOrders(newOrders)
	assert.NoError(t, err)

	balance, err = s.GetBalance(u.Login)
	assert.NoError(t, err)
	assert.NotNil(t, balance)

}

func TestPGSStore_UpdateOrders(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")

	var u = storage.AcceptUser{
		Login:    "test",
		Password: "123456",
	}
	order := "12345678903"

	err := s.Register(&u)
	assert.NoError(t, err)

	statusCode, err := s.CollectOrder(u.Login, order)
	assert.NoError(t, err)
	assert.NotNil(t, statusCode)

	var newOrders = []storage.Orders{
		{
			UserID:  0,
			Order:   "12345678903",
			Status:  "PROCESSED",
			Accrual: 500,
			Sum:     0,
		},
	}

	err = s.UpdateOrders(newOrders)
	assert.NoError(t, err)
}

func TestPGSStore_Valid(t *testing.T) {
	s, teardown := TestPGStore(t, CFG)
	defer teardown("users", "orders")
	tests := []struct {
		name           string
		order          string
		expectedStatus bool
	}{
		{
			name:           "is valid",
			order:          "12345678903",
			expectedStatus: true,
		},
		{
			name:  "not valid",
			order: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedStatus, s.Valid(tt.order))
		})
	}
}
