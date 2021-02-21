package tracker

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.astrospark.com/bolorama/bolo"
)

var yesNo = map[bool]string{
	true:  "Yes",
	false: "No",
}

var gameTypeName = map[int]string{
	1: "Open Game",
	2: "Tournament",
	3: "Strict Tournament",
}

func getTrackerText(hostname string, gameState GameState) string {
	var sb strings.Builder

	sb.WriteString("= =================================================================== =\r")
	sb.WriteString("=                                                                     =\r")
	sb.WriteString("=                         Astrospark Bolorama                         =\r")
	sb.WriteString("=                                                                     =\r")
	sb.WriteString("= =================================================================== =\r")
	sb.WriteString("\r")

	var games []bolo.GameInfo
	for _, game := range gameState.mapGameIdGameInfo {
		games = append(games, game)
	}
	sort.Slice(games, func(i, j int) bool {
		return games[i].ServerStartTimestamp.After(games[j].ServerStartTimestamp)
	})

	if len(games) == 0 {
		sb.WriteString("   There are no games in progress.\r\r")
		return sb.String()
	}

	for _, game := range games {
		ports := getGamePorts(gameState, game.GameId)
		players := getPlayers(gameState, ports)
		sort.Ints(ports)
		sb.WriteString(getGameInfoText(hostname, ports[0], game, players))
		sb.WriteString("\r")
	}

	if len(games) == 1 {
		sb.WriteString("   There is 1 game in progress.\r\r")
	} else {
		sb.WriteString(fmt.Sprintf("   There are %d games in progress.\r\r", len(games)))
	}

	return sb.String()
}

func getGameInfoText(hostname string, hostport int, gameInfo bolo.GameInfo, players []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Host: %s", hostname))
	// sb.WriteString(fmt.Sprintf("\r {%d}", port))
	sb.WriteString(fmt.Sprintf("  Players: %d", gameInfo.PlayerCount))
	sb.WriteString(fmt.Sprintf("  Bases: %d", gameInfo.NeutralBaseCount))
	sb.WriteString(fmt.Sprintf("  Pills: %d\r", gameInfo.NeutralPillboxCount))

	sb.WriteString(fmt.Sprintf("Map: %s", gameInfo.MapName))
	sb.WriteString(fmt.Sprintf("  Game: %s", gameTypeName[gameInfo.GameType]))
	sb.WriteString(fmt.Sprintf("  Mines: %s", yesNo[gameInfo.AllowHiddenMines]))
	sb.WriteString(fmt.Sprintf("  Bots: %s", yesNo[gameInfo.AllowComputer]))
	sb.WriteString(fmt.Sprintf("  PW: %s\r", yesNo[gameInfo.HasPassword]))

	sb.WriteString(fmt.Sprintf("Version: 0.99.7"))
	sb.WriteString(fmt.Sprintf("  Tracked-For: %d minutes", gameDuration(gameInfo)))
	sb.WriteString(fmt.Sprintf("  Player-List:\r"))

	playersText := strings.Join(players, ", ")
	sb.WriteString(fmt.Sprintf("   %s\r", playersText))

	// sb.WriteString(fmt.Sprintf("   -\r"))

	return sb.String()
}

func getGamePorts(gameState GameState, targetGameId bolo.GameId) []int {
	var ports []int
	for port, gameId := range gameState.mapProxyPortGameId {
		if gameId == targetGameId {
			ports = append(ports, port)
		}
	}
	return ports
}

func getPlayers(gameState GameState, ports []int) []string {
	var players []string
	for _, port := range ports {
		name, ok := gameState.mapProxyPortPlayerName[port]
		if !ok {
			name = "<<unknown>>"
		}
		players = append(players, name)
	}
	return players
}

func gameDuration(gameInfo bolo.GameInfo) int {
	duration := time.Now().Sub(gameInfo.ServerStartTimestamp)
	return int(duration.Minutes())
}
