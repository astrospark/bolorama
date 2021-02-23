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
