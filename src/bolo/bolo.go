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

package bolo

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"git.astrospark.com/bolorama/proxy"
	"git.astrospark.com/bolorama/util"
	"github.com/snksoft/crc"
)

const PacketHeaderSize = 8
const boloSignature = "Bolo"

const hexPacketSignature = "426f6c6f"
const hexPacketVersion = "659908"

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
	return bytes.Equal(msg[4:7], []byte{0x65, 0x99, 0x08})
}

func GetPacketType(msg []byte) int {
	return int(msg[7])
}

func ValidatePacket(packet proxy.UdpPacket) (bool, string) {
	if packet.Len < PacketHeaderSize {
		return false, fmt.Sprintf("datagram too short (smaller than bolo header) (%d)", packet.Len)
	}

	if !verifyBoloSignature(packet.Buffer) {
		return false, fmt.Sprintf("datagram failed bolo signature check (%s)", hex.EncodeToString(packet.Buffer[0:4]))
	}

	if !verifyBoloVersion(packet.Buffer) {
		return false, fmt.Sprintf("unsupported bolo version (%s)", hex.EncodeToString(packet.Buffer[4:7]))
	}

	return true, ""
}

func MarshalPacketType6(ipAddr net.IP, port int) []byte {
	var portBytes [2]byte
	binary.BigEndian.PutUint16(portBytes[:], uint16(port))
	ipHex := hex.EncodeToString(ipAddr.To4())
	portHex := hex.EncodeToString(portBytes[:])
	packetHex := hexPacketSignature + hexPacketVersion + "06ffff0123" + ipHex + portHex + "456789ab"
	buffer, err := hex.DecodeString(packetHex)
	if err != nil {
		return []byte{}
	}
	return buffer
}

func MarshalPacketTypeD() []byte {
	buffer, err := hex.DecodeString(hexPacketSignature + hexPacketVersion + "0d")
	if err != nil {
		return []byte{}
	}
	return buffer
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
	fmt.Println("Game Id:", hex.EncodeToString(gameInfo.GameId[:]))
	fmt.Println("Map Name:", gameInfo.MapName)
	fmt.Println("Start Timestamp:", ParseBoloTimestamp(gameInfo.StartTimestamp))
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
	fmt.Println()
}

func ParseBoloTimestamp(timestamp uint32) time.Time {
	return time.Unix(int64(timestamp-seconds1904ToUnixEpoch), 0)
}

func rewriteOpcodePlayerInfo(
	pos int,
	buffer []byte,
	proxyPort int,
	proxyIP net.IP,
	srcPlayer util.PlayerAddr,
	playerLeaveGameChannel chan util.PlayerAddr,
) {
	// skip address length, first address
	pos = pos + 7

	playerPort := binary.BigEndian.Uint16(buffer[pos+4 : pos+6])
	fmt.Printf("Player disconnecting: %d (NAT %d.%d.%d.%d:%d)\n", proxyPort, buffer[pos+0], buffer[pos+1], buffer[pos+2], buffer[pos+3], playerPort)
	//if bytes.Equal(srcRoute.PlayerIPAddr.IP, buffer[pos:pos+4]) && int(playerPort) == srcRoute.PlayerIPAddr.Port {
	if !bytes.Equal(buffer[pos:pos+4], proxyIP) {
		fmt.Println("Sending LeaveGame event")
		playerLeaveGameChannel <- srcPlayer
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

	// game id is more unique if we leave the original ip address
	/*
		buffer[pos+0] = proxyIP[0]
		buffer[pos+1] = proxyIP[1]
		buffer[pos+2] = proxyIP[2]
		buffer[pos+3] = proxyIP[3]
	*/
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
	srcPlayer util.PlayerAddr,
	playerInfoEventChannel chan util.PlayerInfoEvent,
	playerLeaveGameChannel chan util.PlayerAddr,
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
	sender := buffer[pos] & 0x0f
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
		case OpcodePlayerName:
			if (packetSequence == 0x02) && (buffer[posStart]&0x80 == 0) {
				playerInfoEventChannel <- util.PlayerInfoEvent{srcPlayer, true, false, int(sender), ""}
			}
			nameLength := int(buffer[pos+1])
			playerName := string(buffer[pos+2 : pos+2+nameLength])
			playerInfoEventChannel <- util.PlayerInfoEvent{srcPlayer, false, true, int(sender), playerName}
		case OpcodeDisconnect:
			rewriteOpcodePlayerInfo(pos+2, buffer, proxyPort, proxyIP, srcPlayer, playerLeaveGameChannel)
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

func rewritePacketGameState(
	buffer []byte,
	proxyIP net.IP,
	proxyPort int,
	srcPlayer util.PlayerAddr,
	playerInfoEventChannel chan util.PlayerInfoEvent,
	playerLeaveGameChannel chan util.PlayerAddr,
) {
	pos := PacketHeaderSize
	packetSequence := int(buffer[pos])
	pos = pos + 1 // skip state sequence

	for pos < len(buffer) {
		pos = rewriteGameStateBlock(
			packetSequence,
			pos,
			buffer,
			proxyPort,
			proxyIP,
			srcPlayer,
			playerInfoEventChannel,
			playerLeaveGameChannel,
		)
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

func RewritePacket(
	buffer []byte,
	proxyIP net.IP,
	proxyPort int,
	srcPlayer util.PlayerAddr,
	playerInfoEventChannel chan util.PlayerInfoEvent,
	playerLeaveGameChannel chan util.PlayerAddr,
) {
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
		rewritePacketGameState(buffer, proxyIP, proxyPort, srcPlayer, playerInfoEventChannel, playerLeaveGameChannel)
	case PacketType6:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType6PeerAddrOffset)
	case PacketType7:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType7PeerAddrOffset)
	case PacketType9:
		rewritePacketFixedPosition(buffer, proxyIP, proxyPort, PacketType9PeerAddrOffset)
	}
}
