package storage

type Storage interface {
	Register(u *AcceptUser) error
	Login(u *AcceptUser) error
	GetOrder(login string) (int, []Orders, error)
	CollectOrder(login string, order int) (int, error)
	GetBalance(login string) (Balance, error)
	GetAllOrders() ([]Orders, error)
	UpdateOrders([]Orders) error
}
