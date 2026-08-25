package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestFetchBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	dl := New(3)
	data, err := dl.FetchBytes(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}
}

func TestFetchBytesRetry(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	dl := New(3)
	data, err := dl.FetchBytes(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Errorf("got %q, want %q", string(data), "ok")
	}
}

func TestFetchToFile(t *testing.T) {
	content := []byte("file content here")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.Write(content)
	}))
	defer server.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "test.bin")

	dl := New(3)
	var lastPct float64
	err := dl.Fetch(context.Background(), server.URL, dest, func(downloaded, total int64, speed float64) {
		lastPct = float64(downloaded) / float64(total) * 100
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Errorf("file content: got %q, want %q", string(data), string(content))
	}
	if lastPct != 100 {
		t.Errorf("final progress: got %f, want 100", lastPct)
	}
}

func TestFetchResume(t *testing.T) {
	content := []byte("0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Write(content)
			return
		}
		var offset int
		fmt.Sscanf(rangeHeader, "bytes=%d-", &offset)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, len(content)-1, len(content)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)-offset))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(content[offset:])
	}))
	defer server.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "resume.bin")

	os.WriteFile(dest, content[:8], 0644)

	dl := New(3)
	err := dl.Fetch(context.Background(), server.URL, dest, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Errorf("resumed content: got %q, want %q", string(data), string(content))
	}
}

func TestFetchBytesAllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dl := New(2)
	_, err := dl.FetchBytes(context.Background(), server.URL)
	if err == nil {
		t.Error("expected error after all retries fail")
	}
}

func TestFetchContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 100; i++ {
			w.Write([]byte("x"))
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dl := New(0)
	dir := t.TempDir()
	err := dl.Fetch(ctx, server.URL, filepath.Join(dir, "test.bin"), nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestFetchRangeUnsatisfiableResets(t *testing.T) {
	content := []byte("0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.Write(content)
	}))
	defer server.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "resume.bin")
	os.WriteFile(dest, content[:4], 0644)

	dl := New(1)
	if err := dl.Fetch(context.Background(), server.URL, dest, nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Errorf("got %q, want %q", string(data), string(content))
	}
}

func TestFetchTruncatedBodyErrors(t *testing.T) {
	content := []byte("0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.Write(content[:8])
	}))
	defer server.Close()

	dir := t.TempDir()
	dl := New(0)
	err := dl.Fetch(context.Background(), server.URL, filepath.Join(dir, "test.bin"), nil)
	if err == nil {
		t.Error("expected error for truncated body")
	}
}
