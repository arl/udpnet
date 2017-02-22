package udp

import (
	"net"
	"testing"
)

func TestSocketOpenClose(t *testing.T) {
	var socket Socket
	if socket.IsOpen() {
		t.Fatalf("got isOpen() = true, want false")
	}
	if err := socket.Open(30000); err != nil {
		t.Fatalf("got Open(3000) = %v, want nil", err)
	}
	if !socket.IsOpen() {
		t.Fatalf("got isOpen() = false, want true")
	}
	socket.Close()
	if socket.IsOpen() {
		t.Fatalf("got isOpen() = true, want false")
	}
	if err := socket.Open(30000); err != nil {
		t.Fatalf("got Open(3000) = %v, want nil", err)
	}
	if !socket.IsOpen() {
		t.Fatalf("got isOpen() = false, want true")
	}
	socket.Close()
}

func TestSocketFailsOnSamePort(t *testing.T) {
	var a, b Socket

	if err := a.Open(30000); err != nil {
		t.Fatalf("got a.Open(3000) = %v, want nil", err)
	}
	defer a.Close()
	err := b.Open(30000)
	if err == nil {
		defer b.Close()
		t.Fatalf("got b.Open(3000) = nil, want != nil", err)
	}
	if e, ok := err.(*net.OpError); !ok {
		t.Fatalf("got b.Open(3000) = %v , want net.OpError", e)
	}
	if !a.IsOpen() {
		t.Fatalf("got a.isOpen() = false, want true")
	}
	if b.IsOpen() {
		t.Fatalf("got b.isOpen() = true, want false")
	}
}
