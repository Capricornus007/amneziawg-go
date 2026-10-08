package conn

import (
	"errors"
	"sync"
)

var _ Bind = (*Multibind)(nil)

func NewMultibind(udp Bind, tcp Bind) *Multibind {
	return &Multibind{
		udp:  udp,
		tcp:  tcp,
		Bind: udp,
	}
}

type Multibind struct {
	Bind
	udp   Bind
	tcp   Bind
	mutex sync.Mutex
}

func (mb *Multibind) SelectNetwork(proto string) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if proto == "udp" {
		mb.Bind = mb.udp
		return nil
	}

	if proto == "tcp" {
		mb.Bind = mb.tcp
		return nil
	}

	return errors.New("unknown network")
}

func (mb *Multibind) Proto() string {
	switch mb.Bind {
	case mb.tcp:
		return "tcp"
	case mb.udp:
		return "udp"
	}

	return "unknown"
}
