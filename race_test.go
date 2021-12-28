//go:build race
// +build race

package liner

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

func TestWriteHistory(t *testing.T) {
	oldout := os.Stdout
	defer func() { os.Stdout = oldout }()
	oldin := os.Stdout
	defer func() { os.Stdin = oldin }()

	newinr, newinw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = newinr
	newoutr, newoutw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer newoutr.Close()
	os.Stdout = newoutw

	var wait sync.WaitGroup
	wait.Add(1)
	h := &sliceHistory{}
	s := NewLiner(h)
	go func() {
		h.AppendHistory("foo")
		h.AppendHistory("bar")
		s.Prompt("")
		wait.Done()
	}()

	h.WriteHistory(ioutil.Discard)

	newinw.Close()
	wait.Wait()
}
