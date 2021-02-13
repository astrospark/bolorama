package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const boloHeaderSize = 8

const boloSignature = "Bolo"

const boloPacketTypeOffset = 0x07
const boloPacketTypeGameState = 0x02
const boloPacketTypeGameStateAck = 0x04
const boloPacketType5 = 0x05
const boloPacketType6 = 0x06
const boloPacketType7 = 0x07
const boloPacketType8 = 0x08
const boloPacketType9 = 0x09
const boloPacketTypeGameInfo = 0x0e

const boloPacketType6PeerAddrOffset = 12
const boloPacketType6PeerPortOffset = 16
const boloPacketType7PeerAddrOffset = 12
const boloPacketType7PeerPortOffset = 16
const boloPacketType9PeerAddrOffset = 8
const boloPacketType9PeerPortOffset = 12

const boloMinesVisibleBitmask = 1 << 6

const trackerPort = 50000
const firstGamePort = 50001
const firstPlayerPort = 40001

/* Macs count time since Midnight, 1st Jan 1904. Unix counts from 1970.
   This value adjusts for the 66 years and 17 leap-days difference. */
const seconds1904ToUnixEpoch = (((1970-1904)*365 + 17) * 24 * 60 * 60)

type boloGameInfo struct {
	gameID              [8]byte
	mapName             string
	hostAddr            net.UDPAddr
	startTimestamp      uint32
	gameType            int
	allowHiddenMines    bool
	allowComputer       bool
	computerAdvantage   bool
	startDelay          uint32
	timeLimit           uint32
	playerCount         uint16
	neutralPillboxCount uint16
	neutralBaseCount    uint16
	hasPassword         bool
}

func parsePacketGameInfo(msg []byte, hostAddr net.UDPAddr) boloGameInfo {
	var gameInfo boloGameInfo
	var pos int = 0

	gameInfo.mapName = string(msg[1 : msg[0]+1])
	pos = pos + 36

	copy(gameInfo.gameID[:], msg[pos:pos+8])

	gameInfo.hostAddr = hostAddr
	pos = pos + 4

	gameInfo.startTimestamp = binary.BigEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.gameType = int(msg[pos])
	pos = pos + 1

	gameInfo.allowHiddenMines = !((msg[pos] & boloMinesVisibleBitmask) == boloMinesVisibleBitmask)
	pos = pos + 1

	gameInfo.allowComputer = msg[pos] > 0
	pos = pos + 1

	gameInfo.computerAdvantage = msg[pos] > 0
	pos = pos + 1

	gameInfo.startDelay = binary.LittleEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.timeLimit = binary.LittleEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.playerCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.neutralPillboxCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.neutralBaseCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.hasPassword = msg[pos] > 0
	pos = pos + 1

	return gameInfo
}

func printGameInfo(gameInfo boloGameInfo) {
	fmt.Println()
	fmt.Println("Game ID:", gameInfo.gameID)
	fmt.Println("Map Name:", gameInfo.mapName)
	fmt.Println("Host:", gameInfo.hostAddr.IP.String()+":"+strconv.Itoa(gameInfo.hostAddr.Port))
	fmt.Println("Start Timestamp:", time.Unix(int64(gameInfo.startTimestamp-seconds1904ToUnixEpoch), 0))
	fmt.Println("Game Type:", gameInfo.gameType)
	fmt.Println("Allow Hidden Mines:", gameInfo.allowHiddenMines)
	fmt.Println("Allow Computer:", gameInfo.allowComputer)
	fmt.Println("Computer Advantage:", gameInfo.computerAdvantage)
	fmt.Println("Start Delay:", gameInfo.startDelay)
	fmt.Println("Time Limit:", gameInfo.timeLimit)
	fmt.Println("Player Count:", gameInfo.playerCount)
	fmt.Println("Neutral Pillbox Count:", gameInfo.neutralPillboxCount)
	fmt.Println("Neutral Base Count:", gameInfo.neutralBaseCount)
	fmt.Println("Password:", gameInfo.hasPassword)
}

// Get preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "1.1.1.1:1")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func translateBoloPacketFixedPosition(packet udpPacket, proxyPort int, proxyIP net.IP, offset int) {
	//fmt.Println()
	//fmt.Println(hex.Dump(packet.buffer))
	packetIP := packet.buffer[offset : offset+4]
	if !bytes.Equal(packetIP, proxyIP) {
		port := make([]byte, 2)
		binary.BigEndian.PutUint16(port, uint16(proxyPort))
		packet.buffer[offset+0] = proxyIP[0]
		packet.buffer[offset+1] = proxyIP[1]
		packet.buffer[offset+2] = proxyIP[2]
		packet.buffer[offset+3] = proxyIP[3]
		packet.buffer[offset+4] = port[0]
		packet.buffer[offset+5] = port[1]
		//fmt.Println()
		//fmt.Println(hex.Dump(packet.buffer))
	}
}

