package repositories

import "github.com/CyrilSbrodov/GopherAPIStore/internal/storage"

type StoreGopher struct {
	Store map[string]storage.User
}

func NewStoreGopher() *StoreGopher {
	store := make(map[string]storage.User)
	return &StoreGopher{Store: store}
}

func (s *StoreGopher) GetOrder(u *storage.User) (storage.User, error) {
	//TODO implement me
	panic("implement me")
}
