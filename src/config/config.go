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

package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"git.astrospark.com/bolorama/util"
)

const configFilename = "config.txt"

var configMap map[string]string = nil

var valid []string = []string{
	"database_filename",
	"debug",
	"enable_statistics",
	"hostname",
	"game_info_ping_seconds",
	"player_timeout_seconds",
	"tracker_debug_port",
	"tracker_port",
}

var defaults = map[string]string{
	"database_filename":      "db.sqlite",
	"debug":                  "false",
	"enable_statistics":      "false",
	"game_info_ping_seconds": "20",
	"player_timeout_seconds": "60",
	"tracker_debug_port":     "50001",
	"tracker_port":           "50000",
}

var mapBoolValue = map[string]bool{
	"true":  true,
	"false": false,
}

func GetValueString(name string) string {
	load()
	value, ok := configMap[name]
	if !ok {
		log.Fatalln("Config property is not present:", name)
	}
	return value
}

func GetValueInt(name string) int {
	load()
	valueString := GetValueString(name)
	value, err := strconv.Atoi(valueString)
	if err != nil {
		log.Fatalln("Config property is not an integer:", name)
	}
	return value
}

func GetValueBool(name string) bool {
	load()
	valueString := strings.ToLower(GetValueString(name))
	valueBool, ok := mapBoolValue[valueString]
	if !ok {
		log.Fatalln("Config property is not a boolean:", name)
	}
	return valueBool
}

func load() {
	if configMap != nil {
		return
	}

	configMap = make(map[string]string)
	for key, value := range defaults {
		configMap[key] = value
	}

	file, err := os.Open(configFilename)
	if err != nil {
		log.Fatalln("Failed to open config file:", configFilename)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			// skip blank lines
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) < 2 {
			log.Fatalln("Malformed config:", line)
		}
		if util.ContainsString(valid, s[0]) {
			configMap[s[0]] = s[1]
		}
	}

	file.Close()
}
