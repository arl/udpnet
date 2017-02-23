package main

import (
	"fmt"
	"net"
	"time"

	"github.com/aurelien-rainone/udp"
)

func main() {
	port := 30000
	fmt.Printf("creating socket on port %d\n", port)

	var socket udp.Socket
	if err := socket.Open(port); err != nil {
		fmt.Printf("failed to create Socket: %v\n", err)
		return
	}
	defer socket.Close()

	// read in addresses.txt to get the set of addresses we will send packets to
	addresses := []string{
		"127.0.0.1:30000",
		"127.0.0.1:30001",
		"127.0.0.1:30002",
		"127.0.0.1:30003",
		"127.0.0.1:30004",
	}

	// send and receive packets until the user ctrl-breaks...
	for {
		data := []byte("hello world!")
		for _, as := range addresses {
			ad, err := net.ResolveUDPAddr("udp", as)
			if err != nil {
				fmt.Printf("can't resolve udp addr %v\n", ad)
				break
			}
			socket.Send(ad, data)
		}

		for {
			var (
				sender net.UDPAddr
				buf    [256]byte
			)
			bytesRead := socket.Receive(&sender, buf[:])
			if bytesRead == 0 {
				break
			}

			fmt.Printf("received packet from %v (%d bytes): '%v'\n",
				sender.String(), bytesRead, string(buf[:bytesRead]))
		}

		time.Sleep(1000 * time.Millisecond)
	}
}
