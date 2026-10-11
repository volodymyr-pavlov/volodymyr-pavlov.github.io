package gen

import (
	"runtime"
	"sync"
)

// inParallel runs task for every index below n, at most one per CPU at a time, and returns the first error by index.
func inParallel(n int, task func(i int) error) error {
	errs := make([]error, n)
	slots := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i := range n {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			errs[i] = task(i)
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
