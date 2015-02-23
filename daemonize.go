// Copyright 2015 The Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package daemonize provides utilities for managing background daemons.
package daemonize

import (
	"sync"
)

// A DaemonPool manages a pool of background goroutines. Each function
// daemonized using Start or StartName will be called repeatedly. Stop
// and StopAll will block until the relevant functions have returned,
// so daemonized functions should not run indefinitely, but should return
// every so often to allow the DaemonPool to stop daemons when requested.
// This can be accomplished with time.After, net.Conn.SetDeadline, etc.
//
// All methods are thread-safe, though calling Stop concurrently with
// StopAll is dangerous, as StopAll may stop the daemon named by Stop
// before Stop gets to it, causing Stop to panic.
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

// StarName daemonizes f (runs it in a background goroutine).
func (d *DaemonPool) Start(f func()) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	d.wg.Add(1)
	go d.run(f, nil)
}

// StarName daemonizes f (runs it in a background goroutine),
// associating it with the given name.
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

// Stop stops the named daemon. Stop will panic
// if the named daemon does not exist.
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

// StopAll stops all daemons.
func (d *DaemonPool) StopAll() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.init()
	d.all <- struct{}{}
	d.wg.Wait()
	d.named = make(map[string]chan chan struct{})
}

func (d *DaemonPool) run(f func(), c chan chan struct{}) {
	for {
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
}
