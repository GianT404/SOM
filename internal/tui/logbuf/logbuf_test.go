package logbuf

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func TestRingOrderAndCapacity(t *testing.T) {
	r := New(3)
	for i := 1; i <= 3; i++ {
		r.Push("line-" + strconv.Itoa(i))
	}
	if got := r.Lines(); !reflect.DeepEqual(got, []string{"line-1", "line-2", "line-3"}) {
		t.Fatalf("got %v", got)
	}
	if r.Len() != 3 {
		t.Fatalf("len=%d", r.Len())
	}
}

func TestRingWrapOverwritesOldest(t *testing.T) {
	r := New(3)
	for i := 1; i <= 5; i++ {
		r.Push(strconv.Itoa(i))
	}
	want := []string{"3", "4", "5"}
	if got := r.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if r.Len() != 3 {
		t.Fatalf("len=%d want 3", r.Len())
	}
}

func TestWriteSplitsLines(t *testing.T) {
	r := New(10)
	r.Write([]byte("a\nb\n\nc\n"))
	want := []string{"a", "b", "c"}
	if got := r.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestReplaceSeeds(t *testing.T) {
	r := New(3)
	r.Replace([]string{"x", "y", "z", "w"})
	if got := r.Lines(); !reflect.DeepEqual(got, []string{"y", "z", "w"}) {
		t.Fatalf("got %v", got)
	}
}

func TestConcurrentPush(t *testing.T) {
	r := New(64)
	const workers = 8
	const per = 2000
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < per; i++ {
				r.Push(strconv.Itoa(w) + ":" + strconv.Itoa(i))
			}
		}(w)
	}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			_ = r.Lines()
			_ = r.Len()
		}
		close(done)
	}()
	wg.Wait()
	<-done
	if r.Len() != 64 {
		t.Fatalf("len=%d want 64", r.Len())
	}
}

func TestSnapshotNoDeadlock(t *testing.T) {
	r := New(4)
	r.Push("before")
	r.Push("during")
	r.Lines() // publish snapshot version

	r.mu.Lock()
	got := r.Snapshot() // không được deadlock
	r.mu.Unlock()

	if !reflect.DeepEqual(got, []string{"before", "during"}) {
		t.Fatalf("got %v", got)
	}
}

// TestSnapshotEmptyBeforeAnyPublish: snapshot rơi về bản rỗng nếu chưa từng có
// publish (không crash, chỉ kiểm tra hành vi).
func TestSnapshotEmptyBeforeAnyPublish(t *testing.T) {
	r := New(4)
	r.mu.Lock()
	got := r.Snapshot()
	r.mu.Unlock()
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestDumpCrashTo(t *testing.T) {
	r := New(16)
	r.Push("hello")
	r.Push("world")

	path := filepath.Join(t.TempDir(), "crash_som_tui_test.log")
	if err := r.DumpCrashTo(path, "test note"); err != nil {
		t.Fatalf("dump: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(data)
	for _, want := range []string{"SOM TUI crash dump", "test note", "hello", "world", "goroutine stack trace"} {
		if !contains(s, want) {
			t.Fatalf("missing %q in dump", want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
