package scraper

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedLock_SameKeySerializes(t *testing.T) {
	k := newKeyedLock()
	var mu sync.Mutex
	var order []int
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			unlock := k.Lock("video-A")
			defer unlock()

			mu.Lock()
			order = append(order, n)
			mu.Unlock()

			time.Sleep(50 * time.Millisecond) // gia lap dang tai file
		}(i)
		time.Sleep(10 * time.Millisecond) // dam bao goroutine 0 vao lock truoc
	}
	wg.Wait()

	if len(order) != 2 {
		t.Fatalf("ca 2 goroutine phai chay xong tuan tu, got %d ket qua", len(order))
	}
}

func TestKeyedLock_DifferentKeysRunConcurrently(t *testing.T) {
	k := newKeyedLock()
	var running int32
	var maxConcurrent int32
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			unlock := k.Lock(key)
			defer unlock()

			cur := atomic.AddInt32(&running, 1)
			for {
				m := atomic.LoadInt32(&maxConcurrent)
				if cur <= m || atomic.CompareAndSwapInt32(&maxConcurrent, m, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&running, -1)
		}("video-" + string(rune('A'+i)))
	}
	wg.Wait()

	if maxConcurrent < 2 {
		t.Fatalf("cac videoID khac nhau phai chay song song duoc, maxConcurrent=%d", maxConcurrent)
	}
}

func TestKeyedLock_CleansUpMapAfterUnlock(t *testing.T) {
	k := newKeyedLock()
	unlock := k.Lock("video-X")
	unlock()

	k.mu.Lock()
	_, exists := k.locks["video-X"]
	k.mu.Unlock()

	if exists {
		t.Fatal("entry phai bi xoa khoi map sau khi khong con ai giu lock, tranh leak map vo han")
	}
}

func TestKeyedLock_RefCountHandlesOverlap(t *testing.T) {
	k := newKeyedLock()

	unlock1 := k.Lock("video-Y")
	go func() {
		time.Sleep(30 * time.Millisecond)
		unlock1()
	}()

	unlock2 := k.Lock("video-Y") // block cho toi khi unlock1 chay xong
	unlock2()

	k.mu.Lock()
	_, exists := k.locks["video-Y"]
	k.mu.Unlock()

	if exists {
		t.Fatal("sau khi ca 2 lan lock/unlock xong, entry phai duoc don")
	}
}

func TestCleanupStaleTempFiles_RemovesOldOnly(t *testing.T) {
	dir := os.TempDir()

	oldFile := filepath.Join(dir, "dopusold12345.opus")
	newFile := filepath.Join(dir, "dopusnew12345.opus")

	for _, f := range []string{oldFile, newFile} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("khong tao duoc file test: %v", err)
		}
	}
	defer os.Remove(oldFile)
	defer os.Remove(newFile)

	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("khong set duoc mtime: %v", err)
	}

	cleanupStaleTempFiles()

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("file qua 1 gio phai bi xoa")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatal("file moi khong duoc dong toi")
	}
}
