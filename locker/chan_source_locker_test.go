package locker_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charlienet/misc/locker"
)

func TestLock(t *testing.T) {
	l := locker.NewChanSourceLocker()
	t.Log(l.Lock("abc"))
	t.Log(l.Lock("abc"))
}

func TestSourceLock(t *testing.T) {
	lock := locker.NewChanSourceLocker()

	var maxKey = 10
	var maxNum = 1000
	var totalOwnNum int32
	var totalNotOwnNum int32
	for a := 0; a < maxKey; a++ {
		var ownNum int32
		var notOwnNum int32
		key := fmt.Sprintf("%s_%d", "aaa", a)
		go func() {
			for i := 0; i < maxNum; i++ {
				go func() {
					o, c := lock.Lock(key)
					if o {
						atomic.AddInt32(&ownNum, 1)
						atomic.AddInt32(&totalOwnNum, 1)
						time.Sleep(time.Second * 1)
						lock.Unlock(key)
						return
					}

					<-c
					atomic.AddInt32(&notOwnNum, 1)
					atomic.AddInt32(&totalNotOwnNum, 1)

				}()
			}
			time.Sleep(time.Second * 2)
			if ownNum != 1 {
				t.Error("ownNum != 1")
			}
			if notOwnNum != int32(maxNum)-ownNum {
				t.Error("notOwnNum err")
			}
		}()
	}
	time.Sleep(time.Second * 3)
	if totalOwnNum != 10 {
		t.Error("totalOwnNum != 10")
	}
	if totalNotOwnNum != int32(maxNum*maxKey)-totalOwnNum {
		t.Error("totalNotOwnNum err")
	}
}
