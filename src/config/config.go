package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

const configFilename = "config.txt"

var configMap map[string]string = nil

var valid []string = []string{"hostname", "tracker_port"}

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

func load() {
	if configMap != nil {
		return
	}

	configMap = make(map[string]string)

	file, err := os.Open(configFilename)
	if err != nil {
		log.Fatalln("Failed to open config file:", configFilename)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		s := strings.SplitN(line, "=", 2)
		if len(s) < 2 {
			log.Fatalln("Malformed config:", line)
		}
		configMap[s[0]] = s[1]
	}

	file.Close()
}
