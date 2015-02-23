package daemonize

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRace(t *testing.T) {
	var d DaemonPool
	for i := 0; i < 100; i++ {
		d.Start(func() {
			time.Sleep(time.Millisecond)
		})
	}
	d.StopAll()
	checkEmpty(&d, t)
}

func TestName(t *testing.T) {
	var d DaemonPool
	nums := rand.Perm(1000)
	names := make([]string, 1000)
	for i, n := range nums {
		names[i] = strconv.Itoa(n)
		d.StartName(names[i], func() {
			time.Sleep(time.Millisecond)
		})
	}
	var wg sync.WaitGroup
	nums = rand.Perm(1000)
	for _, n := range nums[:500] {
		wg.Add(1)
		go func(n int) {
			d.Stop(strconv.Itoa(n))
			wg.Done()
		}(n)
	}
	wg.Wait()
	d.StopAll()
	checkEmpty(&d, t)
}

func checkEmpty(d *DaemonPool, t *testing.T) {
	if len(d.named) != 0 {
		t.Errorf("non-empty DaemonPool.named: len = %v", len(d.named))
	}
}
