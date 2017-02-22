package udp

import (
	"fmt"
	"net"
	"testing"
)

func TestSocketOpenClose(t *testing.T) {
	var socket Socket
	if socket.IsOpen() {
		t.Fatalf("got isOpen() = true, want false")
	}
	if err := socket.Open(30000); err != nil {
		t.Fatalf("got Open(30000) = %v, want nil", err)
	}
	if !socket.IsOpen() {
		t.Fatalf("got isOpen() = false, want true")
	}
	socket.Close()
	if socket.IsOpen() {
		t.Fatalf("got isOpen() = true, want false")
	}
	if err := socket.Open(30000); err != nil {
		t.Fatalf("got Open(30000) = %v, want nil", err)
	}
	if !socket.IsOpen() {
		t.Fatalf("got isOpen() = false, want true")
	}
	socket.Close()
}

func TestSocketFailsOnSamePort(t *testing.T) {
	var a, b Socket

	if err := a.Open(30000); err != nil {
		t.Fatalf("got a.Open(30000) = %v, want nil", err)
	}
	defer a.Close()
	err := b.Open(30000)
	if err == nil {
		defer b.Close()
		t.Fatalf("got b.Open(30000) = nil, want != nil", err)
	}
	if e, ok := err.(*net.OpError); !ok {
		t.Fatalf("got b.Open(30000) = %v , want net.OpError", e)
	}
	if !a.IsOpen() {
		t.Fatalf("got a.isOpen() = false, want true")
	}
	if b.IsOpen() {
		t.Fatalf("got b.isOpen() = true, want false")
	}
}

func TestSocketSendReceivePackets(t *testing.T) {
	var a, b Socket

	if err := a.Open(30000); err != nil {
		t.Fatalf("got a.Open(30000) = %v, want nil", err)
	}
	defer a.Close()
	if err := b.Open(30001); err != nil {
		t.Fatalf("got b.Open(30001) = %v, want nil", err)
	}
	defer b.Close()

	packet := []byte("packet data")
	var aReceivedPacket, bReceivedPacket bool

	for !aReceivedPacket && !bReceivedPacket {
		dst, err := net.ResolveUDPAddr("udp", "127.0.0.1:30000")
		if err != nil {
			t.Fatalf("couldn't resolve udp address: %v", err)
		}

		// send packet to a
		if err := a.Send(dst, packet); err != nil {
			t.Fatalf("got a.Send() = %v, want nil", err)
		}
		// send packet to b
		if err := b.Send(dst, packet); err != nil {
			t.Fatalf("got b.Send() = %v, want nil", err)
		}

		// receive packet from a
		for {
			var (
				sender net.UDPAddr
				buf    [256]byte
			)
			bytesRead := a.Receive(&sender, buf[:])
			if bytesRead == 0 {
				fmt.Println("0 bytes read on a")
				break
			}
			if bytesRead == len(packet) && string(buf[:bytesRead]) == string(packet) {
				aReceivedPacket = true
			}
		}
		// receive packet from b
		for {
			var (
				sender net.UDPAddr
				buf    [256]byte
			)
			bytesRead := b.Receive(&sender, buf[:])
			if bytesRead == 0 {
				break
			}
			if bytesRead == len(packet) && string(buf[:bytesRead]) == string(packet) {
				bReceivedPacket = true
			}
		}
	}
}
