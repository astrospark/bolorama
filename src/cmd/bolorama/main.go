package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/tracker"
)

const trackerPort = 50000

// get preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "1.1.1.1:1")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func printRouteTable(gameIDRouteTableMap map[[8]byte][]proxy.Route) {
	fmt.Println()
	fmt.Println("Route Table")
	for gameID, routeTable := range gameIDRouteTableMap {
		fmt.Println(" ", gameID)
		for _, route := range routeTable {
			fmt.Println("   ", route.PlayerIPAddr, route.ProxyPort)
		}
	}
}

func initSignalHandler(shutdownChannel chan struct{}, proxyControlChannel chan int) {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChannel
		close(shutdownChannel)
		close(proxyControlChannel)
	}()
}

type context struct {
	gameIDRouteTableMap map[[8]byte][]proxy.Route
	rxChannel           chan proxy.UdpPacket
	gameInfoChannel     chan bolo.GameInfo
	proxyControlChannel chan int
	shutdownChannel     chan struct{}
	proxyIP             net.IP
	wg                  *sync.WaitGroup
}

func main() {
	proxyHostname := config.GetValue("hostname")

	var context context
	context.gameIDRouteTableMap = make(map[[8]byte][]proxy.Route)
	context.rxChannel = make(chan proxy.UdpPacket)
	context.gameInfoChannel = make(chan bolo.GameInfo)
	context.proxyControlChannel = make(chan int)
	context.shutdownChannel = make(chan struct{})
	context.proxyIP = getOutboundIP()
	context.wg = new(sync.WaitGroup)

	fmt.Println("Hostname:", proxyHostname)
	fmt.Println("IP Address:", context.proxyIP)

	defer func() {
		fmt.Println("Shutdown completed")
	}()

	initSignalHandler(context.shutdownChannel, context.proxyControlChannel)

	context.wg.Add(2)
	go tracker.UdpListener(context.wg, context.shutdownChannel, trackerPort, context.rxChannel)
	go tracker.Tracker(context.wg, context.shutdownChannel, context.proxyIP, trackerPort, context.gameInfoChannel)

loop:
	for {
		select {
		case _, ok := <-context.shutdownChannel:
			if !ok {
				break loop
			}
		case packet := <-context.rxChannel:
			processPacket(context, packet)
		}
	}

	context.wg.Wait()
}

func processPacket(context context, data proxy.UdpPacket) {
	valid, _ := bolo.ValidatePacket(data)
	if !valid {
		// skip non-bolo packets
		return
	}

	switch bolo.GetPacketType(data.Buffer) {
	case bolo.PacketTypeGameInfo:
		if data.DstPort != trackerPort {
			// ignore tracker packets except on tracker port
			break
		}

		bolo.RewritePacketGameInfo(data.Buffer, context.proxyIP)
		gameInfo := bolo.ParsePacketGameInfo(data.Buffer)
		context.gameInfoChannel <- gameInfo
		bolo.PrintGameInfo(gameInfo)

		// TODO: check if this player address+port is in any
		// other game. if so, remove it from that game before
		// creating new game.

		_, ok := context.gameIDRouteTableMap[gameInfo.GameID]
		if !ok {
			playerRoute := proxy.AddPlayer(context.wg, context.proxyControlChannel, data.SrcAddr, context.rxChannel)
			context.gameIDRouteTableMap[gameInfo.GameID] = []proxy.Route{playerRoute}
		}

		printRouteTable(context.gameIDRouteTableMap)

	default:
		if data.DstPort == trackerPort {
			// drop non-tracker packets received on tracker port
			break
		}

		// get destination player ip by proxy port
		gameID, dstRoute, err := proxy.GetRouteByPort(context.gameIDRouteTableMap, data.DstPort)
		if err != nil {
			// shouldn't be able to receive data on a port that isn't mapped
			fmt.Println(err)
			break
		}

		// get proxy port by source player ip
		srcRoute, err := proxy.GetRouteByAddr(context.gameIDRouteTableMap, data.SrcAddr)
		if err != nil {
			srcRoute = proxy.AddPlayer(context.wg, context.proxyControlChannel, data.SrcAddr, context.rxChannel)
			context.gameIDRouteTableMap[gameID] = append(context.gameIDRouteTableMap[gameID], srcRoute)

			printRouteTable(context.gameIDRouteTableMap)
		}

		bolo.RewritePacket(data.Buffer, context.proxyIP, srcRoute.ProxyPort)

		if bytes.Contains(data.Buffer, []byte{0xC0, 0xA8, 0x00, 0x50}) {
			fmt.Println()
			fmt.Println("Warning: Outgoing packet matches 192.168.0.80")
			fmt.Printf("Src: %s:%d Dst: %s:%d\n",
				srcRoute.PlayerIPAddr.IP.String(), srcRoute.PlayerIPAddr.Port,
				dstRoute.PlayerIPAddr.IP.String(), dstRoute.PlayerIPAddr.Port)
			fmt.Println(hex.Dump(data.Buffer))
		}

		data.DstAddr = dstRoute.PlayerIPAddr
		srcRoute.TxChannel <- data
	}
}
