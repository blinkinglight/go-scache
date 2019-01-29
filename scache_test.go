package scache

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

type _tmp struct {
	buf []byte
}

func (t *_tmp) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	return 0, nil
}

type _tmpE struct {
	buf []byte
}

func (t *_tmpE) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("")
}

func TestMain(t *testing.T) {
	h := New(1)

	var s1 *_tmp = new(_tmp)
	var s2 *_tmp = new(_tmp)
	var s3 *_tmp = new(_tmp)
	var s4 *_tmp = new(_tmp)

	var s10 *_tmpE = new(_tmpE)

	go func() {
		h.ReplayAndSubscribeTo(s1)
	}()

	time.Sleep(100 * time.Microsecond)

	h.Write([]byte("a"))

	time.Sleep(100 * time.Microsecond)

	if bytes.Compare(s1.buf, []byte("a")) != 0 {
		t.Fatalf("%s", s1.buf)
	}

	go func() {
		h.ReplayAndSubscribeTo(s2)
	}()

	time.Sleep(100 * time.Microsecond)

	h.Write([]byte("b"))

	time.Sleep(100 * time.Microsecond)
	if bytes.Compare(s2.buf, []byte("ab")) != 0 {
		t.Fatalf("%s", s2.buf)
	}

	if bytes.Compare(s1.buf, []byte("ab")) != 0 {
		t.Fatalf("%s", s1.buf)
	}

	go func() {
		h.ReplayAndSubscribeTo(s3)
	}()

	time.Sleep(100 * time.Microsecond)

	h.Write([]byte("c"))

	time.Sleep(100 * time.Microsecond)

	if bytes.Compare(s1.buf, []byte("abc")) != 0 {
		t.Fatal()
	}
	if bytes.Compare(s2.buf, []byte("abc")) != 0 {
		t.Fatal()
	}
	if bytes.Compare(s3.buf, []byte("abc")) != 0 {
		t.Fatal()
	}
	h.Write([]byte("def"))
	time.Sleep(100 * time.Microsecond)

	h.Done()

	time.Sleep(100 * time.Microsecond)

	go func() {
		h.ReplayAndSubscribeTo(s10)
	}()

	time.Sleep(100 * time.Millisecond)

	h.ReplayAndSubscribeTo(s4)

	time.Sleep(100 * time.Millisecond)

	if bytes.Compare(s4.buf, []byte("abcdef")) != 0 {
		t.Fatal(s4.buf)
	}

	h.Close()

	time.Sleep(100 * time.Millisecond)
}
