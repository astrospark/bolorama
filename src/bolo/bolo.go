package bolo

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
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

const PacketType0PeerAddrOffset = 8
const PacketType0PeerPortOffset = 12
const PacketType1PeerAddrOffset = 8
const PacketType1PeerPortOffset = 12
const PacketType6PeerAddrOffset = 12
const PacketType6PeerPortOffset = 16
const PacketType7PeerAddrOffset = 12
const PacketType7PeerPortOffset = 16
const PacketType9PeerAddrOffset = 8
const PacketType9PeerPortOffset = 12

const MinesVisibleBitmask = 1 << 6

const OpcodeGameInfo = 0x11
const OpcodeMapData = 0x13
const OpcodePlayerName = 0x18
const OpcodeSendMessage = 0x1a
const OpcodeDisconnect = 0x30
const OpcodeGameInfoSubcodeGame = 0x01
const OpcodeGameInfoSubcodePillbox = 0x02
const OpcodeGameInfoSubcodeBase = 0x03
const OpcodeGameInfoSubcodeStart = 0x04

/* Macs count time since Midnight, 1st Jan 1904. Unix counts from 1970.
   This value adjusts for the 66 years and 17 leap-days difference. */
const seconds1904ToUnixEpoch = (((1970-1904)*365 + 17) * 24 * 60 * 60)

type GameId [8]byte

type GameInfo struct {
	GameId               GameId
	ServerStartTimestamp time.Time
	MapName              string
	StartTimestamp       uint32
	GameType             int
	AllowHiddenMines     bool
	AllowComputer        bool
	ComputerAdvantage    bool
	StartDelay           uint32
	TimeLimit            uint32
	PlayerCount          uint16
	NeutralPillboxCount  uint16
	NeutralBaseCount     uint16
	HasPassword          bool
}

