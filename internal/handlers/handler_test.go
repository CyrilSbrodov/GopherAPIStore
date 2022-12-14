package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/mocks"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

func TestHandler_Registration(t *testing.T) {

	tests := []struct {
		name         string
		body         storage.AcceptUser
		answer       interface{}
		expectedCode int
	}{
		{
			name: "Test ok",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "123456",
			},
			answer:       nil,
			expectedCode: http.StatusOK,
		},
		{
			name: "Test 409",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "123456",
			},
			answer:       errors.New("conflict"),
			expectedCode: http.StatusConflict,
		},
		{
			name: "Test 400",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "",
			},
			answer:       errors.New("err"),
			expectedCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			bodyJSON, err := json.Marshal(tt.body)
			assert.NoError(t, err)

			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBuffer(bodyJSON))
			s.EXPECT().Register(gomock.Any()).Return(tt.answer).AnyTimes()
			h.Registration().ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
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
				"user_id": "test",
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "not authenticated",
			cookieValue:  nil,
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusOK)
			})
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
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
		answerFromDB *storage.Balance
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test ok",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answerFromDB: &storage.Balance{
				Current:   1,
				Withdrawn: 2,
			},
			errFromDB:    nil,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Test 401",
			cookieValue:  nil,
			answerFromDB: nil,
			errFromDB:    nil,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answerFromDB: nil,
			errFromDB:    errors.New("err"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
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
	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		answerFromDB []storage.Orders
		statusCode   int
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test ok",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answerFromDB: []storage.Orders{
				{
					Order:   "12345678903",
					Status:  "PROCESSED",
					Accrual: 500,
				},
			},
			errFromDB:    nil,
			statusCode:   http.StatusOK,
			expectedCode: http.StatusOK,
		},
		{
			name:        "Test 401",
			cookieValue: nil,
			answerFromDB: []storage.Orders{
				{
					Order: "12345678903",
				},
			},
			errFromDB:    nil,
			statusCode:   http.StatusOK,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 204",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answerFromDB: nil,
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusNoContent,
			expectedCode: http.StatusNoContent,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answerFromDB: []storage.Orders{},
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusInternalServerError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/user/orders", nil)
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().GetOrder(gomock.Any()).Return(tt.statusCode, tt.answerFromDB, tt.errFromDB).AnyTimes()
			h.Auth(h.GetOrders()).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		body         storage.AcceptUser
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test ok",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "123456",
			},
			errFromDB:    nil,
			expectedCode: http.StatusOK,
		},
		{
			name: "Test 400",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "",
			},
			errFromDB:    nil,
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Test 401",
			body: storage.AcceptUser{
				Login:    "test",
				Password: "12345",
			},
			errFromDB:    errors.New("err"),
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			rec := httptest.NewRecorder()
			bodyJSON, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBuffer(bodyJSON))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().Login(gomock.Any()).Return(tt.errFromDB).AnyTimes()
			h.Login().ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_Orders(t *testing.T) {
	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		body         string
		statusCode   int
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test Accepted/202",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "12345678903",
			errFromDB:    nil,
			statusCode:   http.StatusAccepted,
			expectedCode: http.StatusAccepted,
		},
		{
			name: "Test ok/200",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "12345678903",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusOK,
			expectedCode: http.StatusOK,
		},
		{
			name: "Test 400",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "1",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusBadRequest,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Test 401",
			cookieValue:  nil,
			body:         "1",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusUnauthorized,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 409",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "12345678903",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusConflict,
			expectedCode: http.StatusConflict,
		},
		{
			name: "Test 422",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "1Afsaf123",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusUnprocessableEntity,
			expectedCode: http.StatusUnprocessableEntity,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body:         "1Afsaf123",
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusInternalServerError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBuffer([]byte(tt.body)))
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().CollectOrder(gomock.Any(), gomock.Any()).Return(tt.statusCode, tt.errFromDB).AnyTimes()
			h.Auth(h.Orders()).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_Withdraw(t *testing.T) {
	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		body         storage.Order
		statusCode   int
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test 200",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body: storage.Order{
				Order: "2377225624",
				Sum:   500,
			},
			errFromDB:    nil,
			statusCode:   http.StatusOK,
			expectedCode: http.StatusOK,
		},
		{
			name: "Test 422",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body: storage.Order{
				Order: "23",
				Sum:   500,
			},
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusUnprocessableEntity,
			expectedCode: http.StatusUnprocessableEntity,
		},
		{
			name:        "Test 401",
			cookieValue: nil,
			body: storage.Order{
				Order: "2377225624",
				Sum:   500,
			},
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusUnauthorized,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 402",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body: storage.Order{
				Order: "2377225624",
				Sum:   500,
			},
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusPaymentRequired,
			expectedCode: http.StatusPaymentRequired,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			body: storage.Order{
				Order: "2377225624",
				Sum:   500,
			},
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusInternalServerError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			bodyJSON, err := json.Marshal(tt.body)
			assert.NoError(t, err)
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBuffer(bodyJSON))
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().Withdraw(gomock.Any(), gomock.Any()).Return(tt.statusCode, tt.errFromDB).AnyTimes()
			h.Auth(h.Withdraw()).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_WithdrawInfo(t *testing.T) {
	tests := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		answer       []storage.Order
		statusCode   int
		errFromDB    error
		expectedCode int
	}{
		{
			name: "Test 200",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answer: []storage.Order{
				{
					Order:       "12345678903",
					Sum:         500,
					ProcessedAt: time.Now(),
				},
			},
			errFromDB:    nil,
			statusCode:   http.StatusOK,
			expectedCode: http.StatusOK,
		},
		{
			name: "Test 204",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answer:       nil,
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusNoContent,
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Test 401",
			cookieValue:  nil,
			answer:       nil,
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusUnauthorized,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Test 500",
			cookieValue: map[interface{}]interface{}{
				"user_id": "test",
			},
			answer:       nil,
			errFromDB:    errors.New("err"),
			statusCode:   http.StatusInternalServerError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := securecookie.New([]byte("secret"), nil)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			s := mocks.NewMockStorage(ctrl)
			logger := loggers.NewLogger()
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			cookieStr, _ := sc.Encode(sessionName, tt.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			h := &Handler{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessions.NewCookieStore([]byte("secret")),
			}

			s.EXPECT().Withdrawals(gomock.Any()).Return(tt.statusCode, tt.answer, tt.errFromDB).AnyTimes()
			h.Auth(h.WithdrawInfo()).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}

func TestHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mocks.NewMockStorage(ctrl)
	logger := loggers.NewLogger()
	sessionStore := sessions.NewCookieStore([]byte("secret"))
	router := chi.NewRouter()

	type fields struct {
		Storage      storage.Storage
		logger       loggers.Logger
		sessionStore sessions.Store
	}
	type args struct {
		r *chi.Mux
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "register",
			fields: fields{
				Storage:      s,
				logger:       *logger,
				sessionStore: sessionStore,
			},
			args: args{r: router},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Storage:      tt.fields.Storage,
				logger:       tt.fields.logger,
				sessionStore: tt.fields.sessionStore,
			}

			h.Register(tt.args.r)
		})
	}
}

func TestNewHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := mocks.NewMockStorage(ctrl)
	logger := loggers.NewLogger()
	sessionStore := sessions.NewCookieStore([]byte("secret"))

	type args struct {
		storage      storage.Storage
		logger       *loggers.Logger
		sessionStore sessions.Store
	}
	tests := []struct {
		name string
		args args
		want Handlers
	}{
		{
			name: "get new handler",
			args: args{
				storage:      s,
				logger:       logger,
				sessionStore: sessionStore,
			},
			want: NewHandler(s, logger, sessionStore),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, NewHandler(tt.args.storage, tt.args.logger, tt.args.sessionStore), "NewHandler(%v, %v, %v)", tt.args.storage, tt.args.logger, tt.args.sessionStore)
		})
	}
}
