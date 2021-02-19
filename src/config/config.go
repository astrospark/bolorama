package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

const configFilename = "config.txt"

var configMap map[string]string = nil

var valid []string = []string{"hostname"}

func GetValue(name string) string {
	if configMap == nil {
		configMap = make(map[string]string)
		load()
	}

	return configMap[name]
}

func load() {
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
