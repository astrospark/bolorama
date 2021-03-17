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

package util

import (
	"log"
	"net"
	"time"
)

// The largest safe UDP packet length is 576 for IPv4 and 1280 for IPv6, where
// "safe" is defined as “guaranteed to be able to be reassembled, if fragmented."
const MaxUdpPacketSize = 1024

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
