package bolo

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"git.astrospark.com/bolorama/proxy"
	"github.com/snksoft/crc"
)

const PacketHeaderSize = 8
const boloSignature = "Bolo"

const PacketTypeOffset = 0x07
const PacketType0 = 0x00
const PacketType1 = 0x01
const PacketTypeGameState = 0x02
const PacketTypeGameStateAck = 0x04
const PacketType5 = 0x05
const PacketType6 = 0x06
const PacketType7 = 0x07
const PacketType8 = 0x08
const PacketType9 = 0x09
const PacketTypeGameInfo = 0x0e

const PacketType0PeerAddrOffset = 0
const PacketType0PeerPortOffset = 4
const PacketType1PeerAddrOffset = 0
const PacketType1PeerPortOffset = 4
const PacketType6PeerAddrOffset = 12
const PacketType6PeerPortOffset = 16
const PacketType7PeerAddrOffset = 12
const PacketType7PeerPortOffset = 16
const PacketType9PeerAddrOffset = 8
const PacketType9PeerPortOffset = 12

const MinesVisibleBitmask = 1 << 6

const OpcodeMapData = 0xf1
const OpcodePlayerInfo = 0xff
const OpcodeMapDataSubcode01 = 0x01

/* Macs count time since Midnight, 1st Jan 1904. Unix counts from 1970.
   This value adjusts for the 66 years and 17 leap-days difference. */
const seconds1904ToUnixEpoch = (((1970-1904)*365 + 17) * 24 * 60 * 60)

type GameInfo struct {
	GameID              [8]byte
	MapName             string
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

func RewritePacketGameInfo(buffer []byte, ip net.IP) {
	var pos int = PacketHeaderSize
	pos = pos + 36 // skip map name
	buffer[pos+0] = ip[0]
	buffer[pos+1] = ip[1]
	buffer[pos+2] = ip[2]
	buffer[pos+3] = ip[3]
}

func ParsePacketGameInfo(msg []byte) GameInfo {
	var gameInfo GameInfo
	var pos int = PacketHeaderSize

	gameInfo.MapName = string(msg[pos+1 : pos+1+int(msg[pos])])
	pos = pos + 36

	copy(gameInfo.GameID[:], msg[pos:pos+8])

	// skip host ip
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

func rewriteOpcodePlayerInfo(pos int, buffer []byte, proxyPort int, proxyIP net.IP) {
	// skip unknown byte, address length, first address
	pos = pos + 8

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(hex.Dump(buffer))
			fmt.Println("panic occurred:", err)
			debug.PrintStack()
		}
	}()

	buffer[pos+0] = proxyIP[0]
	buffer[pos+1] = proxyIP[1]
	buffer[pos+2] = proxyIP[2]
	buffer[pos+3] = proxyIP[3]
	binary.BigEndian.PutUint16(buffer[pos+4:pos+6], uint16(proxyPort))
}

func rewriteOpcodeGameInfo(pos int, buffer []byte, proxyPort int, proxyIP net.IP) {
	//mapName := buffer[pos+1 : pos+1+int(buffer[pos])]
	pos = pos + 36

	//fmt.Println("OpcodeMapData Map Name:", string(mapName))

	buffer[pos+0] = proxyIP[0]
	buffer[pos+1] = proxyIP[1]
	buffer[pos+2] = proxyIP[2]
	buffer[pos+3] = proxyIP[3]
}

func rewriteGameStateBlock(startPos int, buffer []byte, proxyPort int, proxyIP net.IP) int {
	blockLength := int(buffer[startPos]&0x7f) + 1
	pos := startPos + 1

	nextBlockPos := pos + blockLength
	endPos := nextBlockPos - 1

	if blockLength < 4 {
		return nextBlockPos
	}

	pos = pos + 3 // skip sequence, sender + flags, unknown byte

	// sometimes we overrun the buffer here
	if pos < len(buffer) {
		opcode := int(buffer[pos])
		pos = pos + 1

		// TODO: it's possible for blocks to contain multiple opcodes

		rewriteCrc := false
		switch opcode {
		case OpcodeMapData:
			subcode := int(buffer[pos])
			pos = pos + 1

			if subcode == OpcodeMapDataSubcode01 {
				rewriteOpcodeGameInfo(pos, buffer, proxyPort, proxyIP)
				rewriteCrc = true
			}
		case OpcodePlayerInfo:
			if buffer[pos] == 0xf0 && blockLength == 26 {
				rewriteOpcodePlayerInfo(pos, buffer, proxyPort, proxyIP)
				rewriteCrc = true
			}
		}

		if rewriteCrc {
			crc64 := crc.CalculateCRC(crc.XMODEM, buffer[startPos:nextBlockPos-2])
			binary.BigEndian.PutUint16(buffer[endPos-1:nextBlockPos], uint16(crc64))
		}
	} else {
		fmt.Printf("Warning: buffer overrun (pos = %d, len(buffer) = %d\n", pos, len(buffer))
		fmt.Println(hex.Dump(buffer))
	}

	return nextBlockPos
}

func rewritePacketGameState(buffer []byte, proxyPort int, proxyIP net.IP) {
	pos := PacketHeaderSize
	pos = pos + 1 // skip state sequence

	for pos < len(buffer) {
		pos = rewriteGameStateBlock(pos, buffer, proxyPort, proxyIP)
	}
}

func rewritePacketFixedPosition(packet proxy.UdpPacket, proxyPort int, proxyIP net.IP, offset int) {
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
	}
}

func RewritePacket(packet proxy.UdpPacket, srcRoute proxy.Route, proxyIP net.IP) {
	// only the player who starts the game will send packets with the wrong ip address, and it will
	// be their own. so we can search for any ip that isn't ours, replace it with ours, and replace
	// the port with the player's assigned port

	switch packet.Buffer[PacketTypeOffset] {
	case PacketType0:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType0PeerAddrOffset)
	case PacketType1:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType1PeerAddrOffset)
	case PacketTypeGameState:
		rewritePacketGameState(packet.Buffer, srcRoute.ProxyPort, proxyIP)
	case PacketType6:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType6PeerAddrOffset)
	case PacketType7:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType7PeerAddrOffset)
	case PacketType9:
		rewritePacketFixedPosition(packet, srcRoute.ProxyPort, proxyIP, PacketType9PeerAddrOffset)
	}
}
