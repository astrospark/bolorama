package util

import (
	"log"
	"net"
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
