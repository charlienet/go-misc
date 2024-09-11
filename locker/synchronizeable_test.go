package locker_test

import (
	"testing"

	"github.com/charlienet/misc/locker"
)

func TestLocker(t *testing.T) {
	l := struct {
		locker.Locker
	}{}

	l.Synchronize()
	l.Lock()
	println("lock")
	println("lock")

	defer l.Unlock()

	l2 := struct {
		locker.RWLocker
	}{}

	l2.Lock()
	defer l2.Unlock()
}
