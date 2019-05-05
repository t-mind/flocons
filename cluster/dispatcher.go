package cluster

import "github.com/macq/maglev"

type Dispatcher interface {
	Get(name string) (string, error)
	Set(backends []string) error
	Clear()
}

func NewMaglevDispatcher() (Dispatcher, error) {
	return maglev.NewMaglev(nil, 257)
}
