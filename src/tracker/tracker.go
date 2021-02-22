package tracker

import (
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/util"
)

type GameStart struct {
	PlayerAddr net.UDPAddr
	GameId     bolo.GameId
}

type NewRoute struct {
	PlayerRoute proxy.Route
	GameId      bolo.GameId
}

type JoinGame struct {
	SrcProxyPort int
	DstProxyPort int
}

type PlayerName struct {
	ProxyPort int
	Name      string
}

type PlayerPong struct {
	ProxyPort int
	Timestamp time.Time
}

type GameState struct {
	routes                      []proxy.Route
	mapGameIdGameInfo           map[bolo.GameId]bolo.GameInfo
	mapProxyPortGameId          map[int]bolo.GameId
	mapProxyPortPlayerName      map[int]string
	mapProxyPortPingStopChannel map[int]chan struct{}
}

func Tracker(
	wg *sync.WaitGroup,
	shutdownChannel chan struct{},
	gameStartChannel chan GameStart,
	newRouteChannel chan NewRoute,
	joinGameChannel chan JoinGame,
	leaveGameChannel chan proxy.Route,
	playerTimeoutChannel chan proxy.Route,
) {
	defer wg.Done()
	udpPacketChannel := make(chan proxy.UdpPacket)
	tcpRequestChannel := make(chan net.Conn)
	playerPongChannel := make(chan PlayerPong)
	playerPingTimeoutChannel := make(chan int)
	hostname := config.GetValueString("hostname")
	port := config.GetValueInt("tracker_port")
	proxyIp := util.GetOutboundIp()
	gameState := initGameState()

	udpConnection := connectUdp(port)

	wg.Add(3)
	go udpListener(wg, shutdownChannel, udpConnection, port, udpPacketChannel)
	go tcpListener(wg, shutdownChannel, port, tcpRequestChannel)
	go pingTimeout(wg, shutdownChannel, playerPongChannel, playerPingTimeoutChannel)

	for {
		select {
		case _, ok := <-shutdownChannel:
			if !ok {
				return
			}
		case playerPort := <-playerPingTimeoutChannel:
			for _, route := range gameState.routes {
				if route.ProxyPort == playerPort {
					playerTimeoutChannel <- route
					break
				}
			}
		case leaveGameRoute := <-leaveGameChannel:
			gameId := gameState.mapProxyPortGameId[leaveGameRoute.ProxyPort]
			delete(gameState.mapProxyPortGameId, leaveGameRoute.ProxyPort)
			delete(gameState.mapProxyPortPlayerName, leaveGameRoute.ProxyPort)
			close(gameState.mapProxyPortPingStopChannel[leaveGameRoute.ProxyPort])
			delete(gameState.mapProxyPortPingStopChannel, leaveGameRoute.ProxyPort)
			gameState.routes = proxy.DeleteRoute(gameState.routes, leaveGameRoute)
			playerCount := countGamePlayers(gameState, gameId)
			if playerCount == 0 {
				delete(gameState.mapGameIdGameInfo, gameId)
			} else {
				updatePlayerCount(gameState, gameId, playerCount)
			}
		case newRoute := <-newRouteChannel:
			gameState.routes = append(gameState.routes, newRoute.PlayerRoute)
			if newRoute.GameId != (bolo.GameId{}) {
				gameState.mapProxyPortGameId[newRoute.PlayerRoute.ProxyPort] = newRoute.GameId
				playerCount := countGamePlayers(gameState, newRoute.GameId)
				updatePlayerCount(gameState, newRoute.GameId, playerCount)
			}
			pingStopChannel := make(chan struct{})
			gameState.mapProxyPortPingStopChannel[newRoute.PlayerRoute.ProxyPort] = pingStopChannel
			go pingGameInfo(udpConnection, newRoute.PlayerRoute, pingStopChannel)
		case joinGame := <-joinGameChannel:
			gameId, ok := gameState.mapProxyPortGameId[joinGame.DstProxyPort]
			if ok {
				playerJoinGame(gameState, joinGame.SrcProxyPort, gameId)
			}
		case packet := <-udpPacketChannel:
			route, err := proxy.GetRouteByAddr(gameState.routes, packet.SrcAddr)
			if err == nil {
				playerPongChannel <- PlayerPong{route.ProxyPort, time.Now()}
			}
			handleGameInfoPacket(gameState, gameStartChannel, proxyIp, packet)
		case conn := <-tcpRequestChannel:
			conn.Write([]byte(getTrackerText(hostname, gameState)))
			conn.Close()
		}
	}
}

