package tracker

import (
	"fmt"
	"net"
	"strings"

	"git.astrospark.com/bolorama/bolo"
)

func Tracker(proxyIP net.IP, port int, gameInfoChannel chan bolo.GameInfo, controlChannel chan int) {
	tcpRequestChannel := make(chan net.Conn)

	listenAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprint(":", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := net.ListenTCP("tcp4", listenAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()

	go func() {
		for {
			stopPort := <-controlChannel
			if stopPort == port {
				connection.Close()
			}
		}
	}()

	fmt.Println("Listening on TCP port", port)

	go tcpListener(connection, tcpRequestChannel)

	var gameInfo bolo.GameInfo
	for {
		select {
		case newGameInfo := <-gameInfoChannel:
			gameInfo = newGameInfo
		case conn := <-tcpRequestChannel:
			conn.Write([]byte(getTrackerText(proxyIP, gameInfo)))
			conn.Close()
		}
	}
}

func tcpListener(connection *net.TCPListener, tcpRequestChannel chan net.Conn) {
	for {
		conn, err := connection.Accept()
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on TCP port")
			break
		}

		tcpRequestChannel <- conn
	}
}
