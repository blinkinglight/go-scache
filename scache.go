package scache

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
)

func New(buffsize int) *T {
	t := new(T)
	t.dd = []byte{}
	t.mu = &sync.RWMutex{}
	t.blocker = &sync.Mutex{}
	t.cond = sync.NewCond(t.blocker)
	t.connections = make(map[*io.Writer]*bool)
	t.buffsize = buffsize
	return t
}

type T struct {
	dd  []byte
	ddt uint64
	cnt int32

	eof bool
	mu  *sync.RWMutex

	blocker *sync.Mutex
	cond    *sync.Cond

	ppos int

	connections map[*io.Writer]*bool

	buffsize int
}

func (t *T) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.blocker.Lock()
	t.eof = true
	t.blocker.Unlock()
}

func (t *T) ReplayAndSubscribeTo(w io.Writer) {
	var wait bool
	t.blocker.Lock()
	t.connections[&w] = &wait
	t.blocker.Unlock()

	ch, cfn := t.reader1(&w)

	defer func() {
		cfn()
	}()
	for b := range ch {
		_, e := w.Write(b)
		if e != nil {
			return
		}
	}
}

func (t *T) reader1(w *io.Writer) (chan []byte, context.CancelFunc) {
	ch := make(chan []byte)
	var num int32 = 0
	fln := func() int {
		t.mu.RLock()
		defer t.mu.RUnlock()
		return len(t.dd)
	}
	for fln() == 0 {
		t.blocker.Lock()
		t.cond.Wait()
		t.blocker.Unlock()
	}

	rmc := func() {
		t.blocker.Lock()
		delete(t.connections, w)
		t.blocker.Unlock()
	}

	swait := func(v bool) {
		*t.connections[w] = v
	}

	mwait := func() bool {
		return *t.connections[w]
	}

	bmwait := func() bool {
		t.blocker.Lock()
		defer t.blocker.Unlock()
		return mwait()
	}

	brk := false
	ppos := 0

	ctx, cfn := context.WithCancel(context.TODO())

	fn := func() []byte {

		t.mu.RLock()
		defer t.mu.RUnlock()

		var buff = make([]byte, t.buffsize)
		rdr := bytes.NewReader(t.dd)

		n, err := rdr.ReadAt(buff, int64(ppos))

		if err != nil {
			if err != io.EOF {
				brk = true
				return []byte{}
			} else if err == io.EOF {
				swait(true)
				if t.eof {
					brk = true
				}
			}
		}

		ppos += n
		return buff[0:n]
	}
	go func() {
		defer func() {
			close(ch)
			rmc()
		}()
		for {
			for !bmwait() && !brk {
				select {
				case ch <- fn():
				case <-ctx.Done():
					return
				}
				num++
			}
			t.blocker.Lock()
			if t.eof || brk {
				t.blocker.Unlock()
				return
			}
			if !t.eof && mwait() {
				t.cond.Wait()
			}
			t.blocker.Unlock()
		}

	}()
	return ch, cfn
}

func (t *T) Write(chunk []byte) (n int, err error) {

	atomic.AddInt32(&t.cnt, 1)

	t.mu.Lock()
	t.dd = append(t.dd, chunk...)
	t.ddt += uint64(len(chunk))
	t.mu.Unlock()

	t.blocker.Lock()
	for _, v := range t.connections {
		*v = false
	}
	t.cond.Broadcast()
	t.blocker.Unlock()

	return len(chunk), nil
}

// im not sure about this but ..
func (t *T) Close() {
	t.cond = nil
	t.blocker = nil
	t.mu = nil
}
