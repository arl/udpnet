package main

import (
	"fmt"
	"net"
	"time"

	"github.com/aurelien-rainone/udp"
)

func main() {
	// create socket

	var (
		port   = 30000
		socket udp.Socket
	)

	fmt.Printf("creating socket on port %d\n", port)

	if err := socket.Open(port); err != nil {
		fmt.Printf("failed to create Socket: %v\n", err)
		return
	}
	defer socket.Close()

	// send and receive packets to ourself until the user ctrl-breaks...
	for {
		data := []byte("hello world!")

		ad, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+fmt.Sprintf("%v", port))
		if err != nil {
			fmt.Printf("can't resolve udp addr %v\n", ad)
			break
		}
		socket.Send(ad, data)

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

		time.Sleep(250 * time.Millisecond)
	}
}