var opcodeLengthLookup = []int{
	4, 6, 8, 10, 4, 1, 3, 3,
	1, 1, 1, 1, 1, 1, 1, 1,
	2, 0, 3, 0, 2, 3, 1, 1,
	0, 2, 0, 4, 2, 1, 1, 3,
	1, 1, 1, 1, 1, 3, 1, 1,
	3, 1, 1, 1, 1, 1, 1, 1,
	0, 1, 3, 3, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
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

	copy(gameInfo.GameId[:], msg[pos:pos+8])

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
	fmt.Println("Game Id:", gameInfo.GameId)
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

func rewriteOpcodePlayerInfo(
	pos int,
	buffer []byte,
	proxyPort int,
	proxyIP net.IP,
	srcRoute proxy.Route,
	leaveGameChannel chan proxy.Route,
) {
	// skip address length, first address
	pos = pos + 7

	playerPort := binary.BigEndian.Uint16(buffer[pos+4 : pos+6])
	fmt.Printf("Player disconnecting %d.%d.%d.%d:%d\n", buffer[pos+0], buffer[pos+1], buffer[pos+2], buffer[pos+3], playerPort)
	//if bytes.Equal(srcRoute.PlayerIPAddr.IP, buffer[pos:pos+4]) && int(playerPort) == srcRoute.PlayerIPAddr.Port {
	if !bytes.Equal(buffer[pos:pos+4], proxyIP) {
		fmt.Println("Sending LeaveGame event")
		leaveGameChannel <- srcRoute
	}

	if !bytes.Equal(proxyIP, buffer[pos:pos+4]) {
		buffer[pos+0] = proxyIP[0]
		buffer[pos+1] = proxyIP[1]
		buffer[pos+2] = proxyIP[2]
		buffer[pos+3] = proxyIP[3]
		binary.BigEndian.PutUint16(buffer[pos+4:pos+6], uint16(proxyPort))
	}
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

// parseOpcode returns the opcode and the length (including the opcode byte(s))
func parseOpcode(pos int, buffer []byte) (int, int) {
	opcode := int(buffer[pos])
	pos = pos + 1
	offset := 0

	if opcode == 0xff {
		opcode = int(buffer[pos])
		pos = pos + 1
		offset = 0x20
	}

	if opcode < 0xf0 {
		opcode = opcode >> 4
	} else {
		opcode = opcode & 0x1f
	}

	opcode = opcode + offset
	opcodeLength := 0

	switch opcode {
	case OpcodeDisconnect:
		addressLength := int(buffer[pos])
		opcodeLength = (addressLength * 3) + 2
	case OpcodeGameInfo:
		subcode := int(buffer[pos])
		count := int(buffer[pos+1])

		switch subcode {
		case OpcodeGameInfoSubcodeGame:
			opcodeLength = 90
		case OpcodeGameInfoSubcodePillbox:
			opcodeLength = (count * 5) + 3
		case OpcodeGameInfoSubcodeBase:
			opcodeLength = (count * 6) + 3
		case OpcodeGameInfoSubcodeStart:
			opcodeLength = (count * 3) + 3
		default:
			opcodeLength = 42
		}
	case OpcodeMapData:
		mapDataLength := int(buffer[pos+2])
		opcodeLength = mapDataLength + 3
	case OpcodePlayerName:
		playerNameLength := int(buffer[pos])
		opcodeLength = playerNameLength + 2
	case OpcodeSendMessage:
		messageLength := int(buffer[pos+2])
		opcodeLength = messageLength + 4
	default:
		opcodeLength = opcodeLengthLookup[opcode]
	}

	return opcode, opcodeLength
}

func rewriteGameStateBlock(
	packetSequence int,
	posStart int,
	buffer []byte,
	proxyPort int,
	proxyIP net.IP,
	srcRoute proxy.Route,
	leaveGameChannel chan proxy.Route,
) int {
	// block length includes length byte, does not include checksum
	blockLength := int(buffer[posStart] & 0x7f)
	posBlockStart := posStart + 1
	posChecksum := posStart + blockLength
	posNextBlock := posChecksum + 2
	rewriteCrc := false

	if blockLength < 4 {
		if blockLength == 0 {
			// don't know what this is, can't continue parsing
			posNextBlock = len(buffer) // skip to end
		}
		return posNextBlock
	}

	//blockSequence := buffer[posBlockStart]
	pos := posBlockStart + 1 // skip sequence
	senderFlags := buffer[pos] & 0xf0
	pos = pos + 1
	flags := buffer[pos]
	pos = pos + 1

	if flags&0x80 > 0 {
		pos = pos + 5
	}

	if senderFlags&0xe0 > 0 {
		pos = pos + 3
	}

	for pos < posChecksum {
		opcode, opcodeLength := parseOpcode(pos, buffer)

		/*
			fmt.Printf("PacketLength: %d PacketSequence: 0x%02x BlockSequence: 0x%02x BlockLength: %d RawOpcode: 0x%02x Opcode: 0x%02x OpcodeLength: %d\n",
				len(buffer), packetSequence, blockSequence, blockLength, buffer[pos], opcode, opcodeLength)
		*/

		switch opcode {
		case OpcodeGameInfo:
			subcode := int(buffer[pos+1])
			//pos = pos + 1

			if subcode == OpcodeGameInfoSubcodeGame {
				rewriteOpcodeGameInfo(pos+2, buffer, proxyPort, proxyIP)
				rewriteCrc = true
			}
		case OpcodeDisconnect:
			rewriteOpcodePlayerInfo(pos+2, buffer, proxyPort, proxyIP, srcRoute, leaveGameChannel)
			rewriteCrc = true
		}

		pos = pos + opcodeLength
	}

	if rewriteCrc {
		crc64 := crc.CalculateCRC(crc.XMODEM, buffer[posStart:posChecksum])
		binary.BigEndian.PutUint16(buffer[posChecksum:posNextBlock], uint16(crc64))
	}

	return posNextBlock
}

func rewritePacketGameState(buffer []byte, proxyIP net.IP, proxyPort int, srcRoute proxy.Route, leaveGameChannel chan proxy.Route) {
	pos := PacketHeaderSize
	packetSequence := int(buffer[pos])
	pos = pos + 1 // skip state sequence

	for pos < len(buffer) {
		pos = rewriteGameStateBlock(packetSequence, pos, buffer, proxyPort, proxyIP, srcRoute, leaveGameChannel)
	}
}

func rewritePacketFixedPosition(buffer []byte, proxyIP net.IP, proxyPort int, offset int) {
	packetIP := buffer[offset : offset+4]
	if !bytes.Equal(packetIP, proxyIP) {
		port := make([]byte, 2)
		binary.BigEndian.PutUint16(port, uint16(proxyPort))
		buffer[offset+0] = proxyIP[0]
		buffer[offset+1] = proxyIP[1]
		buffer[offset+2] = proxyIP[2]
		buffer[offset+3] = proxyIP[3]
		buffer[offset+4] = port[0]
		buffer[offset+5] = port[1]
	}
}

func RewritePacket(buffer []byte, proxyIP net.IP, proxyPort int, srcRoute proxy.Route, leaveGameChannel chan proxy.Route) {
	// only the player who starts the game will send packets with the wrong ip address, and it will
	// be their own. so we can search for any ip that isn't ours, replace it with ours, and replace
	// the port with the player's assigned port

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			fmt.Println(hex.Dump(buffer))
		}
	}()

	switch buffer[PacketTypeOffset] {
	case PacketType0:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType0PeerAddrOffset)
	case PacketType1:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType1PeerAddrOffset)
	case PacketTypeGameState:
		rewritePacketGameState(buffer, proxyIP, proxyPort, srcRoute, leaveGameChannel)
	case PacketType6:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType6PeerAddrOffset)
	case PacketType7:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType7PeerAddrOffset)
	case PacketType9:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType9PeerAddrOffset)
	}
}
