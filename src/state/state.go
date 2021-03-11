package state

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/config"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/util"
)

type ServerContext struct {
	Players         []Player
	Games           map[bolo.GameId]bolo.GameInfo
	ProxyIpAddr     net.IP
	ProxyPort       int
	UdpConnection   *net.UDPConn
	RxChannel       chan proxy.UdpPacket
	ShutdownChannel chan struct{}
	WaitGroup       *sync.WaitGroup
	Mutex           *sync.RWMutex
	Debug           bool
}

type Player struct {
	IpAddr            net.IP
	IpPort            int
	ProxyPort         int
	Connection        *net.UDPConn
	TxChannel         chan proxy.UdpPacket
	DisconnectChannel chan struct{}
	GameId            bolo.GameId
	PlayerId          int
	Name              string
	Peers             map[int]time.Time
	PeerPackets       map[int]proxy.UdpPacket
	NatPort           int
}

func InitContext(port int) *ServerContext {
	debug := config.GetValueBool("debug")
	return &ServerContext{
		Games:           make(map[bolo.GameId]bolo.GameInfo),
		ProxyIpAddr:     util.GetOutboundIp(),
		ProxyPort:       port,
		UdpConnection:   connectUdp(port),
		RxChannel:       make(chan proxy.UdpPacket),
		ShutdownChannel: make(chan struct{}),
		WaitGroup:       &sync.WaitGroup{},
		Mutex:           &sync.RWMutex{},
		Debug:           debug,
	}
}

func connectUdp(port int) *net.UDPConn {
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprint(":", port))
	if err != nil {
		fmt.Println(err)
		return nil
	}

	connection, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return connection
}

func SprintServerState(context *ServerContext, newline string, lock bool) string {
	if lock {
		context.Mutex.RLock()
		defer context.Mutex.RUnlock()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("   Player                   Proxy Port    Game Id%s", newline))
	for _, player := range context.Players {
		ipAddr := fmt.Sprintf("%s:%d", player.IpAddr.String(), player.IpPort)
		sb.WriteString(fmt.Sprintf("   %-21s    %-10d    %s%s", ipAddr, player.ProxyPort, hex.EncodeToString(player.GameId[:]), newline))
	}
	return sb.String()
}

func PrintServerState(context *ServerContext, lock bool) {
	fmt.Print(SprintServerState(context, "\n", lock))
}

func gameCountPlayers(context *ServerContext, targetGameId bolo.GameId, lock bool) int {
	if lock {
		context.Mutex.RLock()
		defer context.Mutex.RUnlock()
	}

	count := 0
	for _, player := range context.Players {
		if player.GameId == targetGameId {
			count = count + 1
		}
	}
	return count
}

func GameUpdatePlayerCount(context *ServerContext, gameId bolo.GameId, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	playerCount := gameCountPlayers(context, gameId, false)
	if playerCount == 0 {
		GameDelete(context, gameId, false)
	} else {
		gameInfo := context.Games[gameId]
		gameInfo.PlayerCount = uint16(playerCount)
		context.Games[gameId] = gameInfo
	}
}

func GameDelete(context *ServerContext, gameId bolo.GameId, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	delete(context.Games, gameId)
}

func PlayerGetByAddr(context *ServerContext, addr net.UDPAddr, lock bool) (Player, error) {
	if lock {
		context.Mutex.RLock()
		defer context.Mutex.RUnlock()
	}

	for _, player := range context.Players {
		if net.IP.Equal(addr.IP, player.IpAddr) && addr.Port == player.IpPort {
			return player, nil
		}
	}

	return Player{}, fmt.Errorf("player with socket %s:%d not found",
		addr.IP.String(), addr.Port)
}

func PlayerGetByPort(context *ServerContext, port int, lock bool) (Player, error) {
	if lock {
		context.Mutex.RLock()
		defer context.Mutex.RUnlock()
	}

	for _, player := range context.Players {
		if port == player.ProxyPort {
			return player, nil
		}
	}

	return Player{}, fmt.Errorf("player with proxy port %d not found", port)
}