func handleGameInfoPacket(gameState GameState, gameStartChannel chan GameStart, proxyIp net.IP, packet proxy.UdpPacket) {
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

	bolo.RewritePacketGameInfo(packet.Buffer, proxyIp)
	newGameInfo := bolo.ParsePacketGameInfo(packet.Buffer)

	gameInfo, ok := gameState.mapGameIdGameInfo[newGameInfo.GameId]
	if ok {
		newGameInfo.ServerStartTimestamp = gameInfo.ServerStartTimestamp
	} else {
		newGameInfo.ServerStartTimestamp = time.Now()
		bolo.PrintGameInfo(newGameInfo)
		fmt.Println()
	}
	gameState.mapGameIdGameInfo[newGameInfo.GameId] = newGameInfo

	route, err := proxy.GetRouteByAddr(gameState.routes, packet.SrcAddr)
	if err == nil {
		playerJoinGame(gameState, route.ProxyPort, newGameInfo.GameId)
	} else {
		gameStartChannel <- GameStart{packet.SrcAddr, newGameInfo.GameId}
	}
}

func pingGameInfo(connection *net.UDPConn, route proxy.Route, stopChannel chan struct{}) {
	gameInfoPingSeconds := config.GetValueInt("game_info_ping_seconds")
	ticker := time.NewTicker(time.Duration(gameInfoPingSeconds) * time.Second)

	for {
		select {
		case <-stopChannel:
			fmt.Println("Stopped pinging player", route.ProxyPort)
			ticker.Stop()
			return
		case <-ticker.C:
			buffer, err := hex.DecodeString("426f6c6f0099070d")
			if err != nil {
				break
			}
			connection.WriteToUDP(buffer, &route.PlayerIPAddr)
		}
	}
}

func pingTimeout(wg *sync.WaitGroup, shutdownChannel chan struct{}, playerPongChannel chan PlayerPong, playerPingTimeoutChannel chan int) {
	defer wg.Done()
	gameInfoPingDuration := time.Duration(config.GetValueInt("game_info_ping_seconds")) * time.Second
	gameInfoTimeoutDuration := gameInfoPingDuration + (5 * time.Second)
	mapPlayerPortPongTimestamp := make(map[int]time.Time)
	ticker := time.NewTicker(gameInfoPingDuration / 4)

	for {
		select {
		case <-shutdownChannel:
			ticker.Stop()
			return
		case pong := <-playerPongChannel:
			mapPlayerPortPongTimestamp[pong.ProxyPort] = pong.Timestamp
		case <-ticker.C:
			for playerPort, timestamp := range mapPlayerPortPongTimestamp {
				if time.Now().After(timestamp.Add(gameInfoTimeoutDuration)) {
					playerPingTimeoutChannel <- playerPort
					delete(mapPlayerPortPongTimestamp, playerPort)
				}
			}
		}
	}
}

func countGamePlayers(gameState GameState, gameId bolo.GameId) int {
	count := 0
	for _, activeGameId := range gameState.mapProxyPortGameId {
		if activeGameId == gameId {
			count = count + 1
		}
	}
	return count
}

func updatePlayerCount(gameState GameState, gameId bolo.GameId, playerCount int) {
	updatedGameInfo := gameState.mapGameIdGameInfo[gameId]
	updatedGameInfo.PlayerCount = uint16(playerCount)
	gameState.mapGameIdGameInfo[gameId] = updatedGameInfo
}

func playerJoinGame(gameState GameState, playerPort int, newGameId bolo.GameId) {
	oldGameId, oldGameIdOk := gameState.mapProxyPortGameId[playerPort]

	gameState.mapProxyPortGameId[playerPort] = newGameId
	playerCount := countGamePlayers(gameState, newGameId)
	updatePlayerCount(gameState, newGameId, playerCount)

	if oldGameIdOk {
		oldGamePlayerCount := countGamePlayers(gameState, oldGameId)
		if oldGamePlayerCount == 0 {
			delete(gameState.mapGameIdGameInfo, oldGameId)
		} else {
			updatePlayerCount(gameState, oldGameId, oldGamePlayerCount)
		}
	}
}

func initGameState() GameState {
	var gameState GameState
	gameState.mapGameIdGameInfo = make(map[bolo.GameId]bolo.GameInfo)
	gameState.mapProxyPortGameId = make(map[int]bolo.GameId)
	gameState.mapProxyPortPlayerName = make(map[int]string)
	gameState.mapProxyPortPingStopChannel = make(map[int]chan struct{})
	return gameState
}
