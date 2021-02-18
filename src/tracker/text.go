package tracker

import (
	"fmt"
	"net"
	"strings"

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

func getTrackerText(proxyIP net.IP, gameInfo bolo.GameInfo) string {
	var sb strings.Builder

	sb.WriteString("= =================================================================== =\r")
	sb.WriteString("=                                                                     =\r")
	sb.WriteString("=                         Astrospark Bolorama                         =\r")
	sb.WriteString("=                                                                     =\r")
	sb.WriteString("= =================================================================== =\r")

	if gameInfo != (bolo.GameInfo{}) {
		sb.WriteString(getGameInfoText(proxyIP, gameInfo))

		sb.WriteString(fmt.Sprintf("\r\r   There is 1 game in progress.\r"))
	} else {
		sb.WriteString(fmt.Sprintf("\r\r   There are no games in progress.\r"))
	}

	return sb.String()
}

func getGameInfoText(proxyIP net.IP, gameInfo bolo.GameInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\rHost: %s", proxyIP))
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
	// sb.WriteString(fmt.Sprintf("  Tracked-For: - min."))
	// sb.WriteString(fmt.Sprintf("  Player-List:\r"))

	// sb.WriteString(fmt.Sprintf("   -\r"))

	return sb.String()
}
