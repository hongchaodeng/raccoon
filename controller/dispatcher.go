package controller

import (
	"sync"
)

type dispatcher struct {
	listeners map[string][]eventListener
	sync.RWMutex
}

func newDispatcher() *dispatcher {
	d := &dispatcher{
		listeners: make(map[string][]eventListener),
	}

	return d
}

func (d *dispatcher) addListener(typ string, listener eventListener) {
	d.Lock()
	defer d.Unlock()

	listeners := d.listeners[typ]
	d.listeners[typ] = append(listeners, listener)
}

func (d *dispatcher) dispatch(e event) {
	d.RLock()
	defer d.RUnlock()

	listeners := d.listeners[e.Type()]
	for _, l := range listeners {
		l(e)
	}
}