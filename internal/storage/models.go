package storage

import "time"

type User struct {
	ID             int      `json:"ID"`
	Login          string   `json:"login"`
	HashedPassword string   `json:"password"`
	Orders         []Orders `json:"orders"`
	Accrual        Balance  `json:"accrual"`
}

type Orders struct {
	UserID     int       `json:"user_id,omitempty"`
	Order      string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual"`
	Sum        float64   `json:"sum"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type OrdersUpload struct {
	Order int `json:"order"`
}

type AcceptUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Balance struct {
	Current   float64
	Withdrawn float64
}
