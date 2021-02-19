package tracker

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"git.astrospark.com/bolorama/bolo"
)

func Tracker(wg *sync.WaitGroup, shutdownChannel chan struct{}, proxyIP net.IP, port int, gameInfoChannel chan bolo.GameInfo) {
	tcpRequestChannel := make(chan net.Conn)
	defer wg.Done()

	wg.Add(1)
	go tcpListener(wg, shutdownChannel, port, tcpRequestChannel)

	var gameInfo bolo.GameInfo
	for {
		select {
		case newGameInfo := <-gameInfoChannel:
			gameInfo = newGameInfo
		case conn := <-tcpRequestChannel:
			conn.Write([]byte(getTrackerText(proxyIP, gameInfo)))
			conn.Close()
		case _, ok := <-shutdownChannel:
			if !ok {
				return
			}
		}
	}
}

func tcpListener(wg *sync.WaitGroup, shutdownChannel chan struct{}, port int, tcpRequestChannel chan net.Conn) {
	defer wg.Done()

	listenAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprint(":", port))
	if err != nil {
		log.Fatalln(err)
	}

	connection, err := net.ListenTCP("tcp4", listenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for {
			_, ok := <-shutdownChannel
			if !ok {
				connection.Close()
				break
			}
		}
	}()

	fmt.Println("Listening on TCP port", port)

	for {
		conn, err := connection.Accept()
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on TCP port", port)
			break
		}

		tcpRequestChannel <- conn
	}
}
