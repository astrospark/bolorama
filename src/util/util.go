package util

import (
	"log"
	"net"
	"time"
)

type PlayerAddr struct {
	IpAddr    string
	IpPort    int
	ProxyPort int
}

type PlayerInfoEvent struct {
	PlayerAddr PlayerAddr
	SetId      bool
	SetName    bool
	PlayerId   int
	Name       string
}

// get preferred outbound ip of this machine
func GetOutboundIp() net.IP {
	conn, err := net.Dial("udp", "1.1.1.1:1")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func MaxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func MaxTime(a time.Time, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func ContainsString(strings []string, target string) bool {
	for _, element := range strings {
		if element == target {
			return true
		}
	}

	return false
}
