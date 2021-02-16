package bolo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"git.astrospark.com/bolorama/proxy"
)

const PacketHeaderSize = 8
const boloSignature = "Bolo"

const PacketTypeOffset = 0x07
const PacketTypeGameState = 0x02
const PacketTypeGameStateAck = 0x04
const PacketType5 = 0x05
const PacketType6 = 0x06
const PacketType7 = 0x07
const PacketType8 = 0x08
const PacketType9 = 0x09
const PacketTypeGameInfo = 0x0e

const PacketType6PeerAddrOffset = 12
const PacketType6PeerPortOffset = 16
const PacketType7PeerAddrOffset = 12
const PacketType7PeerPortOffset = 16
const PacketType9PeerAddrOffset = 8
const PacketType9PeerPortOffset = 12

const MinesVisibleBitmask = 1 << 6

/* Macs count time since Midnight, 1st Jan 1904. Unix counts from 1970.
   This value adjusts for the 66 years and 17 leap-days difference. */
const seconds1904ToUnixEpoch = (((1970-1904)*365 + 17) * 24 * 60 * 60)

type GameInfo struct {
	GameID              [8]byte
	MapName             string
	HostAddr            net.UDPAddr
	StartTimestamp      uint32
	GameType            int
	AllowHiddenMines    bool
	AllowComputer       bool
	ComputerAdvantage   bool
	StartDelay          uint32
	TimeLimit           uint32
	PlayerCount         uint16
	NeutralPillboxCount uint16
	NeutralBaseCount    uint16
	HasPassword         bool
}

func verifyBoloSignature(msg []byte) bool {
	return string(msg[0:4]) == boloSignature
}

func verifyBoloVersion(msg []byte) bool {
	return bytes.Equal(msg[4:7], []byte{0x00, 0x99, 0x07})
}

func GetPacketType(msg []byte) int {
	return int(msg[7])
}

func ValidatePacket(packet proxy.UdpPacket) (bool, string) {
	if packet.Len < PacketHeaderSize {
		return false, fmt.Sprint("datagram too short (smaller than bolo header)")
	}

	if !verifyBoloSignature(packet.Buffer) {
		return false, fmt.Sprint("datagram failed bolo signature check")
	}

	if !verifyBoloVersion(packet.Buffer) {
		return false, fmt.Sprint("unsupported bolo version")
	}

	return true, ""
}

func ParsePacketGameInfo(msg []byte, hostAddr net.UDPAddr) GameInfo {
	var gameInfo GameInfo
	var pos int = PacketHeaderSize

	gameInfo.MapName = string(msg[pos+1 : msg[pos]+1])
	pos = pos + 36

	copy(gameInfo.GameID[:], msg[pos:pos+8])

	gameInfo.HostAddr = hostAddr
	pos = pos + 4

	gameInfo.StartTimestamp = binary.BigEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.GameType = int(msg[pos])
	pos = pos + 1

	gameInfo.AllowHiddenMines = !((msg[pos] & MinesVisibleBitmask) == MinesVisibleBitmask)
	pos = pos + 1

	gameInfo.AllowComputer = msg[pos] > 0
	pos = pos + 1

	gameInfo.ComputerAdvantage = msg[pos] > 0
	pos = pos + 1

	gameInfo.StartDelay = binary.LittleEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.TimeLimit = binary.LittleEndian.Uint32(msg[pos : pos+4])
	pos = pos + 4

	gameInfo.PlayerCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.NeutralPillboxCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.NeutralBaseCount = binary.LittleEndian.Uint16(msg[pos : pos+2])
	pos = pos + 2

	gameInfo.HasPassword = msg[pos] > 0
	pos = pos + 1

	return gameInfo
}

func PrintGameInfo(gameInfo GameInfo) {
	fmt.Println()
	fmt.Println("Game ID:", gameInfo.GameID)
	fmt.Println("Map Name:", gameInfo.MapName)
	fmt.Println("Host:", gameInfo.HostAddr.IP.String()+":"+strconv.Itoa(gameInfo.HostAddr.Port))
	fmt.Println("Start Timestamp:", time.Unix(int64(gameInfo.StartTimestamp-seconds1904ToUnixEpoch), 0))
	fmt.Println("Game Type:", gameInfo.GameType)
	fmt.Println("Allow Hidden Mines:", gameInfo.AllowHiddenMines)
	fmt.Println("Allow Computer:", gameInfo.AllowComputer)
	fmt.Println("Computer Advantage:", gameInfo.ComputerAdvantage)
	fmt.Println("Start Delay:", gameInfo.StartDelay)
	fmt.Println("Time Limit:", gameInfo.TimeLimit)
	fmt.Println("Player Count:", gameInfo.PlayerCount)
	fmt.Println("Neutral Pillbox Count:", gameInfo.NeutralPillboxCount)
	fmt.Println("Neutral Base Count:", gameInfo.NeutralBaseCount)
	fmt.Println("Password:", gameInfo.HasPassword)
}

func rewritePacketFixedPosition(packet proxy.UdpPacket, proxyPort int, proxyIP net.IP, offset int) {
	//fmt.Println()
	//fmt.Println(hex.Dump(packet.buffer))
	packetIP := packet.Buffer[offset : offset+4]
	if !bytes.Equal(packetIP, proxyIP) {
		port := make([]byte, 2)
		binary.BigEndian.PutUint16(port, uint16(proxyPort))
		packet.Buffer[offset+0] = proxyIP[0]
		packet.Buffer[offset+1] = proxyIP[1]
		packet.Buffer[offset+2] = proxyIP[2]
		packet.Buffer[offset+3] = proxyIP[3]
		packet.Buffer[offset+4] = port[0]
		packet.Buffer[offset+5] = port[1]
		//fmt.Println()
		//fmt.Println(hex.Dump(packet.buffer))
	}
}

func RewritePacket(packet proxy.UdpPacket, srcRoute proxy.Route, proxyIP net.IP) {
	// only the player who starts the game will send packets with the wrong ip address, and it will
	// be their own. so we can search for any ip that isn't ours, replace it with ours, and replace
	// the port with the player's assigned port

	switch packet.Buffer[PacketTypeOffset] {
	case PacketTypeGameInfo:
		// is it necessary to re-write this one? if it is used by the former upstream of the player
		// leaving to identify their new downstream, then as long as they are looking for themselves
		// and not the player leaving, then it shouldn't be necessary to change the "middle" ip.
		//
		// upstream -> middle -> downstream
		//     upstream recognizes themselves, changes their downstream
		//
		// upstream -> middle -> downstream
		//     upstream recognizes middle as their downstream, changes their downstream
	case PacketType6:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType6PeerAddrOffset)
	case PacketType7:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType7PeerAddrOffset)
	case PacketType9:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType9PeerAddrOffset)
	}
}
