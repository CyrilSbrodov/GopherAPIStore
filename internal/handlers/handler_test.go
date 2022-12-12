package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/mocks"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

func TestHandler_Registration(t *testing.T) {
	type fields struct {
		Storage mocks.MockStorage
		logger  loggers.Logger
	}
	type want struct {
		statusCode int
	}

	tests := []struct {
		name    string
		body    storage.AcceptUser
		request string
		answer  interface{}
		fields  fields
		want    want
	}{
		{
			name:    "Test ok",
			request: "http://localhost:8080/api/user/register",
			body: storage.AcceptUser{
				Login:    "nana",
				Password: "123456",
			},
			answer: nil,
			want: want{
				200,
			},
		},
		{
			name:    "Test 409",
			request: "http://localhost:8080/api/user/register",
			body: storage.AcceptUser{
				Login:    "nana",
				Password: "123456",
			},
			answer: errors.New("conflict"),
			want: want{
				409,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)

			h := &Handler{
				Storage: s,
				logger:  tt.fields.logger,
			}

			s.EXPECT().Register(gomock.Any()).Return(tt.answer)

			bodyJSON, err := json.Marshal(tt.body)
			assert.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, tt.request, bytes.NewBuffer(bodyJSON))
			w := httptest.NewRecorder()

			h.Registration().ServeHTTP(w, request)
			result := w.Result()
			defer result.Body.Close()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
		})
	}
}

func TestHandler_Auth(t *testing.T) {

	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		expectedCode int
	}{
		{
			name: "authenticated",
			cookieValue: map[interface{}]interface{}{
				"user_id": "opa",
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "not authenticated",
			cookieValue:  nil,
			expectedCode: http.StatusUnauthorized,
		},
	}

	sc := securecookie.New([]byte("secret"), nil)
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mocks.NewMockStorage(ctrl)
	logger := loggers.NewLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/user/balance", nil)
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}
			h.Auth(handler).ServeHTTP(rec, req)
			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_Balance(t *testing.T) {
	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		answerFromDB storage.Balance
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test ok",
			cookieValue: map[interface{}]interface{}{
				"user_id": "nana",
			},
			answerFromDB: storage.Balance{
				Current:   1,
				Withdrawn: 2,
			},
			errFromDB:    nil,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Test 401",
			cookieValue:  nil,
			answerFromDB: storage.Balance{},
			errFromDB:    nil,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "nana",
			},
			answerFromDB: storage.Balance{},
			errFromDB:    errors.New("err"),
			expectedCode: http.StatusInternalServerError,
		},
	}
	sc := securecookie.New([]byte("secret"), nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mocks.NewMockStorage(ctrl)
	logger := loggers.NewLogger()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/user/balance", nil)
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().GetBalance(gomock.Any()).Return(tt.answerFromDB, tt.errFromDB).AnyTimes()
			h.Auth(h.Balance()).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_GetOrders(t *testing.T) {
	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
			}
			if got := h.GetOrders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOrders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
			}
			if got := h.Login(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Login() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Orders(t *testing.T) {
	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
			}
			if got := h.Orders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Withdraw(t *testing.T) {
	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
			}
			if got := h.Withdraw(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Withdraw() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_WithdrawInfo(t *testing.T) {
	type fields struct {
		Storage storage.Storage
		logger  loggers.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage: tt.fields.Storage,
				logger:  tt.fields.logger,
			}
			if got := h.WithdrawInfo(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithdrawInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
