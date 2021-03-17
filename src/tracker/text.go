/*
	Copyright 2021 Astrospark Technologies

	This file is part of bolorama. Bolorama is free software: you can
	redistribute it and/or modify it under the terms of the GNU Affero General
	Public License as published by the Free Software Foundation, either version
	3 of the License, or (at your option) any later version.

	Bolorama is distributed in the hope that it will be useful, but WITHOUT ANY
	WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
	FOR A PARTICULAR PURPOSE. See the GNU General Public License for more
	details.

	You should have received a copy of the GNU Affero General Public License
	along with Bolorama. If not, see <https://www.gnu.org/licenses/>.
*/

package tracker

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/state"
)

var yesNo = map[bool]string{
	true:  "Yes",
	false: "No",
}

var minesHiddenVisible = map[bool]string{
	true:  "Hidden",
	false: "Visible",
}

var gameTypeName = map[int]string{
	1: "Open Game",
	2: "Tournament",
	3: "Strict Tournament",
}

func getTrackerText(context *state.ServerContext, hostname string) string {
	context.Mutex.RLock()
	defer context.Mutex.RUnlock()

	var sb strings.Builder

	sb.WriteString("= =================================================================== =\r")
	sb.WriteString("=                         Astrospark Bolorama                         =\r")
	sb.WriteString("=                                                                     =\r")
	sb.WriteString("=                      http://bolo.astrospark.com                     =\r")
	sb.WriteString("= =================================================================== =\r")
	sb.WriteString("\r")

	var games []bolo.GameInfo
	for _, game := range context.Games {
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
		ports := getGamePlayerPorts(context, game.GameId)
		players := getGamePlayerNames(context, game.GameId)
		sort.Ints(ports)
		sb.WriteString(getGameInfoText(hostname, ports[0], game, players))
		sb.WriteString("\r")
	}

	if len(games) == 1 {
		sb.WriteString("   There is 1 game in progress.\r\r")
	} else {
		sb.WriteString(fmt.Sprintf("   There are %d games in progress.\r\r", len(games)))
	}

	sb.WriteString(state.SprintServerState(context, "\r", false))

	return sb.String()
}

func getGameInfoText(hostname string, hostport int, gameInfo bolo.GameInfo, players []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Host: %s {%d}", hostname, hostport))
	sb.WriteString(fmt.Sprintf("  Players: %d", gameInfo.PlayerCount))
	sb.WriteString(fmt.Sprintf("  Bases: %d", gameInfo.NeutralBaseCount))
	sb.WriteString(fmt.Sprintf("  Pills: %d\r", gameInfo.NeutralPillboxCount))

	sb.WriteString(fmt.Sprintf("Map: %s", gameInfo.MapName))
	sb.WriteString(fmt.Sprintf("  Game: %s", gameTypeName[gameInfo.GameType]))
	sb.WriteString(fmt.Sprintf("  Mines: %s", minesHiddenVisible[gameInfo.AllowHiddenMines]))
	sb.WriteString(fmt.Sprintf("  Bots: %s", yesNo[gameInfo.AllowComputer]))
	sb.WriteString(fmt.Sprintf("  PW: %s\r", yesNo[gameInfo.HasPassword]))

	sb.WriteString("Version: 0.99.8")
	sb.WriteString(fmt.Sprintf("  Tracked-For: %d minutes", gameDuration(gameInfo)))
	sb.WriteString("  Player-List:\r")

	startIdx := 0
	lineLength := 0
	for i := range players {
		playerLength := len(players[i])
		if lineLength+playerLength+2 > 80 {
			sb.WriteString(fmt.Sprintf("   %s", strings.Join(players[startIdx:i], ", ")))
			if i < len(players) {
				sb.WriteString(", ")
			}
			sb.WriteString("\r")
			startIdx = i
			lineLength = 0
		} else {
			lineLength = lineLength + playerLength + 2
		}
	}
	sb.WriteString(fmt.Sprintf("   %s\r", strings.Join(players[startIdx:], ", ")))

	return sb.String()
}

func getGamePlayerPorts(context *state.ServerContext, targetGameId bolo.GameId) []int {
	var ports []int
	for _, player := range context.Players {
		if player.GameId == targetGameId {
			ports = append(ports, player.ProxyPort)
		}
	}
	return ports
}

func getGamePlayerNames(context *state.ServerContext, targetGameId bolo.GameId) []string {
	var playerNames []string
	for _, player := range context.Players {
		if player.GameId == targetGameId {
			playerNames = append(playerNames, player.Name)
		}
	}
	return playerNames
}

func gameDuration(gameInfo bolo.GameInfo) int {
	duration := time.Since(gameInfo.ServerStartTimestamp)
	return int(duration.Minutes())
}
