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

package tracker

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

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
