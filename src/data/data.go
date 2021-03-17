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

package data

import (
	"database/sql"
	"log"
	"runtime/debug"
	"strings"

	"git.astrospark.com/bolorama/config"
	_ "github.com/mattn/go-sqlite3"
)

const kDataSchemaVersion = 1

type DataGame struct {
	GameId               string
	MapName              string
	StartTimestamp       string
	EndTimestamp         sql.NullString
	MaxPlayerCount       int
	ElapsedPlayerMinutes int
}

func Init() *sql.DB {
	db_filename := config.GetValueString("database_filename")
	db, err := sql.Open("sqlite3", db_filename+"?Mode=rwc")
	if err != nil {
		debug.PrintStack()
		log.Fatalf("failed to open/create database (%s): %s\n", db_filename, err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = 'config' and type = 'table'").Scan(&count)
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}

	if count == 0 {
		InitTables(db)
	}

	return db
}

func InitTables(db *sql.DB) {
	_, err := db.Exec(
		"CREATE TABLE game (" +
			"id TEXT PRIMARY KEY, " +
			"map_name TEXT NOT NULL, " +
			"started_at TEXT NOT NULL, " +
			"ended_at TEXT, " +
			"max_player_count INTEGER NOT NULL, " +
			"elapsed_player_minutes INTEGER NOT NULL" +
			")",
	)
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}

	_, err = db.Exec(
		"CREATE TABLE player_session (" +
			"id INTEGER PRIMARY KEY AUTOINCREMENT, " +
			"player_id TEXT NOT NULL, " +
			"joined_at TEXT NOT NULL, " +
			"left_at TEXT" +
			")",
	)
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}

	_, err = db.Exec("CREATE TABLE config (name TEXT PRIMARY KEY, value TEXT)")
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}

	statement, err := db.Prepare("INSERT INTO config (name, value) VALUES ($1, $2)")
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}
	defer statement.Close()

	_, err = statement.Exec("schema_version", kDataSchemaVersion)
	if err != nil {
		debug.PrintStack()
		log.Fatalln("sqlite error", err)
	}
}

func SelectGames(db *sql.DB, gameIds []string) []DataGame {
	var games []DataGame

	if len(gameIds) == 0 {
		return games
	}

	args := make([]interface{}, len(gameIds))
	for i, gameId := range gameIds {
		args[i] = gameId
	}

	sql :=
		"SELECT " +
			"id, " +
			"map_name, " +
			"started_at, " +
			"ended_at, " +
			"max_player_count, " +
			"elapsed_player_minutes " +
			"FROM game " +
			"WHERE id in (?" + strings.Repeat(",?", len(args)-1) + ")"

	rows, err := db.Query(sql, args...)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return games
	}
	defer rows.Close()

	for rows.Next() {
		var game DataGame
		err = rows.Scan(
			&game.GameId,
			&game.MapName,
			&game.StartTimestamp,
			&game.EndTimestamp,
			&game.MaxPlayerCount,
			&game.ElapsedPlayerMinutes,
		)
		if err != nil {
			debug.PrintStack()
			log.Println("sqlite error", err)
			return games
		}
		games = append(games, game)
	}

	err = rows.Err()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
	}

	return games
}

func InsertGame(db *sql.DB, game DataGame) {
	result, err := db.Exec(
		"INSERT INTO game "+
			"(id, map_name, started_at, max_player_count, elapsed_player_minutes) "+
			"VALUES ($1, $2, datetime($3, 'unixepoch'), $4, $5)",
		game.GameId,
		game.MapName,
		game.StartTimestamp,
		game.MaxPlayerCount,
		game.ElapsedPlayerMinutes,
	)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	if rowCount != 1 {
		debug.PrintStack()
		log.Println("sql insert game failed")
	}
}

func UpdateGame(db *sql.DB, game DataGame) {
	result, err := db.Exec(
		"UPDATE game "+
			"SET "+
			"max_player_count = $1, "+
			"elapsed_player_minutes = $2 "+
			"WHERE id = $3",
		game.MaxPlayerCount,
		game.ElapsedPlayerMinutes,
		game.GameId,
	)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	if rowCount != 1 {
		debug.PrintStack()
		log.Println("sql update game failed")
	}
}

func EndGame(db *sql.DB, gameId string) {
	result, err := db.Exec(
		"UPDATE game "+
			"SET "+
			"ended_at = datetime('now') "+
			"WHERE id = $2",
		gameId,
	)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	if rowCount != 1 {
		debug.PrintStack()
		log.Println("sql update game failed")
	}
}

func InsertPlayerSession(db *sql.DB, playerId string) {
	result, err := db.Exec(
		"INSERT INTO player_session "+
			"(player_id, joined_at) "+
			"VALUES ($1, datetime('now'))",
		playerId,
	)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	if rowCount != 1 {
		debug.PrintStack()
		log.Println("sql insert player session failed")
	}
}

func EndPlayerSession(db *sql.DB, playerId string) {
	sql := "UPDATE player_session " +
		"SET " +
		"left_at = datetime('now') " +
		"WHERE id in (SELECT max(id) FROM player_session WHERE player_id = $1) " +
		"AND left_at IS NULL"

	result, err := db.Exec(sql, playerId)
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	rowCount, err := result.RowsAffected()
	if err != nil {
		debug.PrintStack()
		log.Println("sqlite error", err)
		return
	}
	if rowCount != 1 {
		debug.PrintStack()
		log.Println("sql end player session failed")
	}
}
