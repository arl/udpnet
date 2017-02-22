package udp

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// A Socket represents an UDP socket
type Socket struct {
	conn *net.UDPConn
}

// Open binds the socket to 127.0.0.1:port
func (s *Socket) Open(port int) error {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("127.0.0.1"),
	}
	// bind socket
	var err error
	s.conn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}
	return err
}

// Close closes the underlying UDP socket
func (s *Socket) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
	s.conn = nil
}

// IsOpen returns true if the socket has been successfully opened
func (s *Socket) IsOpen() bool {
	return s.conn != nil
}

// Send writes the data buffer on addr
func (s *Socket) Send(addr *net.UDPAddr, data []byte) error {
	var (
		written int // bytes written
		err     error
	)

	// set non-blocking io
	deadline := time.Now().Add(100 * time.Millisecond)
	err = s.conn.SetDeadline(deadline)
	if err != nil {
		s.Close()
		return err
	}

	if s.conn != nil {
		written, err = s.conn.WriteToUDP(data, addr)
		switch {
		case err != nil:
			fallthrough
		case written == len(data):
			break
		default:
			err = errors.New("Socket.Send: not all data was sent")
		}
		return err
	}
	return errors.New("Socket.Send: no connection")
}

// Receive receives data from the socket, sets addr afterwards and returns the
// number of bytes received
func (s *Socket) Receive(addr *net.UDPAddr, data []byte) int {
	var (
		received int // bytes received
		err      error
	)

	// set non-blocking io
	deadline := time.Now().Add(100 * time.Millisecond)
	err = s.conn.SetDeadline(deadline)
	if err != nil {
		s.Close()
		fmt.Printf("couldn't set deadline in Socket.Receive: %v\n", err)
		return 0
	}

	var rAddr *net.UDPAddr
	received, rAddr, err = s.conn.ReadFromUDP(data)
	netErr, ok := err.(net.Error)
	switch {
	case err == nil:
		break
	case ok && netErr.Timeout():
		break
	default:
		fmt.Printf("Socket.Receive error: %v\n", err)
	}

	if rAddr != nil {
		*addr = *rAddr
	}

	return received
}
