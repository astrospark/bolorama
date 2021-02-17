package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"git.astrospark.com/bolorama/bolo"
	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/tracker"
)

const trackerPort = 50000

// get preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "1.1.1.1:1")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func printRouteTable(gameIDRouteTableMap map[[8]byte][]proxy.Route) {
	fmt.Println()
	fmt.Println("Route Table")
	for gameID, routeTable := range gameIDRouteTableMap {
		fmt.Println(" ", gameID)
		for _, route := range routeTable {
			fmt.Println("   ", route.PlayerIPAddr, route.ProxyPort)
		}
	}
}

func main() {
	const boloPort = 50000
	gameIDRouteTableMap := make(map[[8]byte][]proxy.Route)
	rxChannel := make(chan proxy.UdpPacket)
	trackerControlChannel := make(chan int)
	proxyIP := getOutboundIP()

	go tracker.UdpListener(trackerPort, rxChannel, trackerControlChannel)

	for {
		data := <-rxChannel

		valid, _ := bolo.ValidatePacket(data)
		if !valid {
			// skip non-bolo packets
			continue
		}

		switch bolo.GetPacketType(data.Buffer) {
		case bolo.PacketTypeGameInfo:
			if data.DstPort != trackerPort {
				// ignore tracker packets except on tracker port
				break
			}

			bolo.RewritePacketGameInfo(data.Buffer, proxyIP)
			gameInfo := bolo.ParsePacketGameInfo(data.Buffer)
			bolo.PrintGameInfo(gameInfo)

			_, ok := gameIDRouteTableMap[gameInfo.GameID]
			if !ok {
				rewrite := true
				playerRoute := proxy.AddPlayer(data.SrcAddr, rxChannel, rewrite)
				gameIDRouteTableMap[gameInfo.GameID] = []proxy.Route{playerRoute}
			}

			printRouteTable(gameIDRouteTableMap)

		default:
			if data.DstPort == trackerPort {
				// drop non-tracker packets received on tracker port
				break
			}

			// get destination player ip by proxy port
			gameID, dstRoute, err := proxy.GetRouteByPort(gameIDRouteTableMap, data.DstPort)
			if err != nil {
				// shouldn't be able to receive data on a port that isn't mapped
				fmt.Println(err)
				continue
			}

			// get proxy port by source player ip
			srcRoute, err := proxy.GetRouteByAddr(gameIDRouteTableMap, data.SrcAddr)
			if err != nil {
				rewrite := false
				srcRoute = proxy.AddPlayer(data.SrcAddr, rxChannel, rewrite)
				gameIDRouteTableMap[gameID] = append(gameIDRouteTableMap[gameID], srcRoute)

				printRouteTable(gameIDRouteTableMap)
			}

			if srcRoute.Rewrite {
				bolo.RewritePacket(data, srcRoute, proxyIP)
			}

			if bytes.Contains(data.Buffer, []byte{0xC0, 0xA8, 0x00, 0x50}) {
				fmt.Println()
				fmt.Println("Warning: Outgoing packet matches 192.168.0.80")
				fmt.Printf("Src: %s:%d Dst: %s:%d\n",
					srcRoute.PlayerIPAddr.IP.String(), srcRoute.PlayerIPAddr.Port,
					dstRoute.PlayerIPAddr.IP.String(), dstRoute.PlayerIPAddr.Port)
				fmt.Println(hex.Dump(data.Buffer))
			}

			data.DstAddr = dstRoute.PlayerIPAddr
			srcRoute.TxChannel <- data
		}
	}
}
