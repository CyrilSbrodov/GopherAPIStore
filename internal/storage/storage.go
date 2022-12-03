package storage

type Storage interface {
	Register(u *AcceptUser) error
	Login(u *AcceptUser) error
	GetOrder(login string) ([]Orders, error)
	CollectOrder(login string, order int) (int, error)
}
