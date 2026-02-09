package store

import (
	"errors"

	"github.com/nfedorov/port_server/internal/model"
)

var (
	ErrPortTaken        = errors.New("port already allocated")
	ErrPortBusy         = errors.New("port in use on system")
	ErrServiceAllocated = errors.New("service already allocated")
	ErrNotFound         = errors.New("allocation not found")
)

type Filter struct {
	App      string
	Instance string
	Service  string
	Port     int
}

type Store interface {
	Allocate(req model.AllocateRequest, portMin, portMax int) (*model.Allocation, error)
	List(f Filter) ([]model.Allocation, error)
	GetByPort(port int) (*model.Allocation, error)
	DeleteByID(id int64) error
	DeleteByFilter(f Filter) (int64, error)
	Close() error
}
