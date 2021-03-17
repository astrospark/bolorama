package stats

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/data"
	"git.astrospark.com/bolorama/state"
	"git.astrospark.com/bolorama/util"
)

const kLogIntervalSeconds = 60
const kElapsedMinutesPerLogInterval = 1

func Logger(context *state.ServerContext, db *sql.DB) {
	defer context.WaitGroup.Done()
	ticker := time.NewTicker(kLogIntervalSeconds * time.Second)

	for {
		select {
		case <-context.ShutdownChannel:
			fmt.Println("Stopped statistics")
			ticker.Stop()
			return
		case <-ticker.C:
			LogGames(context, db)
		case gameId := <-context.LogGameEndChannel:
			LogEndGame(db, gameId)
		case playerAddr := <-context.LogPlayerJoinChannel:
			LogPlayerJoin(db, net.ParseIP(playerAddr.IpAddr), playerAddr.IpPort)
		case playerAddr := <-context.LogPlayerLeaveChannel:
			LogPlayerLeave(db, net.ParseIP(playerAddr.IpAddr), playerAddr.IpPort)
		}
	}
}

func LogGames(context *state.ServerContext, db *sql.DB) {
	context.Mutex.Lock()

	games := make(map[string]data.DataGame)
	for gameId, game := range context.Games {
		hash := sha256.Sum256(gameId[:])
		strHash := hex.EncodeToString(hash[:])
		games[strHash] = data.DataGame{
			GameId:               strHash,
			MapName:              game.MapName,
			StartTimestamp:       strconv.FormatInt(game.ServerStartTimestamp.Unix(), 10),
			EndTimestamp:         sql.NullString{String: "", Valid: false},
			MaxPlayerCount:       0,
			ElapsedPlayerMinutes: 0,
		}
	}

	for _, player := range context.Players {
		hash := sha256.Sum256(player.GameId[:])
		strHash := hex.EncodeToString(hash[:])
		game := games[strHash]
		game.MaxPlayerCount = game.MaxPlayerCount + 1
		games[strHash] = game
	}

	context.Mutex.Unlock()

	var gameIds []string
	for gameId := range games {
		gameIds = append(gameIds, gameId)
	}
	dbGames := data.SelectGames(db, gameIds)

	var insertGames []data.DataGame
	var updateGames []data.DataGame
	for gameId, game := range games {
		found := false
		for _, dbGame := range dbGames {
			if dbGame.GameId == gameId {
				found = true
				if dbGame.ElapsedPlayerMinutes > 0 || game.MaxPlayerCount > 1 {
					game.ElapsedPlayerMinutes = dbGame.ElapsedPlayerMinutes + (game.MaxPlayerCount * kElapsedMinutesPerLogInterval)
				}
				game.MaxPlayerCount = util.MaxInt(game.MaxPlayerCount, dbGame.MaxPlayerCount)
				updateGames = append(updateGames, game)
				break
			}
		}
		if !found {
			insertGames = append(insertGames, game)
		}
	}

	for _, game := range insertGames {
		data.InsertGame(db, game)
	}

	for _, game := range updateGames {
		data.UpdateGame(db, game)
	}
}

func LogEndGame(db *sql.DB, gameId bolo.GameId) {
	hash := sha256.Sum256(gameId[:])
	data.EndGame(db, hex.EncodeToString(hash[:]))
}

func LogPlayerJoin(db *sql.DB, ipAddr net.IP, port int) {
	hash := hashPlayerId(ipAddr, port)
	data.InsertPlayerSession(db, hash)
}

func LogPlayerLeave(db *sql.DB, ipAddr net.IP, port int) {
	hash := hashPlayerId(ipAddr, port)
	data.EndPlayerSession(db, hash)
}

func hashPlayerId(ipAddr net.IP, port int) string {
	var playerId [6]byte
	copy(playerId[:], ipAddr.To4())
	binary.BigEndian.PutUint16(playerId[4:6], uint16(port))
	hash := sha256.Sum256(playerId[:])
	strHash := hex.EncodeToString(hash[:])
	return strHash
}
