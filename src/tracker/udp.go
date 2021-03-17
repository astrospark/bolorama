package tracker

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/util"
)

func udpListener(wg *sync.WaitGroup, shutdownChannel chan struct{}, connection *net.UDPConn, port int, dataChannel chan proxy.UdpPacket) {
	defer wg.Done()

	buffer := make([]byte, util.MaxUdpPacketSize)

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
		dataChannel <- proxy.UdpPacket{
			SrcAddr: *addr,
			DstAddr: net.UDPAddr{},
			DstPort: port,
			Len:     n,
			Buffer:  data,
		}
	}
}
