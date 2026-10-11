package gen

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestInParallelRunsEveryTaskAndReturnsTheFirstErrorByIndex(t *testing.T) {
	var ran atomic.Int32
	first, second := errors.New("first"), errors.New("second")
	err := inParallel(50, func(i int) error {
		ran.Add(1)
		switch i {
		case 40:
			return second
		case 7:
			return first
		}
		return nil
	})
	if err != first {
		t.Errorf("inParallel() = %v, want the error of the lowest index", err)
	}
	if ran.Load() != 50 {
		t.Errorf("inParallel() ran %d of 50 tasks", ran.Load())
	}
	if err := inParallel(0, func(int) error { return first }); err != nil {
		t.Errorf("inParallel(0) = %v", err)
	}
}
