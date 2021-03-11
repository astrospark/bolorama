package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

func listenNetShutdown(shutdownChannel chan struct{}) {
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprint(":", 49999))
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	const bufferSize = 1024
	buffer := make([]byte, bufferSize)

	_, _, err = connection.ReadFromUDP(buffer)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "use of closed network connection") {
			fmt.Println(err)
		}
		fmt.Println("Stopped listening on UDP port", 49999)
	}

	connection.Close()
	close(shutdownChannel)
}

func main() {
	proxyHostname := config.GetValueString("hostname")
	trackerPort := config.GetValueInt("tracker_port")

	context := state.InitContext(trackerPort)
	playerInfoEventChannel := make(chan util.PlayerInfoEvent)
	playerLeaveGameChannel := make(chan util.PlayerAddr)
	startPlayerPingChannel := make(chan state.Player)
	beginShutdownChannel := make(chan struct{})
	mainShutdownChannel := make(chan struct{})

	fmt.Println("Hostname:", proxyHostname)
	fmt.Println("IP Address:", context.ProxyIpAddr)

	defer func() {
		fmt.Println("Shutdown completed")
	}()

	initSignalHandler(beginShutdownChannel)
	go listenNetShutdown(beginShutdownChannel)

	context.WaitGroup.Add(1)
	go tracker.Tracker(
		context,
		startPlayerPingChannel,
	)

	go func() {
		<-beginShutdownChannel
		fmt.Println("Shutting down")
		close(context.ShutdownChannel)
		context.WaitGroup.Wait()
		close(mainShutdownChannel)
	}()

loop:
	for {
		select {
		case _, ok := <-mainShutdownChannel:
			if !ok {
				break loop
			}
		case playerInfo := <-playerInfoEventChannel:
			if playerInfo.SetId {
				state.PlayerSetId(context, playerInfo.PlayerAddr, playerInfo.PlayerId, true)
			} else if playerInfo.SetName {
				state.PlayerSetName(context, playerInfo.PlayerAddr, playerInfo.PlayerId, playerInfo.Name)
			}
		case playerPort := <-playerLeaveGameChannel:
			state.PlayerDelete(context, playerPort, true)
			state.PrintServerState(context, true)
		case packet := <-context.RxChannel:
			processPacket(context, packet, startPlayerPingChannel, playerInfoEventChannel, playerLeaveGameChannel)
		}
	}
}

