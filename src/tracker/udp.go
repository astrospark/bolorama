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
	"net"
	"strings"
	"sync"

	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/util"
)

func udpListener(wg *sync.WaitGroup, shutdownChannel chan struct{}, connection *net.UDPConn, port int, dataChannel chan proxy.UdpPacket) {
	defer wg.Done()

	buffer := make([]byte, util.MaxUdpPacketSize)

	go func() {
		for {
			_, ok := <-shutdownChannel
			if !ok {
				connection.Close()
				break
			}
		}
	}()

	fmt.Println("Listening on UDP port", port)

	for {
		n, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			if !strings.HasSuffix(err.Error(), "use of closed network connection") {
				fmt.Println(err)
			}
			fmt.Println("Stopped listening on UDP port", port)
			break
		}

		data := make([]byte, n)
		copy(data, buffer)
		dataChannel <- proxy.UdpPacket{
			SrcAddr: *addr,
			DstAddr: net.UDPAddr{},
			DstPort: port,
			Len:     n,
			Buffer:  data,
		}
	}
}
