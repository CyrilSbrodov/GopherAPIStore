package handlers

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/CyrilSbrodov/GopherAPIStore/cmd/loggers"
	"github.com/CyrilSbrodov/GopherAPIStore/internal/storage"
)

type Handlers interface {
	Register(router *chi.Mux)
}

type Handler struct {
	storage.Storage
	logger loggers.Logger
}

func NewHandler(storage storage.Storage, logger *loggers.Logger) Handlers {
	return &Handler{
		storage,
		*logger,
	}
}

func (h *Handler) Register(r *chi.Mux) {
	compressor := middleware.NewCompressor(gzip.DefaultCompression)
	r.Use(compressor.Handler)
	r.Post("/api/user/register", h.Registration())
	r.Post("/api/user/login", h.Login())
	r.Post("/api/user/orders", h.Orders())
	r.Get("/api/user/orders", h.GetOrders())
	r.Get("/api/user/balance", h.Balance())
	r.Post("/api/user/balance/withdraw", h.Withdraw())
	r.Get("/api/user/withdrawals", h.WithdrawInfo())
	r.Get("/api/orders/*", h.Accrual())
}

func (h *Handler) Registration() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var u storage.AcceptUser

		content, err := ioutil.ReadAll(r.Body)
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

		// Create a new random session token
		sessionToken := uuid.NewString()
		expiresAt := time.Now().Add(120 * time.Second)

		// Set the token in the session map, along with the session information
		sessions.Mutex.Lock()
		defer sessions.Mutex.Unlock()
		sessions.sessions[sessionToken] = session{
			login:  u.Login,
			expiry: expiresAt,
		}

		// Finally, we set the client cookie for "session_token" as the session token we just generated
		// we also set an expiry time of 120 seconds
		http.SetCookie(rw, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: expiresAt,
		})

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) Login() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		var u storage.AcceptUser
		content, err := ioutil.ReadAll(r.Body)
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

		// Create a new random session token
		sessionToken := uuid.NewString()
		expiresAt := time.Now().Add(120 * time.Second)

		// Set the token in the session map, along with the session information
		sessions.Mutex.Lock()
		defer sessions.Mutex.Unlock()
		sessions.sessions[sessionToken] = session{
			login:  u.Login,
			expiry: expiresAt,
		}

		// Finally, we set the client cookie for "session_token" as the session token we just generated
		// we also set an expiry time of 120 seconds
		http.SetCookie(rw, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: expiresAt,
		})

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) Orders() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// If the cookie is not set, return an unauthorized status
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			// For any other type of error, return a bad request status
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		sessionToken := c.Value

		// We then get the session from our session map
		sessions.Mutex.Lock()
		defer sessions.Mutex.Unlock()
		userSession, exists := sessions.sessions[sessionToken]
		if !exists {
			// If the session token is not present in session map, return an unauthorized error
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		// If the session is present, but has expired, we can delete the session, and return
		// an unauthorized status
		if userSession.isExpired() {
			delete(sessions.sessions, sessionToken)
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		order := 0

		content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			h.logger.LogErr(err, "")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(content, &order); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		statusCode, err := h.CollectOrder(userSession.login, order)
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
		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// If the cookie is not set, return an unauthorized status
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			// For any other type of error, return a bad request status
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		sessionToken := c.Value

		// We then get the session from our session map
		sessions.Mutex.Lock()
		defer sessions.Mutex.Unlock()
		userSession, exists := sessions.sessions[sessionToken]
		if !exists {
			// If the session token is not present in session map, return an unauthorized error
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		// If the session is present, but has expired, we can delete the session, and return
		// an unauthorized status
		if userSession.isExpired() {
			delete(sessions.sessions, sessionToken)
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		statusCode, orders, err := h.GetOrder(userSession.login)
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
		c, err := r.Cookie("session_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				// If the cookie is not set, return an unauthorized status
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			// For any other type of error, return a bad request status
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		sessionToken := c.Value

		// We then get the session from our session map
		sessions.Mutex.Lock()
		defer sessions.Mutex.Unlock()
		userSession, exists := sessions.sessions[sessionToken]
		if !exists {
			// If the session token is not present in session map, return an unauthorized error
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		// If the session is present, but has expired, we can delete the session, and return
		// an unauthorized status
		if userSession.isExpired() {
			delete(sessions.sessions, sessionToken)
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		balance, err := h.GetBalance(userSession.login)
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
	}
}

func (h *Handler) WithdrawInfo() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
	}
}

func (h *Handler) Accrual() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		url := strings.Split(r.URL.Path, "/")
		fmt.Println(url[3])
		//req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/orders/"+url[3], nil)
		req, err := http.Get("http://localhost:8080/orders/2377225624")

		if err != nil {
			h.logger.LogErr(err, "Failed to request")
			return
		}
		fmt.Println(req)
	}
}
