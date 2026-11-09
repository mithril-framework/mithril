package db

import (
	"runtime"
	"testing"
)

func TestDefaultMaxConnsBounds(t *testing.T) {
	old := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(old)
	if got := DefaultMaxConns(); got != 10 {
		t.Fatalf("1 CPU: got %d, want floor 10", got)
	}
	runtime.GOMAXPROCS(64)
	if got := DefaultMaxConns(); got != 50 {
		t.Fatalf("64 CPUs: got %d, want cap 50", got)
	}
	runtime.GOMAXPROCS(6)
	if got := DefaultMaxConns(); got != 24 {
		t.Fatalf("6 CPUs: got %d, want 24", got)
	}
}
