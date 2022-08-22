package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
}

// 1. Race on loop counter
// prints 55555, not 01234.
func RaceOnLoopCounter() {
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			fmt.Println(i) // Not the 'i' you are looking for.
			wg.Done()
		}()
		// go func(j int) {
		// 	fmt.Println(j) // Good. Read local copy of the loop counter.
		// 	wg.Done()
		// }(i)
	}
	wg.Wait()
}

// 2. Accidentally shared variable
// note the use of :=
func ParallelWrite(data []byte) chan error {
	res := make(chan error, 2)

	f1, err := os.Create("file1")
	if err != nil {
		res <- err
	} else {
		go func() {
			// This err is shared with the main goroutine,
			// so the write races with the write below.
			_, err = f1.Write(data)
			// _, err := f1.Write(data)
			res <- err
			f1.Close()
		}()
	}

	f2, err := os.Create("file2") // The second conflicting write to err.
	if err != nil {
		res <- err
	} else {
		go func() {
			_, err = f2.Write(data)
			// _, err = f2.Write(data)
			res <- err
			f2.Close()
		}()
	}

	return res
}

// 3. Unprotected global variable
// following code is called from several goroutines, it leads to races on the service map
var (
	service map[string]net.Addr
	// serviceMu sync.Mutex
)

func RegisterService(name string, addr net.Addr) {
	// serviceMu.Lock()
	// defer serviceMu.Unlock()
	service[name] = addr
}

func LookupService(name string) net.Addr {
	// serviceMu.Lock()
	// defer serviceMu.Unlock()
	return service[name]
}

// 4. Primitive unprotected variable
// A typical fix for this race is to use a channel or a mutex. To preserve the lock-free behavior, one can also use the sync/atomic package.
type Watchdog struct {
	last int64
}

func (w *Watchdog) KeepAlive() {
	w.last = time.Now().UnixNano() // First conflicting access.
	// atomic.StoreInt64(&w.last, time.Now().UnixNano())
}

func (w *Watchdog) Start() {
	go func() {
		for {
			time.Sleep(time.Second)
			// Second conflicting access.
			if w.last < time.Now().Add(-10*time.Second).UnixNano() {
				fmt.Println("No keepalives for 10 seconds. Dying.")
				os.Exit(1)
			}

			// if atomic.LoadInt64(&w.last) < time.Now().Add(-10*time.Second).UnixNano() {
			// 	fmt.Println("No keepalives for 10 seconds. Dying.")
			// 	os.Exit(1)
			// }
		}
	}()
}

// 5.Unsynchronized send and close operations
// To synchronize send and close operations, use a receive operation that guarantees the send is done before the close.j
func asyncSendAndCloseOp() {
	c := make(chan struct{})

	go func() {
		c <- struct{}{}
	}()

	// <-c
	close(c)
}
