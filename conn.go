package udpnet

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// ConnCallback is the interface implemented by objects that handles the main
// events happening on a connection, start, connect, disconnect and stop.
type ConnCallback interface {
	OnStart()
	OnStop()
	OnConnect()
	OnDisconnect()
}

// ConnMode indicates the current mode of a connection, valid modes are None,
// Client and Server
type ConnMode int

const (
	// None is the mode of a connection not established yet.
	None ConnMode = iota

	// Client is the mode of a client connection.
	Client

	// Server is the mode of a server connection.
	Server
)

type connState int

const (
	disconnected = iota
	listening
	connecting
	connectFail
	connected
)

// Conn represents a Connection between two distant parties.
type Conn struct {
	protocolID         uint
	timeout            time.Duration
	running            bool
	mode               ConnMode
	state              connState
	socket             Socket
	timeoutAccumulator time.Duration
	address            *net.UDPAddr
	cb                 ConnCallback
}

// NewConn returns a new connection using given protocol id and timeout.
func NewConn(cb ConnCallback, protocolID uint, timeout time.Duration) *Conn {
	c := &Conn{
		protocolID: protocolID,
		timeout:    timeout,
		mode:       None,
		running:    false,
		cb:         cb,
	}
	c.clearData()
	return c
}

// Start initiates the connection on given port
func (c *Conn) Start(port int) bool {
	fmt.Printf("start connection on port %d\n", port)
	if err := c.socket.Open(port); err != nil {
		return false
	}
	c.running = true
	c.cb.OnStart()
	return true
}

// Stop immediately stops the connection and closes the underlying socket.
func (c *Conn) Stop() {
	fmt.Printf("stop connection\n")
	connected := c.IsConnected()
	c.clearData()
	c.socket.Close()
	c.running = false
	if connected {
		c.cb.OnDisconnect()
	}
	c.cb.OnStop()
}

// IsRunning indicates if the connection is currently running.
func (c *Conn) IsRunning() bool {
	return c.running
}

// Listen sets the connection mode as server and starts listening.
func (c *Conn) Listen() {
	fmt.Printf("server listening for connection\n")
	connected := c.IsConnected()
	c.clearData()
	if connected {
		c.cb.OnDisconnect()
	}
	c.mode = Server
	c.state = listening
}

// Connect sets the connection mode as client and tries to connect to the server
// at given address.
func (c *Conn) Connect(address *net.UDPAddr) {
	fmt.Printf("client connecting to %v\n", *address)
	isConnected := c.IsConnected()
	c.clearData()
	if isConnected {
		c.cb.OnDisconnect()
	}
	c.mode = Client
	c.state = connecting
	c.address = address
}

// IsConnecting indicates if the connection is currently trying to connect.
func (c *Conn) IsConnecting() bool {
	return c.state == connecting
}

// ConnectFailed indicates if the connection has failed.
func (c *Conn) ConnectFailed() bool {
	return c.state == connectFail
}

// IsConnected indicates if the connection has succeeded.
func (c *Conn) IsConnected() bool {
	return c.state == connected
}

// IsListening indicates if the connection is currently listening.
func (c *Conn) IsListening() bool {
	return c.state == listening
}

// GetMode returns the current connection mode.
func (c *Conn) GetMode() ConnMode {
	return c.mode
}

// Update updates the connection underlying state, reagarding elapsed time.
func (c *Conn) Update(dt time.Duration) {
	c.timeoutAccumulator += dt
	if c.timeoutAccumulator > c.timeout {
		if c.state == connecting {
			fmt.Printf("connect timed out\n")
			c.clearData()
			c.state = connectFail
			c.cb.OnDisconnect()
		} else if c.state == connected {
			fmt.Printf("connection timed out\n")
			c.clearData()
			if c.state == connecting {
				c.state = connectFail
			}
			c.cb.OnDisconnect()
		}
	}
}

// SendPacket sends a slice of data on the connection.
func (c *Conn) SendPacket(data []byte) error {
	if c.address == nil {
		return errors.New("address not set")
	}
	packet := make([]byte, len(data)+4)
	packet[0] = byte(c.protocolID >> 24)
	packet[1] = byte((c.protocolID >> 16) & 0xFF)
	packet[2] = byte((c.protocolID >> 8) & 0xFF)
	packet[3] = byte((c.protocolID) & 0xFF)
	copy(packet[4:], data)
	return c.socket.Send(c.address, packet)
}

// ReceivePacket received a slice of data from the connection.
func (c *Conn) ReceivePacket(data []byte) int {
	packet := make([]byte, len(data)+4)
	var sender net.UDPAddr
	bytesRead := c.socket.Receive(&sender, packet)
	if bytesRead == 0 {
		return 0
	}
	if bytesRead <= 4 {
		return 0
	}
	if packet[0] != byte(c.protocolID>>24) ||
		packet[1] != byte((c.protocolID>>16)&0xFF) ||
		packet[2] != byte((c.protocolID>>8)&0xFF) ||
		packet[3] != byte(c.protocolID&0xFF) {
		return 0
	}
	if c.mode == Server && !c.IsConnected() {
		fmt.Printf("server accepts connection from client %v\n",
			sender.String())
		c.state = connected
		c.address = &sender
		c.cb.OnConnect()
	}
	// TODO: Aurelien, should check if this the only way to compare two net.UDPAddr
	if sender.String() == c.address.String() {
		if c.mode == Client && c.state == connecting {
			fmt.Printf("client completes connection with server\n")
			c.state = connected
			c.cb.OnConnect()
		}
		c.timeoutAccumulator = time.Duration(0)
		copy(data, packet[4:bytesRead])
		return bytesRead - 4
	}
	return 0
}

// HeaderSize returns the size of the connection header.
func (c *Conn) HeaderSize() int {
	return 4
}

func (c *Conn) clearData() {
	c.state = disconnected
	c.timeoutAccumulator = time.Duration(0)
	c.address = nil
}
