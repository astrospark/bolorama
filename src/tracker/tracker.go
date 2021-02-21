package tracker

import (
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

type GameState struct {
	routes                 []proxy.Route
	mapGameIdGameInfo      map[bolo.GameId]bolo.GameInfo
	mapProxyPortGameId     map[int]bolo.GameId
	mapProxyPortPlayerName map[int]string
}

func Tracker(
	wg *sync.WaitGroup,
	shutdownChannel chan struct{},
	gameStartChannel chan GameStart,
	newRouteChannel chan NewRoute,
	joinGameChannel chan JoinGame,
	leaveGameChannel chan proxy.Route,
) {
	defer wg.Done()
	udpPacketChannel := make(chan proxy.UdpPacket)
	tcpRequestChannel := make(chan net.Conn)
	hostname := config.GetValueString("hostname")
	port := config.GetValueInt("tracker_port")
	proxyIp := util.GetOutboundIp()
	gameState := initGameState()

	wg.Add(2)
	go udpListener(wg, shutdownChannel, port, udpPacketChannel)
	go tcpListener(wg, shutdownChannel, port, tcpRequestChannel)

	for {
		select {
		case _, ok := <-shutdownChannel:
			if !ok {
				return
			}
		case leaveGameRoute := <-leaveGameChannel:
			gameId := gameState.mapProxyPortGameId[leaveGameRoute.ProxyPort]
			delete(gameState.mapProxyPortGameId, leaveGameRoute.ProxyPort)
			delete(gameState.mapProxyPortPlayerName, leaveGameRoute.ProxyPort)
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
		case joinGame := <-joinGameChannel:
			gameId, ok := gameState.mapProxyPortGameId[joinGame.DstProxyPort]
			if ok {
				playerJoinGame(gameState, joinGame.SrcProxyPort, gameId)
			}
		case packet := <-udpPacketChannel:
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
	}
	gameState.mapGameIdGameInfo[newGameInfo.GameId] = newGameInfo

	route, err := proxy.GetRouteByAddr(gameState.routes, packet.SrcAddr)
	if err == nil {
		playerJoinGame(gameState, route.ProxyPort, newGameInfo.GameId)
	} else {
		gameStartChannel <- GameStart{packet.SrcAddr, newGameInfo.GameId}
	}

	bolo.PrintGameInfo(newGameInfo)
	fmt.Println()
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
	return gameState
}
