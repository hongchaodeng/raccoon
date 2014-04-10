package router

import (
	rmtService "github.com/go-distributed/raccoon/service"
)

type Status int

const (
	Ok Status = iota + 1
	NotOk
)

type RouterRPC struct {
	router Router
}

type Reply struct {
	Value string
}

type ServiceArgs struct {
	ServiceName string
	LocalAddr   string
	Policy      Policy
}

type ServiceReply struct {
}

type InstanceArgs struct {
	ServiceName string
	Instance    *rmtService.Instance
}

type InstanceReply struct {
}

func newRouterRPC(router Router) *RouterRPC {
	return &RouterRPC{
		router: router,
	}
}

func (rpc *RouterRPC) Echo(arg string, reply *Reply) error {
	reply.Value = arg
	return nil
}

func (rpc *RouterRPC) AddService(args *ServiceArgs, reply *ServiceReply) error {
	return rpc.router.AddService(args.ServiceName, args.LocalAddr, args.Policy)
}

func (rpc *RouterRPC) RemoveService(args *ServiceArgs, reply *ServiceReply) error {
	return rpc.router.RemoveService(args.ServiceName)
}

func (rpc *RouterRPC) SetServicePolicy(args *ServiceArgs, reply *ServiceReply) error {
	return rpc.router.SetServicePolicy(args.ServiceName, args.Policy)
}

func (rpc *RouterRPC) AddServiceInstance(args *InstanceArgs, reply *InstanceReply) error {
	return rpc.router.AddServiceInstance(args.ServiceName, args.Instance)
}

func (rpc *RouterRPC) RemoveServiceInstance(args *InstanceArgs, reply *InstanceReply) error {
	return rpc.router.RemoveServiceInstance(args.ServiceName, args.Instance)
}