func translateBoloPacket(packet udpPacket, srcRoute route, proxyIP net.IP) {
	// only the player who starts the game will send packets with the wrong ip address, and it will
	// be their own. so we can search for any ip that isn't ours, replace it with ours, and replace
	// the port with the player's assigned port

	switch packet.buffer[boloPacketTypeOffset] {
	case boloPacketTypeGameInfo:
		// is it necessary to re-write this one? if it is used by the former upstream of the player
		// leaving to identify their new downstream, then as long as they are looking for themselves
		// and not the player leaving, then it shouldn't be necessary to change the "middle" ip.
		//
		// upstream -> middle -> downstream
		//     upstream recognizes themselves, changes their downstream
		//
		// upstream -> middle -> downstream
		//     upstream recognizes middle as their downstream, changes their downstream
	case boloPacketType6:
		translateBoloPacketFixedPosition(packet, srcRoute.proxyPort, proxyIP, boloPacketType6PeerAddrOffset)
	case boloPacketType7:
		translateBoloPacketFixedPosition(packet, srcRoute.proxyPort, proxyIP, boloPacketType7PeerAddrOffset)
	case boloPacketType9:
		translateBoloPacketFixedPosition(packet, srcRoute.proxyPort, proxyIP, boloPacketType9PeerAddrOffset)
	}
}

func printRouteTable(gameIDRouteTableMap map[[8]byte][]route) {
	fmt.Println()
	fmt.Println("Route Table")
	for gameID, routeTable := range gameIDRouteTableMap {
		fmt.Println(" ", gameID)
		for _, route := range routeTable {
			fmt.Println("   ", route.playerIPAddr, route.proxyPort)
		}
	}
}

// The largest safe UDP packet length is 576 for IPv4 and 1280 for IPv6, where
// "safe" is defined as “guaranteed to be able to be reassembled, if fragmented."
const bufferSize = 1024

func verifyBoloSignature(msg []byte) bool {
	return string(msg[0:4]) == boloSignature
}

func verifyBoloVersion(msg []byte) bool {
	return bytes.Equal(msg[4:7], []byte{0x00, 0x99, 0x07})
}

func getBoloPacketType(msg []byte) int {
	return int(msg[7])
}

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

type route struct {
	playerIPAddr   net.UDPAddr
	proxyPort      int
	rewrite        bool
	rxChannel      chan udpPacket
	txChannel      chan udpPacket
	controlChannel chan struct{}
}

type udpPacket struct {
	srcAddr net.UDPAddr
	dstAddr net.UDPAddr
	dstPort int
	len     int
	buffer  []byte
}

func createPlayerProxy(playerRoute route) {
	fmt.Println()
	fmt.Printf("Creating proxy: %d => %s:%d", playerRoute.proxyPort, playerRoute.playerIPAddr.IP.String(), playerRoute.playerIPAddr.Port)

	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprint(":", playerRoute.proxyPort))
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	go udpListener(connection, playerRoute)
	go udpTransmitter(connection, playerRoute)
}

func udpListener(connection *net.UDPConn, playerRoute route) {
	buffer := make([]byte, bufferSize)

	go func() {
		for {
			_, ok := <-playerRoute.controlChannel
			if !ok {
				connection.Close()
			}
		}
	}()

	for {
		n, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on port", playerRoute.proxyPort)
			break
		}

		data := make([]byte, n)
		copy(data, buffer)
		playerRoute.rxChannel <- udpPacket{*addr, net.UDPAddr{}, playerRoute.proxyPort, n, data}
	}
}

