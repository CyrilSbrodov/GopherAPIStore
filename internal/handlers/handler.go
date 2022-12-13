package handlers

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

const (
	sessionName        = "session_token"
	ctxKeyUser  ctxKey = iota
)

type ctxKey int8

type Handlers interface {
	Register(router *chi.Mux)
}

type Handler struct {
	storage.Storage
	logger       loggers.Logger
	sessionStore sessions.Store
}

func NewHandler(storage storage.Storage, logger *loggers.Logger, sessionStore sessions.Store) Handlers {
	return &Handler{
		storage,
		*logger,
		sessionStore,
	}
}

func (h *Handler) Register(r *chi.Mux) {
	compressor := middleware.NewCompressor(gzip.DefaultCompression)
	r.Use(compressor.Handler)
	r.Post("/api/user/register", h.Registration())
	r.Post("/api/user/login", h.Login())

	r.Group(func(r chi.Router) {
		r.Use(h.Auth)
		r.Post("/api/user/orders", h.Orders())
		r.Get("/api/user/orders", h.GetOrders())
		r.Get("/api/user/balance", h.Balance())
		r.Post("/api/user/balance/withdraw", h.Withdraw())
		r.Get("/api/user/withdrawals", h.WithdrawInfo())
	})
}

func (h *Handler) Registration() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var u storage.AcceptUser

		content, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(content, &u); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		if u.Login == "" || u.Password == "" {
			rw.WriteHeader(http.StatusBadRequest)
			err = fmt.Errorf("login or password is empty")
			rw.Write([]byte(err.Error()))
			return
		}

		err = h.Storage.Register(&u)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusConflict)
			err = fmt.Errorf("login %v is already registered", u.Login)
			rw.Write([]byte(err.Error()))
			return
		}

		session, err := h.sessionStore.Get(r, sessionName)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		session.Values["user_id"] = u.Login

		if err = h.sessionStore.Save(r, rw, session); err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) Login() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		var u storage.AcceptUser
		content, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(content, &u); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		if u.Login == "" || u.Password == "" {
			rw.WriteHeader(http.StatusBadRequest)
			err = fmt.Errorf("login or password is empty")
			rw.Write([]byte(err.Error()))
			return
		}

		err = h.Storage.Login(&u)
		if err != nil {
			h.logger.LogErr(err, "wrong password or login")
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte(err.Error()))
			return
		}

		session, err := h.sessionStore.Get(r, sessionName)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		session.Values["user_id"] = u.Login

		if err = h.sessionStore.Save(r, rw, session); err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) Orders() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		content, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer r.Body.Close()

		//userSession := "Mas"
		userSession := r.Context().Value(ctxKeyUser).(string)
		statusCode, err := h.CollectOrder(userSession, string(content))
		switch statusCode {
		case http.StatusOK:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusAccepted:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusAccepted)
			return
		case http.StatusBadRequest:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusConflict:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusConflict)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusInternalServerError:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusUnprocessableEntity:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusUnprocessableEntity)
			rw.Write([]byte(err.Error()))
			return
		}
	}
}

func (h *Handler) GetOrders() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		//userSession := "Mas"
		userSession := r.Context().Value(ctxKeyUser).(string)
		statusCode, orders, err := h.GetOrder(userSession)
		switch statusCode {
		case http.StatusOK:
			ordersJSON, err := json.Marshal(orders)
			if err != nil {
				h.logger.LogErr(err, "failed to marshal")
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write(ordersJSON)
			return
		case http.StatusNoContent:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusInternalServerError:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
	}
}

func (h *Handler) Balance() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		//userSession := "Mas"
		userSession := r.Context().Value(ctxKeyUser).(string)
		balance, err := h.GetBalance(userSession)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		ordersJSON, err := json.Marshal(balance)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write(ordersJSON)
	}
}

func (h *Handler) Withdraw() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		content, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		var o storage.Order
		if err := json.Unmarshal(content, &o); err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		userSession := r.Context().Value(ctxKeyUser).(string)
		statusCode, err := h.Storage.Withdraw(userSession, &o)
		switch statusCode {
		case http.StatusOK:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			return
		case http.StatusPaymentRequired:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusPaymentRequired)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusInternalServerError:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusUnprocessableEntity:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusUnprocessableEntity)
			rw.Write([]byte(err.Error()))
			return
		}
	}
}

func (h *Handler) WithdrawInfo() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userSession := r.Context().Value(ctxKeyUser).(string)
		statusCode, withdrawals, err := h.Withdrawals(userSession)
		switch statusCode {
		case http.StatusOK:
			result, err := json.Marshal(withdrawals)
			if err != nil {
				h.logger.LogErr(err, "failed to marshal")
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(err.Error()))
				return
			}
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write(result)
			return
		case http.StatusNoContent:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			rw.Write([]byte(err.Error()))
			return
		case http.StatusInternalServerError:
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
	}
}

func (h *Handler) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		session, err := h.sessionStore.Get(r, sessionName)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		userSession, ok := session.Values["user_id"]
		if !ok {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(rw, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, userSession)))
	})
}
