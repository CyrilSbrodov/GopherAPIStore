package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

func TestAgent_GetAccrual(t *testing.T) {

	client := http.Client{}
	logger := loggers.NewLogger()
	cfg := config.ServerConfigInit()

	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
		client  http.Client
		cfg     config.ServerConfig
	}
	type args struct {
		orders []storage.Orders
	}
	type expected struct {
		orders     []storage.Orders
		statusCode int
		err        error
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		answer   storage.Orders
		expected expected
	}{
		{
			name: "test 200",
			fields: fields{
				Storage: nil,
				logger:  *logger,
				client:  client,
				cfg:     cfg,
			},
			args: args{orders: []storage.Orders{
				{
					Order:  "12345678903",
					Status: "NEW",
				},
			}},
			expected: expected{
				orders: []storage.Orders{
					{
						Order:   "12345678903",
						Status:  "PROCESSED",
						Accrual: 500,
					},
				},
				statusCode: 200,
				err:        nil,
			},
			answer: storage.Orders{
				Order:   "12345678903",
				Status:  "PROCESSED",
				Accrual: 500,
			},
		},
		{
			name: "test 200",
			fields: fields{
				Storage: nil,
				logger:  *logger,
				client:  client,
				cfg:     cfg,
			},
			args: args{orders: []storage.Orders{
				{
					Order:  "12345678903",
					Status: "NEW",
				},
			}},
			expected: expected{
				orders:     nil,
				statusCode: http.StatusTooManyRequests,
				err:        nil,
			},
			answer: storage.Orders{
				Order:   "12345678903",
				Status:  "PROCESSED",
				Accrual: 500,
			},
		},
		{
			name: "test 500",
			fields: fields{
				Storage: nil,
				logger:  *logger,
				client:  client,
				cfg:     cfg,
			},
			args: args{orders: []storage.Orders{
				{
					Order:  "12345678903",
					Status: "NEW",
				},
			}},
			expected: expected{
				orders:     nil,
				statusCode: http.StatusInternalServerError,
				err:        fmt.Errorf("error from accrual system"),
			},
			answer: storage.Orders{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
				client:  tt.fields.client,
				cfg:     tt.fields.cfg,
			}
			ordersJSON, err := json.Marshal(tt.answer)
			assert.NoError(t, err)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.expected.statusCode)
				w.Write(ordersJSON)
			}))
			defer srv.Close()
			a.cfg.Accrual = srv.URL

			got, err := a.GetAccrual(tt.args.orders)
			assert.Equal(t, tt.expected.err, err)
			assert.Equal(t, tt.expected.orders, got)
		})
	}
}
