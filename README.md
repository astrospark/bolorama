# Bolorama

Bolorama enables the classic Mac tank game Bolo to work in a post-NATpocalypse world.

## Build

```
cd src
CGO_ENABLED=1 go build ./cmd/bolorama
```

## Config

The config file is named `config.txt` in the current working directory. The file format is one setting per line, in the form `name=value`. At a minimum, the config file must include the `hostname` setting:

```
hostname=bolo.astrospark.com
```

### Settings

#### database_filename

The name of the database file, if statistics logging is enabled. Type: string. Default: `db.sqlite`

#### debug

Whether to enable debug logging. Type: boolean. Default: `false`

#### enable_statistics

Whether to enable statistics logging. Type: boolean. Default: `false`

#### game_info_ping_seconds

Period for pinging a player for game info. Can affect NAT traversal if too long. Type: integer. Default: `20`

#### hostname

This is the hostname that will appear in the tracker game info for players to connect to. Type: string. No default.

#### player_timeout_seconds

Period for disconnecting a player for network inactivity (not game inactivity). Type: integer. Default: `60`

#### tracker_port

Port number for the tracker to listen on. Type: integer. Default: `50000`