func PlayerNew(
	context *ServerContext,
	playerAddr net.UDPAddr,
	gameId bolo.GameId,
	natPort int,
	lock bool,
) Player {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	disconnectChannel := make(chan struct{})

	proxyPort, txChannel, connection := proxy.AddPlayer(
		context.WaitGroup,
		playerAddr,
		context.RxChannel,
		disconnectChannel,
		context.ShutdownChannel,
	)

	player := Player{
		IpAddr:            playerAddr.IP,
		IpPort:            playerAddr.Port,
		ProxyPort:         proxyPort,
		Connection:        connection,
		TxChannel:         txChannel,
		DisconnectChannel: disconnectChannel,
		GameId:            gameId,
		PlayerId:          -1,
		Name:              "<unknown>",
		Peers:             make(map[int]time.Time),
		PeerPackets:       make(map[int]proxy.UdpPacket),
		NatPort:           natPort,
	}

	context.Players = append(context.Players, player)

	return player
}

func PlayerJoinGame(context *ServerContext, playerPort int, newGameId bolo.GameId, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	var oldGameId bolo.GameId = bolo.GameId{}
	var oldGameIdOk bool = false
	for i, player := range context.Players {
		if player.ProxyPort == playerPort {
			oldGameId = player.GameId
			oldGameIdOk = true
			context.Players[i].GameId = newGameId
			context.Players[i].PlayerId = -1
		}
	}

	GameUpdatePlayerCount(context, newGameId, false)

	if oldGameIdOk && oldGameId != newGameId {
		GameUpdatePlayerCount(context, oldGameId, false)
	}
}

func playerRemoveElement(players []Player, idx int) []Player {
	players[idx] = players[len(players)-1]
	return players[:len(players)-1]
}

func PlayerDelete(context *ServerContext, playerAddr util.PlayerAddr, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	player_idx := -1
	for i, player := range context.Players {
		if net.IP.Equal(player.IpAddr, net.ParseIP(playerAddr.IpAddr)) && player.IpPort == playerAddr.IpPort && player.ProxyPort == playerAddr.ProxyPort {
			player_idx = i
			break
		}
	}

	if player_idx < 0 {
		return
	}

	gameId := context.Players[player_idx].GameId

	close(context.Players[player_idx].DisconnectChannel)
	proxy.DeletePort(context.Players[player_idx].ProxyPort)
	context.Players = playerRemoveElement(context.Players, player_idx)
	GameUpdatePlayerCount(context, gameId, false)
}

func PlayerSetNatPort(context *ServerContext, addr util.PlayerAddr, natPort int, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	playerIdx := -1
	for i, player := range context.Players {
		if (addr.IpAddr == player.IpAddr.String()) && (addr.IpPort == player.IpPort) && (addr.ProxyPort == player.ProxyPort) {
			playerIdx = i
			break
		}
	}

	if playerIdx >= 0 {
		context.Players[playerIdx].NatPort = natPort
	}
}

func PlayerSetId(context *ServerContext, addr util.PlayerAddr, playerId int, lock bool) {
	if lock {
		context.Mutex.Lock()
		defer context.Mutex.Unlock()
	}

	playerIdx := -1
	for i, player := range context.Players {
		if (addr.IpAddr == player.IpAddr.String()) && (addr.IpPort == player.IpPort) && (addr.ProxyPort == player.ProxyPort) {
			playerIdx = i
			break
		}
	}

	if playerIdx >= 0 {
		context.Players[playerIdx].PlayerId = playerId
	}
}

func PlayerSetName(context *ServerContext, addr util.PlayerAddr, playerId int, playerName string) {
	context.Mutex.Lock()
	defer context.Mutex.Unlock()

	gameId := [8]byte{}
	gameIdFound := false
	for _, player := range context.Players {
		if (addr.IpAddr == player.IpAddr.String()) && (addr.IpPort == player.IpPort) && (addr.ProxyPort == player.ProxyPort) {
			gameId = player.GameId
			gameIdFound = true
			break
		}
	}

	if !gameIdFound {
		return
	}

	for i, player := range context.Players {
		if (player.GameId == gameId) && (player.PlayerId == playerId) {
			if strings.HasSuffix(playerName, "Unknown Machine Name") {
				nameSlice := strings.Split(playerName, "@")
				playerName = strings.Join(nameSlice[0:len(nameSlice)-1], "")
			}
			context.Players[i].Name = playerName
			break
		}
	}
}
