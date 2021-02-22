package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/tracker"
	"git.astrospark.com/bolorama/util"
)

func printRouteTable(routes []proxy.Route) {
	fmt.Println()
	fmt.Println("Route Table")
	for _, route := range routes {
		fmt.Printf("   %s:%d <%d>\n", route.PlayerIPAddr.IP, route.PlayerIPAddr.Port, route.ProxyPort)
	}
}

func initSignalHandler(shutdownChannel chan struct{}) {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChannel
		close(shutdownChannel)
	}()
}

type context struct {
	//gameIDRouteTableMap  map[[8]byte][]proxy.Route
	routes                  []proxy.Route
	rxChannel               chan proxy.UdpPacket
	gameStartChannel        chan tracker.GameStart
	newRouteChannel         chan tracker.NewRoute
	joinGameChannel         chan tracker.JoinGame
	trackerLeaveGameChannel chan proxy.Route
	leaveGameChannel        chan proxy.Route
	playerTimeoutChannel    chan proxy.Route
	shutdownChannel         chan struct{}
	proxyIP                 net.IP
	wg                      *sync.WaitGroup
}

func main() {
	proxyHostname := config.GetValueString("hostname")

	var context context
	context.rxChannel = make(chan proxy.UdpPacket)
	context.gameStartChannel = make(chan tracker.GameStart)
	context.newRouteChannel = make(chan tracker.NewRoute)
	context.joinGameChannel = make(chan tracker.JoinGame)
	context.trackerLeaveGameChannel = make(chan proxy.Route)
	context.leaveGameChannel = make(chan proxy.Route)
	context.playerTimeoutChannel = make(chan proxy.Route)
	context.shutdownChannel = make(chan struct{})
	context.proxyIP = util.GetOutboundIp()
	context.wg = new(sync.WaitGroup)

	fmt.Println("Hostname:", proxyHostname)
	fmt.Println("IP Address:", context.proxyIP)

	defer func() {
		fmt.Println("Shutdown completed")
	}()

	initSignalHandler(context.shutdownChannel)

	context.wg.Add(1)
	go tracker.Tracker(
		context.wg,
		context.shutdownChannel,
		context.gameStartChannel,
		context.newRouteChannel,
		context.joinGameChannel,
		context.trackerLeaveGameChannel,
		context.playerTimeoutChannel,
	)

loop:
	for {
		select {
		case _, ok := <-context.shutdownChannel:
			if !ok {
				break loop
			}
		case leaveGameRoute := <-context.playerTimeoutChannel:
			close(leaveGameRoute.DisconnectChannel)
			context.routes = proxy.DeleteRoute(context.routes, leaveGameRoute)
			context.trackerLeaveGameChannel <- leaveGameRoute
			printRouteTable(context.routes)
		case leaveGameRoute := <-context.leaveGameChannel:
			close(leaveGameRoute.DisconnectChannel)
			context.routes = proxy.DeleteRoute(context.routes, leaveGameRoute)
			context.trackerLeaveGameChannel <- leaveGameRoute
			printRouteTable(context.routes)
		case gameStart := <-context.gameStartChannel:
			playerRoute := proxy.AddPlayer(context.wg, context.shutdownChannel, gameStart.PlayerAddr, context.rxChannel)
			context.routes = append(context.routes, playerRoute)
			context.newRouteChannel <- tracker.NewRoute{PlayerRoute: playerRoute, GameId: gameStart.GameId}
			printRouteTable(context.routes)
		case packet := <-context.rxChannel:
			processPacket(&context, packet)
		}
	}

	context.wg.Wait()
}

func processPacket(context *context, packet proxy.UdpPacket) {
	valid, _ := bolo.ValidatePacket(packet)
	if !valid {
		// skip non-bolo packets
		return
	}

	packetType := bolo.GetPacketType(packet.Buffer)

	// get destination player ip by proxy port
	dstRoute, err := proxy.GetRouteByPort(context.routes, packet.DstPort)
	if err != nil {
		// shouldn't be able to receive data on a port that isn't mapped
		fmt.Println(err)
		return
	}

	// get proxy port by source player ip
	srcRoute, err := proxy.GetRouteByAddr(context.routes, packet.SrcAddr)
	if err != nil {
		srcRoute = proxy.AddPlayer(context.wg, context.shutdownChannel, packet.SrcAddr, context.rxChannel)
		context.routes = append(context.routes, srcRoute)
		context.newRouteChannel <- tracker.NewRoute{PlayerRoute: srcRoute, GameId: bolo.GameId{}}

		printRouteTable(context.routes)
	}

	if packetType == bolo.PacketType5 {
		context.joinGameChannel <- tracker.JoinGame{SrcProxyPort: srcRoute.ProxyPort, DstProxyPort: dstRoute.ProxyPort}
	}

	go forwardPacket(packet, context.proxyIP, srcRoute, dstRoute, context.leaveGameChannel)
}

func forwardPacket(packet proxy.UdpPacket, proxyIP net.IP, srcRoute proxy.Route, dstRoute proxy.Route, leaveGameChannel chan proxy.Route) {
	bolo.RewritePacket(packet.Buffer, proxyIP, srcRoute.ProxyPort, srcRoute, leaveGameChannel)

	if bytes.Contains(packet.Buffer, []byte{0xC0, 0xA8, 0x00, 0x50}) {
		fmt.Println()
		fmt.Println("Warning: Outgoing packet matches 192.168.0.80")
		fmt.Printf("Src: %s:%d Dst: %s:%d\n",
			srcRoute.PlayerIPAddr.IP.String(), srcRoute.PlayerIPAddr.Port,
			dstRoute.PlayerIPAddr.IP.String(), dstRoute.PlayerIPAddr.Port)
		fmt.Println(hex.Dump(packet.Buffer))
	}

	packet.DstAddr = dstRoute.PlayerIPAddr
	srcRoute.TxChannel <- packet
}
