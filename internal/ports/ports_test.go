package ports

import (
	"fmt"
	"net"
	"testing"
)

func TestFindAvailableReturnsPort(t *testing.T) {
	port, err := FindAvailable(18080, 10)
	if err != nil {
		t.Fatalf("FindAvailable failed: %v", err)
	}
	if port < 18080 {
		t.Errorf("port = %d, want >= 18080", port)
	}
}

func TestFindAvailableSkipsBusyPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	busyAddr := ln.Addr().(*net.TCPAddr)
	port, err := FindAvailable(busyAddr.Port, 5)
	if err != nil {
		t.Fatalf("FindAvailable failed: %v", err)
	}
	if port == busyAddr.Port {
		t.Errorf("returned busy port %d", port)
	}
}

func TestFindAvailableExhaustion(t *testing.T) {
	base := 28100
	listeners := make([]net.Listener, 0, 3)
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", base+i)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			for _, l := range listeners {
				_ = l.Close()
			}
			t.Fatalf("listen %s: %v", addr, err)
		}
		listeners = append(listeners, ln)
	}
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()

	_, err := FindAvailable(base, 3)
	if err == nil {
		t.Error("expected error when all ports are busy")
	}
}
