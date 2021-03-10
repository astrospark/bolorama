package tracker

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/state"
	"git.astrospark.com/bolorama/util"
)

func Tracker(
	context *state.ServerContext,
	startPlayerPingChannel chan state.Player,
) {
	defer context.WaitGroup.Done()
	defer func() {
		fmt.Println("Stopped tracker")
	}()
	udpPacketChannel := make(chan proxy.UdpPacket)
	tcpRequestChannel := make(chan net.Conn)
	playerPongChannel := make(chan util.PlayerAddr)
	playerPingTimeoutChannel := make(chan util.PlayerAddr)
	trackerShutdownChannel := make(chan struct{})
	hostname := config.GetValueString("hostname")
	port := config.GetValueInt("tracker_port")
	proxyIp := util.GetOutboundIp()
	wg := sync.WaitGroup{}

	wg.Add(3)
	go udpListener(&wg, context.ShutdownChannel, context.UdpConnection, port, udpPacketChannel)
	go tcpListener(&wg, context.ShutdownChannel, port, tcpRequestChannel)
	go pingTimeout(&wg, context.ShutdownChannel, playerPongChannel, playerPingTimeoutChannel)

	go func() {
		wg.Wait()
		close(trackerShutdownChannel)
	}()

	for {
		select {
		case _, ok := <-trackerShutdownChannel:
			if !ok {
				return
			}
		case packet := <-udpPacketChannel:
			player, err := state.PlayerGetByAddr(context, packet.SrcAddr, true)
			if err == nil {
				playerPongChannel <- util.PlayerAddr{IpAddr: player.IpAddr.String(), IpPort: player.IpPort, ProxyPort: player.ProxyPort}
			}
			handleGameInfoPacket(context, proxyIp, port, packet, playerPongChannel)
		case conn := <-tcpRequestChannel:
			conn.Write([]byte(getTrackerText(context, hostname)))
			conn.Close()
		case player := <-startPlayerPingChannel:
			playerPongChannel <- util.PlayerAddr{IpAddr: player.IpAddr.String(), IpPort: player.IpPort, ProxyPort: player.ProxyPort}
			go pingGameInfo(context.UdpConnection, player, context.ShutdownChannel)
		case playerAddr := <-playerPingTimeoutChannel:
			log.Printf("Player timed out %s:%d\n", playerAddr.IpAddr, playerAddr.IpPort)
			state.PlayerDelete(context, playerAddr, true)
			state.PrintServerState(context, true)
		}
	}
}

func handleGameInfoPacket(
	context *state.ServerContext,
	proxyIp net.IP,
	trackerPort int,
	packet proxy.UdpPacket,
	playerPongChannel chan util.PlayerAddr,
) {
	valid, _ := bolo.ValidatePacket(packet)
	if !valid {
		// skip non-bolo packets
		return
	}

	packetType := bolo.GetPacketType(packet.Buffer)
	if packetType != bolo.PacketTypeGameInfo {
		// ignore all packets except gameinfo ones
		return
	}

	// game id is more unique if we leave the original ip address
	//bolo.RewritePacketGameInfo(packet.Buffer, proxyIp)
	newGameInfo := bolo.ParsePacketGameInfo(packet.Buffer)

	context.Mutex.Lock()
	defer func() { context.Mutex.Unlock() }()

	newGame := false
	gameInfo, ok := context.Games[newGameInfo.GameId]
	if ok {
		newGameInfo.ServerStartTimestamp = gameInfo.ServerStartTimestamp
	} else {
		newGameInfo.ServerStartTimestamp = time.Now()
		newGame = true
		bolo.PrintGameInfo(newGameInfo)
	}
	context.Games[newGameInfo.GameId] = newGameInfo

	player, err := state.PlayerGetByAddr(context, packet.SrcAddr, false)
	if err == nil {
		if player.GameId != newGameInfo.GameId {
			state.PlayerJoinGame(context, player.ProxyPort, newGameInfo.GameId, false)
			if player.NatPort != trackerPort {
				state.PlayerSetNatPort(context, util.PlayerAddr{IpAddr: player.IpAddr.String(), IpPort: player.IpPort, ProxyPort: player.ProxyPort}, 0, false)
			}
		}
	} else {
		player = state.PlayerNew(context, packet.SrcAddr, newGameInfo.GameId, trackerPort, false)
		playerPongChannel <- util.PlayerAddr{IpAddr: player.IpAddr.String(), IpPort: player.IpPort, ProxyPort: player.ProxyPort}
		go pingGameInfo(context.UdpConnection, player, context.ShutdownChannel)
		if newGame {
			state.PlayerSetId(context, util.PlayerAddr{IpAddr: player.IpAddr.String(), IpPort: player.IpPort, ProxyPort: player.ProxyPort}, 0, false)
		}
		state.PrintServerState(context, false)
	}
}

func pingGameInfo(
	connection *net.UDPConn,
	player state.Player,
	shutdownChannel chan struct{},
) {
	gameInfoPingSeconds := config.GetValueInt("game_info_ping_seconds")
	ticker := time.NewTicker(time.Duration(gameInfoPingSeconds) * time.Second)

	for {
		select {
		case <-player.DisconnectChannel:
			fmt.Println("Stopped pinging player", player.ProxyPort)
			ticker.Stop()
			return
		case <-shutdownChannel:
			fmt.Println("Stopped pinging player", player.ProxyPort)
			ticker.Stop()
			return
		case <-ticker.C:
			buffer, err := hex.DecodeString("426f6c6f6599080d")
			if err != nil {
				break
			}
			dstAddr := &net.UDPAddr{IP: player.IpAddr, Port: player.IpPort}
			connection.WriteToUDP(buffer, dstAddr)
		}
	}
}

func pingTimeout(
	wg *sync.WaitGroup,
	shutdownChannel chan struct{},
	playerPongChannel chan util.PlayerAddr,
	playerPingTimeoutChannel chan util.PlayerAddr,
) {
	defer wg.Done()
	gameInfoPingDuration := time.Duration(config.GetValueInt("game_info_ping_seconds")) * time.Second
	gameInfoTimeoutDuration := gameInfoPingDuration + (5 * time.Second)
	mapPlayerTimestamp := make(map[util.PlayerAddr]time.Time)
	ticker := time.NewTicker(gameInfoPingDuration / 4)

	for {
		select {
		case <-shutdownChannel:
			ticker.Stop()
			return
		case playerAddr := <-playerPongChannel:
			mapPlayerTimestamp[playerAddr] = time.Now()
		case <-ticker.C:
			for playerAddr, timestamp := range mapPlayerTimestamp {
				if time.Now().After(timestamp.Add(gameInfoTimeoutDuration)) {
					playerPingTimeoutChannel <- playerAddr
					delete(mapPlayerTimestamp, playerAddr)
				}
			}
		}
	}
}
