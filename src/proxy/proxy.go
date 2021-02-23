package proxy

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

const firstPlayerPort = 40001

// The largest safe UDP packet length is 576 for IPv4 and 1280 for IPv6, where
// "safe" is defined as “guaranteed to be able to be reassembled, if fragmented."
const bufferSize = 1024

// Route associates a proxy port with a player's real IP address + port
type Route struct {
	PlayerIPAddr      net.UDPAddr
	ProxyPort         int
	Connection        *net.UDPConn
	RxChannel         chan UdpPacket
	TxChannel         chan UdpPacket
	DisconnectChannel chan struct{}
}

// UdpPacket represents a packet being sent from srcAddr to dstAddr
type UdpPacket struct {
	SrcAddr net.UDPAddr
	DstAddr net.UDPAddr
	DstPort int
	Len     int
	Buffer  []byte
}

var assignedPlayerPorts []int

// 0 <= index <= len(a)
func insert(a []int, index int, value int) []int {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
}

func getNextAvailablePort(firstPort int, assignedPorts *[]int) int {
	nextPort := firstPort
	portCount := len(*assignedPorts)

	if portCount == 0 {
		*assignedPorts = append(*assignedPorts, nextPort)
		return nextPort
	}

	// use a first hole in port list, if one exists
	for i, port := range *assignedPorts {
		if port == nextPort {
			nextPort = port + 1
		} else {
			*assignedPorts = insert(*assignedPorts, i, nextPort)
			break
		}
	}

	lastPort := (*assignedPorts)[len(*assignedPorts)-1]
	if nextPort > lastPort {
		*assignedPorts = append(*assignedPorts, nextPort)
	}

	return nextPort
}

func DeletePort(port int) {
	idx := -1
	for i, value := range assignedPlayerPorts {
		if value == port {
			idx = i
			break
		}
	}

	if idx >= 0 {
		copy(assignedPlayerPorts[idx:], assignedPlayerPorts[idx+1:])
		assignedPlayerPorts = assignedPlayerPorts[:len(assignedPlayerPorts)-1]
	}
}

func AddPlayer(
	wg *sync.WaitGroup,
	playerAddr net.UDPAddr,
	rxChannel chan UdpPacket,
	disconnectChannel chan struct{},
	shutdownChannel chan struct{},
) (int, chan UdpPacket, *net.UDPConn) {
	if len(assignedPlayerPorts) > 1000 {
		// TODO this allows someone to deny service
		panic("maximum players exceeded (1000)")
	}
	nextPlayerPort := getNextAvailablePort(firstPlayerPort, &assignedPlayerPorts)
	playerRoute := newPlayerRoute(playerAddr, nextPlayerPort, rxChannel, disconnectChannel)
	createPlayerProxy(wg, playerRoute, shutdownChannel)
	return playerRoute.ProxyPort, playerRoute.TxChannel, playerRoute.Connection
}

func newPlayerRoute(addr net.UDPAddr, port int, rxChannel chan UdpPacket, disconnectChannel chan struct{}) Route {
	txChannel := make(chan UdpPacket)

	return Route{
		addr,
		port,
		nil,
		rxChannel,
		txChannel,
		disconnectChannel,
	}
}

func createPlayerProxy(wg *sync.WaitGroup, playerRoute Route, shutdownChannel chan struct{}) {
	fmt.Println()
	fmt.Printf("Creating proxy: %d => %s:%d\n", playerRoute.ProxyPort,
		playerRoute.PlayerIPAddr.IP.String(), playerRoute.PlayerIPAddr.Port)

	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprint(":", playerRoute.ProxyPort))
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	playerRoute.Connection = connection

	wg.Add(2)
	go udpListener(wg, shutdownChannel, playerRoute)
	go udpTransmitter(wg, shutdownChannel, playerRoute)
}

func udpListener(wg *sync.WaitGroup, shutdownChannel chan struct{}, playerRoute Route) {
	defer wg.Done()
	buffer := make([]byte, bufferSize)

	go func() {
		for {
			select {
			case _, ok := <-playerRoute.DisconnectChannel:
				if !ok {
					playerRoute.Connection.Close()
					return
				}
			case _, ok := <-shutdownChannel:
				if !ok {
					playerRoute.Connection.Close()
					return
				}
			}
		}
	}()

	for {
		n, addr, err := playerRoute.Connection.ReadFromUDP(buffer)
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on UDP port", playerRoute.ProxyPort)
			break
		}

		data := make([]byte, n)
		copy(data, buffer)
		playerRoute.RxChannel <- UdpPacket{*addr, net.UDPAddr{}, playerRoute.ProxyPort, n, data}
	}
}

func udpTransmitter(wg *sync.WaitGroup, shutdownChannel chan struct{}, playerRoute Route) {
	defer wg.Done()
	defer func() {
		fmt.Println("Stopped transmitting on UDP port", playerRoute.ProxyPort)
	}()

	for {
		select {
		case _, ok := <-playerRoute.DisconnectChannel:
			if !ok {
				return
			}
		case _, ok := <-shutdownChannel:
			if !ok {
				return
			}
		case data := <-playerRoute.TxChannel:
			_, err := playerRoute.Connection.WriteToUDP(data.Buffer, &data.DstAddr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
