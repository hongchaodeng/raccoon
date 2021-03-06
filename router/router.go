package router

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strings"
	"sync"

	"github.com/go-distributed/raccoon/instance"
)

type Router interface {
	Id() string
	AddService(sName, localAddr string, policy Policy) error
	RemoveService(sName string) error
	SetServicePolicy(sName string, policy Policy) error
	// TODO: remove first argument 'sName'.
	// instance contains service name already.
	AddServiceInstance(sName string, instance *instance.Instance) error
	RemoveServiceInstance(sName string, instance *instance.Instance) error
	GetServiceInstances(name string) (*[]*instance.Instance, error)
}

type router struct {
	id          string
	services    map[string]*service
	listener    net.Listener
	addr        *net.TCPAddr
	server      *rpc.Server
	failureChan chan *instance.Instance

	controllerAddr *net.TCPAddr
	sync.Mutex
}

type ReportFailureArgs struct {
	Reporter string
	Instance *instance.Instance
}

type RegRouterArgs struct {
	Id   string
	Addr string
}

type RegRouterReply struct {
}

func New(id string, addrStr string, controllerAddr string) (*router, error) {
	r := &router{
		id:          id,
		services:    make(map[string]*service),
		failureChan: make(chan *instance.Instance, 256),
	}

	var err error
	r.addr, err = net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return nil, err
	}

	if controllerAddr != "" {
		r.controllerAddr, err = net.ResolveTCPAddr("tcp", controllerAddr)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *router) Id() string {
	return r.id
}

func (r *router) Start() (err error) {
	s := rpc.NewServer()

	if err = s.Register(newRouterRPC(r)); err != nil {
		return
	}

	r.listener, err = net.ListenTCP("tcp", r.addr)
	if err != nil {
		return
	}

	go func() {
		closeErr := "use of closed network connection"
		for {
			if conn, err := r.listener.Accept(); err != nil {
				if strings.Contains(err.Error(), closeErr) {
					return
				}
				log.Fatal(err)
			} else {
				go s.ServeConn(conn)
			}
		}
	}()

	go r.monitorFaliure()

	gob.Register(new(SimplePolicy))

	return
}

func (r *router) Stop() (err error) {
	for _, service := range r.services {
		err = service.stop()
		// TODO: safe roll back?
		if err != nil {
			return err
		}
	}

	err = r.listener.Close()
	return err

}

func (r *router) AddService(sName, localAddr string, policy Policy) error {
	r.Lock()
	defer r.Unlock()

	_, ok := r.services[sName]
	if ok {
		return fmt.Errorf("service '%s' already exists", sName)
	}

	s, err := newService(sName, localAddr, policy, r.failureChan)
	if err != nil {
		return err
	}

	// TODO: handle error
	go s.start()

	r.services[sName] = s

	log.Println("Router: successfully added service:", sName)

	return nil
}

func (r *router) RemoveService(sName string) error {
	r.Lock()
	defer r.Unlock()

	s, ok := r.services[sName]
	if !ok {
		return fmt.Errorf("service '%s' does not exist", sName)
	}

	err := s.stop()
	if err != nil {
		return err
	}

	delete(r.services, sName)

	return nil
}

func (r *router) SetServicePolicy(sName string, policy Policy) error {
	r.Lock()
	s, ok := r.services[sName]
	r.Unlock()

	if !ok {
		return fmt.Errorf("service '%s' does not exist", sName)
	}

	err := s.setPolicy(policy)
	if err != nil {
		return err
	}

	return nil
}

func (r *router) AddServiceInstance(sName string, instance *instance.Instance) error {
	r.Lock()
	s, ok := r.services[sName]
	r.Unlock()

	if !ok {
		return fmt.Errorf("service '%s' does not exist", sName)
	}

	err := s.addInstance(instance)
	if err != nil {
		return err
	}

	log.Println("Router: successfully added instance:", instance)

	return nil
}

func (r *router) RemoveServiceInstance(sName string, instance *instance.Instance) error {
	r.Lock()
	s, ok := r.services[sName]
	r.Unlock()

	if !ok {
		return fmt.Errorf("service '%s' does not exist", sName)
	}

	err := s.removeInstance(instance)
	if err != nil {
		return err
	}

	log.Println("Router: successfully removed instance:", instance)
	return nil
}

func (r *router) GetServiceInstances(name string) (*[]*instance.Instance, error) {
	r.Lock()
	defer r.Unlock()

	s, err := r.service(name)
	if err != nil {
		return nil, err
	}
	is := s.Instances()

	return &is, nil
}

func (r *router) service(name string) (*service, error) {
	s, ok := r.services[name]
	if !ok {
		return nil, fmt.Errorf("service %s does not exist", name)
	}
	return s, nil
}

func (r *router) monitorFaliure() {
	for i := range r.failureChan {
		log.Println(r.id, "report failure:", i)
		r.ReportFailure(i)
	}
}

func (r *router) ReportFailure(i *instance.Instance) error {
	c, err := rpc.Dial("tcp", r.controllerAddr.String())
	if err != nil {
		log.Println(err)
		return err
	}
	defer c.Close()

	args := &ReportFailureArgs{r.id, i}
	c.Call("ControllerRPC.ReportFailure", args, nil)
	return nil
}

func (r *router) RegisterOnCtler() error {
	c, err := rpc.Dial("tcp", r.controllerAddr.String())
	if err != nil {
		log.Println(err)
		return err
	}
	defer c.Close()

	args := &RegRouterArgs{
		Id:   r.id,
		Addr: r.addr.String(),
	}
	err = c.Call("ControllerRPC.RegisterRouter", args, nil)

	return err
}
