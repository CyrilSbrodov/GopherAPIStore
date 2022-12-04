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
	UserId     int       `json:"user_id,omitempty"`
	Order      int       `json:"orders"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type AcceptUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Balance struct {
	current   float64
	withdrawn float64
}