func udpTransmitter(connection *net.UDPConn, playerRoute route) {
	for {
		select {
		case _, ok := <-playerRoute.controlChannel:
			if !ok {
				break
			}
		case data := <-playerRoute.txChannel:
			_, err := connection.WriteToUDP(data.buffer, &data.dstAddr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func udpTrackerListener(port int, dataChannel chan udpPacket, controlChannel chan int) {
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
			stopPort := <-controlChannel
			if stopPort == port {
				connection.Close()
			}
		}
	}()

	fmt.Println("Listening on port", port)

	for {
		n, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on port", port)
			break
		}

		data := make([]byte, n)
		copy(data, buffer)
		dataChannel <- udpPacket{*addr, net.UDPAddr{}, port, n, data}
	}
}

func newPlayerRoute(addr net.UDPAddr, port int, rewrite bool, rxChannel chan udpPacket) route {
	txChannel := make(chan udpPacket)
	controlChannel := make(chan struct{})

	return route{
		addr,
		port,
		rewrite,
		rxChannel,
		txChannel,
		controlChannel,
	}
}

func getRouteByAddr(gameIDRouteTableMap map[[8]byte][]route, addr net.UDPAddr) (route, error) {
	for _, routes := range gameIDRouteTableMap {
		for _, route := range routes {
			if addr.IP.Equal(route.playerIPAddr.IP) && addr.Port == route.playerIPAddr.Port {
				return route, nil
			}
		}
	}

	return route{}, fmt.Errorf("Error: Socket %s:%d not found in routing tables", addr.IP.String(), addr.Port)
}

func getRouteByPort(gameIDRouteTableMap map[[8]byte][]route, port int) ([8]byte, route, error) {
	for gameID, routes := range gameIDRouteTableMap {
		for _, route := range routes {
			if port == route.proxyPort {
				return gameID, route, nil
			}
		}
	}

	return [8]byte{}, route{}, fmt.Errorf("Error: Port %d not found in routing tables", port)
}

func main() {
	const boloPort = 50000
	gameIDRouteTableMap := make(map[[8]byte][]route)
	var assignedPlayerPorts []int
	rxChannel := make(chan udpPacket)
	trackerControlChannel := make(chan int)
	proxyIP := getOutboundIP()

	go udpTrackerListener(boloPort, rxChannel, trackerControlChannel)

	for {
		data := <-rxChannel

		if data.len < boloHeaderSize {
			fmt.Println("datagram too short (smaller than bolo header)")
			continue
		}

		if !verifyBoloSignature(data.buffer) {
			fmt.Println("datagram failed bolo signature check")
			continue
		}

		if !verifyBoloVersion(data.buffer) {
			fmt.Println("unsupported bolo version")
			continue
		}

		packetType := getBoloPacketType(data.buffer)

		packet := data.buffer[boloHeaderSize:]

		switch packetType {
		case boloPacketTypeGameInfo:
			if data.dstPort != trackerPort {
				// ignore tracker packets except on tracker port
				break
			}

			gameInfo := parsePacketGameInfo(packet, data.srcAddr)
			printGameInfo(gameInfo)

			_, ok := gameIDRouteTableMap[gameInfo.gameID]
			if !ok {
				fmt.Println()
				fmt.Println("assignedPlayerPorts", assignedPlayerPorts)
				nextPlayerPort := getNextAvailablePort(firstPlayerPort, &assignedPlayerPorts)
				fmt.Println("nextPlayerPort", nextPlayerPort)
				fmt.Println("assignedPlayerPorts", assignedPlayerPorts)

				rewrite := true
				playerRoute := newPlayerRoute(data.srcAddr, nextPlayerPort, rewrite, rxChannel)
				gameIDRouteTableMap[gameInfo.gameID] = []route{playerRoute}
				createPlayerProxy(playerRoute)
			}

			printRouteTable(gameIDRouteTableMap)

		default:
			if data.dstPort == trackerPort {
				fmt.Println("dropping non-tracker packet received on tracker port")
				break
			}

			// get destination player ip by proxy port
			gameID, dstRoute, err := getRouteByPort(gameIDRouteTableMap, data.dstPort)
			if err != nil {
				// shouldn't be able to receive data on a port that isn't mapped
				fmt.Println(err)
				continue
			}

			// get proxy port by source player ip
			srcRoute, err := getRouteByAddr(gameIDRouteTableMap, data.srcAddr)
			if err != nil {
				nextPlayerPort := getNextAvailablePort(firstPlayerPort, &assignedPlayerPorts)
				rewrite := false
				srcRoute = newPlayerRoute(data.srcAddr, nextPlayerPort, rewrite, rxChannel)
				gameIDRouteTableMap[gameID] = append(gameIDRouteTableMap[gameID], srcRoute)
				createPlayerProxy(srcRoute)

				printRouteTable(gameIDRouteTableMap)
			}

			//fmt.Printf("%d bytes from %s:%d (rewrite: %t)\n", data.len, data.srcAddr.IP.String(), data.srcAddr.Port, srcRoute.rewrite)

			if srcRoute.rewrite {
				translateBoloPacket(data, srcRoute, proxyIP)
			}

			if bytes.Contains(data.buffer, []byte{0xC0, 0xA8, 0x00, 0x50}) {
				fmt.Println()
				fmt.Println("Warning: Outgoing packet matches 192.168.0.80")
				fmt.Printf("Src: %s:%d Dst: %s%d\n",
					srcRoute.playerIPAddr.IP.String(), srcRoute.playerIPAddr.Port,
					dstRoute.playerIPAddr.IP.String(), dstRoute.playerIPAddr.Port)
				fmt.Println(hex.Dump(data.buffer))
			}

			data.dstAddr = dstRoute.playerIPAddr
			srcRoute.txChannel <- data
		}
	}
}