func processPacket(
	context *state.ServerContext,
	packet proxy.UdpPacket,
	startPlayerPingChannel chan state.Player,
	playerInfoEventChannel chan util.PlayerInfoEvent,
	playerLeaveGameChannel chan util.PlayerAddr,
) {
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

	srcPlayer, err := state.PlayerGetByAddr(context, packet.SrcAddr, false)
	if err != nil {
		srcPlayer = state.PlayerNew(context, packet.SrcAddr, dstPlayer.GameId, dstPlayer.ProxyPort, false)
		startPlayerPingChannel <- srcPlayer
		state.PrintServerState(context, false)
	}

	context.PlayerPongChannel <- util.PlayerAddr{IpAddr: srcPlayer.IpAddr.String(), IpPort: srcPlayer.IpPort, ProxyPort: srcPlayer.ProxyPort}

	if packetType == bolo.PacketType5 {
		if srcPlayer.GameId != dstPlayer.GameId {
			state.PlayerJoinGame(context, srcPlayer.ProxyPort, dstPlayer.GameId, false)
		}
	}

	if context.Debug {
		if packetType == bolo.PacketType5 || packetType == bolo.PacketType6 || packetType == bolo.PacketType7 {
			srcTimestamp := srcPlayer.Peers[dstPlayer.ProxyPort]
			dstTimestamp := dstPlayer.Peers[srcPlayer.ProxyPort]
			timestamp := util.MaxTime(srcTimestamp, dstTimestamp)

			natStatus := "?"
			if time.Since(timestamp).Seconds() < 20 {
				natStatus = "*"
			}

			fmt.Printf("%s PacketType=%d %d (%s:%d) -> %d (%s:%d)\n", natStatus, packetType,
				srcPlayer.ProxyPort, srcPlayer.IpAddr.String(), srcPlayer.IpPort,
				dstPlayer.ProxyPort, dstPlayer.IpAddr.String(), dstPlayer.IpPort,
			)
			fmt.Printf("    Timestamp=%s\n", timestamp)
		}
	}

	if packetType == bolo.PacketType7 {
		if bytes.Equal(packet.Buffer[10:12], []byte{0x01, 0x23}) {
			if bytes.Equal(packet.Buffer[18:22], []byte{0x45, 0x67, 0x89, 0xab}) {
				savedPacket, ok := srcPlayer.PeerPackets[dstPlayer.ProxyPort]
				if !ok {
					fmt.Printf("received nat probe reply (%d -> %d, %s:%d -> %s:%d)\n", srcPlayer.ProxyPort, dstPlayer.ProxyPort, srcPlayer.IpAddr.String(), srcPlayer.IpPort, dstPlayer.IpAddr.String(), dstPlayer.IpPort)
					fmt.Printf("  error: no saved packet")
					context.Mutex.Unlock()
					return
				}
				if context.Debug {
					fmt.Printf("received nat probe reply (%d -> %d, %s:%d -> %s:%d)\n", srcPlayer.ProxyPort, dstPlayer.ProxyPort, srcPlayer.IpAddr.String(), srcPlayer.IpPort, dstPlayer.IpAddr.String(), dstPlayer.IpPort)
					fmt.Printf("  packet length = %d\n", len(savedPacket.Buffer))
					fmt.Printf("  forwarding PacketType=%d (%d -> %d, %s:%d -> %s:%d)\n", bolo.GetPacketType(savedPacket.Buffer), dstPlayer.ProxyPort, srcPlayer.ProxyPort, dstPlayer.IpAddr.String(), dstPlayer.IpPort, srcPlayer.IpAddr.String(), srcPlayer.IpPort)
				}
				delete(srcPlayer.PeerPackets, dstPlayer.ProxyPort)
				srcPlayer.Peers[dstPlayer.ProxyPort] = time.Now()
				context.Mutex.Unlock()
				go forwardPacket(savedPacket, context.ProxyIpAddr, dstPlayer, srcPlayer, playerInfoEventChannel, playerLeaveGameChannel)
				return
			}
		}
	}

	if srcPlayer.NatPort != context.ProxyPort {
		natProbe(context, srcPlayer, context.ProxyPort, false)
	}

	srcTimestamp := srcPlayer.Peers[dstPlayer.ProxyPort]
	dstTimestamp := dstPlayer.Peers[srcPlayer.ProxyPort]
	timestamp := util.MaxTime(srcTimestamp, dstTimestamp)
	if time.Since(timestamp).Seconds() > 20 {
		dstPlayer.PeerPackets[srcPlayer.ProxyPort] = packet
		natProbe(context, dstPlayer, srcPlayer.ProxyPort, false)
		context.Mutex.Unlock()
		return
	}

	srcPlayer.Peers[dstPlayer.ProxyPort] = time.Now()

	context.Mutex.Unlock()

	go forwardPacket(packet, context.ProxyIpAddr, srcPlayer, dstPlayer, playerInfoEventChannel, playerLeaveGameChannel)
}

func natProbe(context *state.ServerContext, dstPlayer state.Player, targetProxyPort int, lock bool) {
	trackerPort := config.GetValueInt("tracker_port")
	buffer := bolo.MarshalPacketType6(context.ProxyIpAddr, targetProxyPort)
	dstAddr := &net.UDPAddr{IP: dstPlayer.IpAddr, Port: dstPlayer.IpPort}

	if context.Debug {
		fmt.Printf("sending nat probe to %s:%d (target port: %d)\n", dstPlayer.IpAddr.String(), dstPlayer.IpPort, targetProxyPort)
	}

	if dstPlayer.NatPort == trackerPort {
		if context.Debug {
			fmt.Printf("  (nat probe source port: %d)\n", trackerPort)
		}
		context.UdpConnection.WriteToUDP(buffer, dstAddr)
	} else {
		natPlayer, err := state.PlayerGetByPort(context, dstPlayer.NatPort, lock)
		if err != nil {
			fmt.Println(err)
			return
		}
		if context.Debug {
			fmt.Printf("  (nat probe source port: %d)\n", natPlayer.ProxyPort)
		}
		natPlayer.TxChannel <- proxy.UdpPacket{DstAddr: *dstAddr, Buffer: buffer}
	}
}

func forwardPacket(
	packet proxy.UdpPacket,
	proxyIP net.IP,
	srcPlayer state.Player,
	dstPlayer state.Player,
	playerInfoEventChannel chan util.PlayerInfoEvent,
	playerLeaveGameChannel chan util.PlayerAddr,
) {
	srcPlayerAddr := util.PlayerAddr{IpAddr: srcPlayer.IpAddr.String(), IpPort: srcPlayer.IpPort, ProxyPort: srcPlayer.ProxyPort}
	bolo.RewritePacket(
		packet.Buffer,
		proxyIP,
		srcPlayer.ProxyPort,
		srcPlayerAddr,
		playerInfoEventChannel,
		playerLeaveGameChannel,
	)

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
