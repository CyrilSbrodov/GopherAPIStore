package handlers

import (
	"sync"
	"time"
)

type allSessions struct {
	sessions map[string]session
	sync.Mutex
}

var sessions = allSessions{
	make(map[string]session),
	sync.Mutex{},
}

type session struct {
	login  string
	expiry time.Time
}

func (s session) isExpired() bool {
	return s.expiry.Before(time.Now())
}
