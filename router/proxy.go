package router

import (
	"fmt"
	"net"
	"sync"
)

type proxy struct {
	status     string
	connectors []*connector
	localAddr  *net.TCPAddr
	// todo(xiangli) save service instance rather than
	// just remote address
	remoteAddrs []*net.TCPAddr
	listener    net.Listener
	// maybe we should decouple selector with proxy
	selector selector
	sync.Mutex
}

const (
	initialized = "initialized"
	running     = "running"
	stopped     = "stopped"
)

func newProxy(laddrStr string, raddrStrs []string) (*proxy, error) {
	p := &proxy{
		connectors:  make([]*connector, 0),
		remoteAddrs: make([]*net.TCPAddr, 0),
		status:      initialized,
		selector:    defaultSelector,
	}

	if len(raddrStrs) == 0 {
		return nil, fmt.Errorf("no remote address is given")
	}

	var err error
	p.localAddr, err = net.ResolveTCPAddr("tcp", laddrStr)
	if err != nil {
		return nil, err
	}

	for i := range raddrStrs {
		if err := p.addRemoteAddr(raddrStrs[i]); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *proxy) start() error {
	var err error
	p.Lock()
	if p.status != initialized {
		defer p.Unlock()
		return fmt.Errorf("the status of proxy is not initialized [%s]", p.status)
	}

	p.status = running
	p.listener, err = net.Listen("tcp", p.localAddr.String())
	p.Unlock()

	if err != nil {
		return err
	}

	for {
		one, err := p.listener.Accept()
		if err != nil {
			// handle error
			return err
		}
		go func(one net.Conn) {
			// todo(xiangli) add a selector
			p.Lock()
			raddr := p.selector(p.remoteAddrs)
			p.Unlock()

			other, err := net.Dial("tcp", raddr.String())
			if err != nil {
				return
			}

			c := newConnector(one, other)
			if err := p.addConnector(c); err != nil {
				return
			}

			c.connect()
		}(one)
	}
}

// todo(xiangli) Graceful shutdown
func (p *proxy) stop() error {
	p.Lock()
	defer p.Unlock()

	if p.status != running {
		return fmt.Errorf("the status of proxy is not running [%s]", p.status)
	}

	p.status = stopped
	p.listener.Close()
	for _, c := range p.connectors {
		c.disconnect()
	}
	p.connectors = nil
	return nil
}

func (p *proxy) addConnector(c *connector) error {
	p.Lock()
	defer p.Unlock()

	if p.status != running {
		return fmt.Errorf("the status of proxy is not running [%s]", p.status)
	}

	p.connectors = append(p.connectors, c)
	return nil
}

func (p *proxy) addRemoteAddr(raddrStr string) error {
	p.Lock()
	defer p.Unlock()

	raddr, err := net.ResolveTCPAddr("tcp", raddrStr)
	if err != nil {
		return err
	}

	p.remoteAddrs = append(p.remoteAddrs, raddr)
	return nil
}
