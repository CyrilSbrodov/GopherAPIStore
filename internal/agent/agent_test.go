package agent

//import (
//	"net/http"
//	"reflect"
//	"testing"
//	"time"
//
//	"github.com/CyrilSbrodov/GopherAPIStore/cmd/config"
//	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
//	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
//)
//
//func TestAgent_GetAccrual(t *testing.T) {
//	type fields struct {
//		Storage storage.Storage
//		logger  loggers.Logger
//		client  http.Client
//		cfg     config.ServerConfig
//	}
//	type args struct {
//		orders []storage.Orders
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    []storage.Orders
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			a := &Agent{
//				Storage: tt.fields.Storage,
//				logger:  tt.fields.logger,
//				client:  tt.fields.client,
//				cfg:     tt.fields.cfg,
//			}
//			got, err := a.GetAccrual(tt.args.orders)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("GetAccrual() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetAccrual() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestAgent_Start(t *testing.T) {
//	type fields struct {
//		Storage storage.Storage
//		logger  loggers.Logger
//		client  http.Client
//		cfg     config.ServerConfig
//	}
//	type args struct {
//		ticker time.Ticker
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			a := &Agent{
//				Storage: tt.fields.Storage,
//				logger:  tt.fields.logger,
//				client:  tt.fields.client,
//				cfg:     tt.fields.cfg,
//			}
//			a.Start(tt.args.ticker)
//		})
//	}
//}
//
//func TestAgent_UpdateOrders(t *testing.T) {
//	type fields struct {
//		Storage storage.Storage
//		logger  loggers.Logger
//		client  http.Client
//		cfg     config.ServerConfig
//	}
//	type args struct {
//		orders []storage.Orders
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			a := &Agent{
//				Storage: tt.fields.Storage,
//				logger:  tt.fields.logger,
//				client:  tt.fields.client,
//				cfg:     tt.fields.cfg,
//			}
//			if err := a.UpdateOrders(tt.args.orders); (err != nil) != tt.wantErr {
//				t.Errorf("UpdateOrders() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestAgent_UploadOrders(t *testing.T) {
//	type fields struct {
//		Storage storage.Storage
//		logger  loggers.Logger
//		client  http.Client
//		cfg     config.ServerConfig
//	}
//	type args struct {
//		orders []storage.Orders
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			a := &Agent{
//				Storage: tt.fields.Storage,
//				logger:  tt.fields.logger,
//				client:  tt.fields.client,
//				cfg:     tt.fields.cfg,
//			}
//			if err := a.UploadOrders(tt.args.orders); (err != nil) != tt.wantErr {
//				t.Errorf("UploadOrders() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestNewAgent(t *testing.T) {
//	type args struct {
//		storage storage.Storage
//		logger  loggers.Logger
//		cfg     config.ServerConfig
//	}
//	tests := []struct {
//		name string
//		args args
//		want *Agent
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := NewAgent(tt.args.storage, tt.args.logger, tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewAgent() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
