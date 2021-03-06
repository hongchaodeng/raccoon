package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-distributed/raccoon/instance"
	"github.com/go-distributed/raccoon/router"
	"github.com/stretchr/testify/assert"
)

var _ = fmt.Printf

func TestRegisterRouter(t *testing.T) {
	rId := "test router"
	r, err := router.New(rId, ":14817", "")
	if err != nil {
		t.Fatal(err)
	}

	err = r.Start()
	if err != nil {
		t.Fatal("router start:", err)
	}
	defer func() {
		r.Stop()
		time.Sleep(time.Millisecond * 50)
	}()

	cr, err := NewCRouter(rId, ":14817")
	if err != nil {
		t.Fatal(err)
	}

	cAddr := "127.0.0.1:14818"
	c, err := New(cAddr)
	if err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, c.Routers)

	err = c.RegisterRouter(cr)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(c.Routers), 1)
	assert.Equal(t, c.Routers["test router"], cr)

	err = c.RegisterRouter(cr)
	assert.NotNil(t, err)
}

func TestRegisterServiceInstance(t *testing.T) {
	rId := "test router"
	r, err := router.New(rId, ":14817", "")
	if err != nil {
		t.Fatal(err)
	}

	err = r.Start()
	if err != nil {
		t.Fatal("router start:", err)
	}
	defer func() {
		r.Stop()
		time.Sleep(time.Millisecond * 50)
	}()

	ins, err := instance.NewInstance("test instance", "test service", ":8888")
	if err != nil {
		t.Fatal(err)
	}

	cAddr := "127.0.0.1:14818"
	c, err := New(cAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = c.RegisterServiceInstance(ins)
	if err != nil {
		t.Fatal(err)
	}

	err = c.RegisterServiceInstance(ins)
	assert.NotNil(t, err)
}
