package singleflight

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Do(t *testing.T) {
	var g Group
	v, err := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})
	if got, want := fmt.Sprintf("%v (%T)", v, v), "bar (string)"; got != want {
		t.Errorf("got=%#v,want=%#v", got, want)
	}
	if err != nil {
		t.Errorf("err=%v", err)
	}
}
func Test_DoErr(t *testing.T) {
	var g Group
	err := errors.New("No reaction")
	v, err := g.Do("key", func() (interface{}, error) {
		return nil, err
	})
	if v != nil {
		t.Errorf("v=%v,want=nil", v)
	}
	if err == nil {
		t.Errorf("err=%v,want=%v", err, err)
	}
}

// 测试Do的去重机制
func Test_DoDupSuppress(t *testing.T) {
	var g Group
	var c int32
	call := make(chan string)
	fn := func() (interface{}, error) {
		atomic.AddInt32(&c, 1)
		return <-call, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := g.Do("key", fn)
			if err != nil {
				t.Errorf("err=%v", err)
			}
			if v.(string) != "wx" {
				t.Errorf("got=%#v,want=%#v", v, "wx")
			}
		}()
	}
	//等待达到阻塞效果
	time.Sleep(time.Millisecond)
	call <- "wx"
	wg.Wait()
	if got := atomic.LoadInt32(&c); got != 1 {
		t.Errorf("got=%d,want=1", got)
	}
}
