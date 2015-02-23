package daemonize

import (
	"sync"
)

type DaemonPool struct {
	named map[string]chan chan struct{}
	all   chan struct{}
	wg    sync.WaitGroup

	initialized bool
	mtx         sync.Mutex
}

func (d *DaemonPool) init() {
	if !d.initialized {
		d.initialized = true
		d.named = make(map[string]chan chan struct{})
		d.all = make(chan struct{}, 1)
	}
}

func (d *DaemonPool) Start(f func()) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	d.wg.Add(1)
	go d.run(f, nil)
}

func (d *DaemonPool) StartName(name string, f func()) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	if _, ok := d.named[name]; ok {
		panic("daemonize: named daemon already exists")
	}
	c := make(chan chan struct{})
	d.named[name] = c
	d.wg.Add(1)
	go d.run(f, c)
}

func (d *DaemonPool) Stop(name string) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	c, ok := d.named[name]
	if !ok {
		panic("daemonize: no such named daemon")
	}
	cc := make(chan struct{})
	c <- cc
	<-cc
	delete(d.named, name)
}

func (d *DaemonPool) StopAll() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	d.all <- struct{}{}
	d.wg.Wait()
	d.named = make(map[string]chan chan struct{})
}

func (d *DaemonPool) run(f func(), c chan chan struct{}) {
	select {
	case s := <-d.all:
		d.all <- s
		d.wg.Done()
		return
	case c := <-c:
		d.wg.Done()
		c <- struct{}{}
		return
	}
	f()
}
