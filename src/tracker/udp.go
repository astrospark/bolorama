package tracker

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"git.astrospark.com/bolorama/proxy"
)

// The largest safe UDP packet length is 576 for IPv4 and 1280 for IPv6, where
// "safe" is defined as “guaranteed to be able to be reassembled, if fragmented."
const bufferSize = 1024

func UdpListener(wg *sync.WaitGroup, shutdownChannel chan struct{}, port int, dataChannel chan proxy.UdpPacket) {
	defer wg.Done()

	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprint(":", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()
	buffer := make([]byte, bufferSize)

	go func() {
		for {
			_, ok := <-shutdownChannel
			if !ok {
				connection.Close()
				break
			}
		}
	}()

	fmt.Println("Listening on UDP port", port)

	for {
		n, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on UDP port", port)
			break
		}

		data := make([]byte, n)
		copy(data, buffer)
		dataChannel <- proxy.UdpPacket{*addr, net.UDPAddr{}, port, n, data}
	}
}
