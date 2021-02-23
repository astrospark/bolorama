package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/state"
	"git.astrospark.com/bolorama/tracker"
	"git.astrospark.com/bolorama/util"
)

func initSignalHandler(shutdownChannel chan struct{}) {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChannel
		close(shutdownChannel)
	}()
}

func main() {
	proxyHostname := config.GetValueString("hostname")

	context := state.InitContext()
	playerLeaveGameChannel := make(chan util.PlayerAddr)
	startPlayerPingChannel := make(chan state.Player)

	fmt.Println("Hostname:", proxyHostname)
	fmt.Println("IP Address:", context.ProxyIpAddr)

	defer func() {
		fmt.Println("Shutdown completed")
	}()

	initSignalHandler(context.ShutdownChannel)

	context.WaitGroup.Add(1)
	go tracker.Tracker(
		context,
		startPlayerPingChannel,
	)

loop:
	for {
		select {
		case _, ok := <-context.ShutdownChannel:
			if !ok {
				break loop
			}
		case playerPort := <-playerLeaveGameChannel:
			state.PlayerDelete(context, playerPort, true)
			state.PrintServerState(context, true)
		case packet := <-context.RxChannel:
			processPacket(context, packet, startPlayerPingChannel, playerLeaveGameChannel)
		}
	}

	context.WaitGroup.Wait()
}

func processPacket(context *state.ServerContext, packet proxy.UdpPacket, startPlayerPingChannel chan state.Player, playerLeaveGameChannel chan util.PlayerAddr) {
	valid, _ := bolo.ValidatePacket(packet)
	if !valid {
		// skip non-bolo packets
		return
	}

	packetType := bolo.GetPacketType(packet.Buffer)

	context.Mutex.Lock()

	// get destination player ip by proxy port
	dstPlayer, err := state.PlayerGetByPort(context, packet.DstPort, false)
	if err != nil {
		// normally won't happen, but there could be a pending packet incoming from a player that was subsequently deleted
		fmt.Println(err)
		context.Mutex.Unlock()
		return
	}

	// get proxy port by source player ip
	srcPlayer, err := state.PlayerGetByAddr(context, packet.SrcAddr, false)
	if err != nil {
		srcPlayer = state.PlayerNew(context, packet.SrcAddr, dstPlayer.GameId, false)
		startPlayerPingChannel <- srcPlayer
		state.PrintServerState(context, false)
	}

	if packetType == bolo.PacketType5 {
		state.PlayerJoinGame(context, srcPlayer.ProxyPort, dstPlayer.GameId, false)
	}

	context.Mutex.Unlock()

	go forwardPacket(packet, context.ProxyIpAddr, srcPlayer, dstPlayer, playerLeaveGameChannel)
}

func forwardPacket(packet proxy.UdpPacket, proxyIP net.IP, srcPlayer state.Player, dstPlayer state.Player, playerLeaveGameChannel chan util.PlayerAddr) {
	srcPlayerAddr := util.PlayerAddr{IpAddr: srcPlayer.IpAddr.String(), IpPort: srcPlayer.IpPort, ProxyPort: srcPlayer.ProxyPort}
	bolo.RewritePacket(packet.Buffer, proxyIP, srcPlayer.ProxyPort, srcPlayerAddr, playerLeaveGameChannel)

	if bytes.Contains(packet.Buffer, []byte{0xC0, 0xA8, 0x00, 0x50}) {
		fmt.Println()
		fmt.Println("Warning: Outgoing packet matches 192.168.0.80")
		fmt.Printf("Src: %s:%d Dst: %s:%d\n",
			srcPlayer.IpAddr.String(), srcPlayer.IpPort,
			dstPlayer.IpAddr.String(), dstPlayer.IpPort)
		fmt.Println(hex.Dump(packet.Buffer))
	}

	packet.DstAddr = net.UDPAddr{IP: dstPlayer.IpAddr, Port: dstPlayer.IpPort}
	srcPlayer.TxChannel <- packet
}
