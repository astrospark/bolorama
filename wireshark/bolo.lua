bolo_protocol = Proto("Bolo",  "Bolo Protocol")

------ Field Definitions ------

local boolean_values =
{
	[0] = "False",
	[1] = "True"
}

unknown_field = ProtoField.bytes("bolo.unknown", "Unknown", base.SPACE)
padding_field = ProtoField.bytes("bolo.padding", "Padding", base.SPACE)

-- Header
signature_field = ProtoField.string("bolo.signature", "Signature", base.ASCII)
version_field = ProtoField.string("bolo.version", "Version", base.ASCII)
packet_type_field = ProtoField.uint8("bolo.packet_type", "Packet Type", base.HEX)

-- Packet Type 0x02
sequence_field = ProtoField.uint8("bolo.sequence", "Sequence", base.HEX)
block_field = ProtoField.uint8("bolo.block", "Block", base.UNIT_STRING, {" byte", " bytes"})
sender_flags_field = ProtoField.uint8("bolo.sender_flags", "Sender Flags", base.HEX, nil, 0xf0)
sender_field = ProtoField.uint8("bolo.sender", "Sender", base.HEX, nil, 0x0f)
block_flags_field = ProtoField.uint8("bolo.block_flags", "Block Flags", base.HEX)

opcode_field = ProtoField.uint8("bolo.opcode", "Opcode", base.HEX)
subcode_field = ProtoField.uint8("bolo.subcode", "Subcode", base.HEX)
checksum_field = ProtoField.uint16("bolo.checksum", "Checksum", base.HEX)

host_address_field = ProtoField.ipv4("bolo.host_address", "Host Address")

-- Opcode 0xfa
message_field = ProtoField.string("bolo.message", "Message", base.ASCII)

-- Opcode 0xff
address_length_field = ProtoField.uint8("bolo.address_length", "Address Length", base.UNIT_STRING, {" Byte", " Bytes"})
upstream_address_field = ProtoField.string("bolo.upstream_address", "Upstream Address")
sender_address_field = ProtoField.string("bolo.sender_address", "Sender Address")
downstream_address_field = ProtoField.string("bolo.downstream_address", "Downstream Address")

map_pillbox_count_field = ProtoField.uint8("bolo.map_pillbox_count", "Map Pillbox Count", base.DEC)
map_pillbox_data_field = ProtoField.bytes("bolo.map_pillbox_data", "Map Pillbox Data", base.SPACE)
map_base_count_field = ProtoField.uint8("bolo.map_base_count", "Map Base Count", base.DEC)
map_base_data_field = ProtoField.bytes("bolo.map_base_data", "Map Base Data", base.SPACE)
map_start_count_field = ProtoField.uint8("bolo.map_start_count", "Map Start Count", base.DEC)
map_start_data_field = ProtoField.bytes("bolo.map_start_data", "Map Start Data", base.SPACE)

player_name_field = ProtoField.string("bolo.player_name", "Player Name", base.ASCII)

map_name_field = ProtoField.string("bolo.map_name", "Map Name", base.ASCII)

start_time_field = ProtoField.string("bolo.start_time", "Start Time", base.ASCII)

local game_type_values =
{
	[1] = "Open Game",
	[2] = "Tournament",
	[3] = "Strict Tournament"
}
game_type_field = ProtoField.uint8("bolo.game_type", "Game Type", base.HEX, game_type_values)

game_flags_field = ProtoField.uint8("bolo.game_flags", "Game Flags", base.HEX, nil, 0xbf)
mines_visible_field = ProtoField.uint8("bolo.mines_visible", "Mines Visible", base.HEX, boolean_values, 0x40)
allow_computer_field = ProtoField.bool("bolo.allow_computer", "Allow Computer")
computer_advantage_field = ProtoField.bool("bolo.computer_advantage", "Computer Advantage")
start_delay_field = ProtoField.uint32("bolo.start_delay", "Start Delay", base.UNIT_STRING, {" Second", " Seconds"})
time_limit_field = ProtoField.uint32("bolo.time_limit", "Time Limit", base.UNIT_STRING, {" Minute", " Minutes"})

-- Packet Type 0x06
peer_address_field = ProtoField.string("bolo.peer_address", "Peer Address")

-- Packet Type 0x08 Password
password_field = ProtoField.string("bolo.password", "Password", base.ASCII)

-- Packet Type 0x0E
num_players_field = ProtoField.uint16("bolo.num_players", "Number of Players", base.DEC)
free_pills_field = ProtoField.uint16("bolo.free_pills", "Free Pills", base.DEC)
free_bases_field = ProtoField.uint16("bolo.free_bases", "Free Bases", base.DEC)
has_password_field = ProtoField.bool("bolo.has_password", "Has Password")

bolo_protocol.fields = {
	unknown_field, padding_field,

	-- Header
	signature_field, version_field, packet_type_field,

	-- Packet Type 0x02 Game State
	sequence_field, block_field,
	sender_flags_field, sender_field, block_flags_field,
	opcode_field, subcode_field, checksum_field,
	host_address_field,
	message_length_field, message_field,
	map_pillbox_count_field, map_pillbox_data_field,
	map_base_count_field, map_base_data_field,
	map_start_count_field, map_start_data_field,

	player_name_length_field, player_name_field,
	map_name_length_field, map_name_field,
	start_time_field,
	game_type_field, game_flags_field, mines_visible_field,
	allow_computer_field, computer_advantage_field,
	start_delay_field, time_limit_field,

	-- Packet Type 0x06
	peer_address_field,

	-- Opcode 0xff
	address_length_field, upstream_address_field, sender_address_field, downstream_address_field,

	-- Packet Type 0x08 Password
	password_length_field, password_field,

	-- Packet Type 0x0E
	num_players_field, free_pills_field, free_bases_field,
	has_password_field
}

------ Expert Definitions ------

unknown_packet_type_expert = ProtoExpert.new("bolo.unknown_packet_type_expert.expert", "Unknown packet type", expert.group.UNDECODED, expert.severity.WARN)
opcode_buffer_underrun_expert = ProtoExpert.new("bolo.opcode_buffer_underrun.expert", "Opcode buffer underrun", expert.group.MALFORMED, expert.severity.WARN)
invalid_string_length_expert = ProtoExpert.new("bolo.invalid_string_length.expert", "Invalid string length", expert.group.MALFORMED, expert.severity.WARN)

bolo_protocol.experts = {
	unknown_packet_type_expert,
	opcode_buffer_underrun_expert,
	invalid_string_length_expert
}

------ Packet Type Dissectors ------

function dissect_packet_type_00(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x00")
	local pos = 0

	if buffer_length == 6 then
		local sender_ip = buffer(pos, 4):ipv4()
		local sender_port = buffer(pos + 4, 2):uint()
		local sender_address = string.format("%s:%d", sender_ip, sender_port)
		t:add(sender_address_field, buffer(pos, 6), sender_address); pos = pos + 6
		t:append_text(string.format(", Sender: %s:%d", sender_address, sender_port))
	else
		t:add(unknown_field, buffer())
	end
end

function dissect_packet_type_01(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x01")
	local pos = 0

	if buffer_length == 6 then
		local sender_ip = buffer(pos, 4):ipv4()
		local sender_port = buffer(pos + 4, 2):uint()
		local sender_address = string.format("%s:%d", sender_ip, sender_port)
		t:add(sender_address_field, buffer(pos, 6), sender_address); pos = pos + 6
		t:append_text(string.format(", Sender: %s:%d", sender_address, sender_port))
	else
		t:add(unknown_field, buffer())
	end
end

function dissect_game_state(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Game State")
	local pos = 0

	local sequence = buffer(pos, 1):uint()
	t:append_text(string.format(", Sequence: 0x%02x", sequence))
	t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

	while pos < buffer_length do
		pos = pos + dissect_block(buffer(pos), pinfo, t)
	end
end

function dissect_packet_type_03(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x03")
	local pos = 0

	local sequence = buffer(pos, 1):uint()
	t:append_text(string.format(", Sequence: 0x%02x", sequence))
	t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

	t:add(unknown_field, buffer())
end

function dissect_game_state_acknowledge(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Game State Acknowledge")
	local pos = 0

	local sequence = buffer(pos, 1):uint()
	t:append_text(string.format(", Sequence: 0x%02x", sequence))
	t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

	if pos < buffer_length then
		t:add(unknown_field, buffer())
	end
end

function dissect_packet_type_05(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x05")
	t:add(unknown_field, buffer())
end

function dissect_packet_type_06(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x06")
	local pos = 0

	if buffer_length >= 10 then
		t:add(unknown_field, buffer(pos, 4)); pos = pos + 4

		local peer_ip = buffer(pos, 4):ipv4()
		local peer_port = buffer(pos + 4, 2):uint()
		local peer_address = string.format("%s:%d", peer_ip, peer_port)
		t:add(peer_address_field, buffer(pos, 6), peer_address); pos = pos + 6

		t:append_text(string.format(", Peer: %s", peer_address))
	end

	if pos < buffer_length then
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_packet_type_07(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x07")
	local pos = 0

	if buffer_length >= 10 then
		t:add(unknown_field, buffer(pos, 4)); pos = pos + 4

		local peer_ip = buffer(pos, 4):ipv4()
		local peer_port = buffer(pos + 4, 2):uint()
		local peer_address = string.format("%s:%d", peer_ip, peer_port)
		t:add(peer_address_field, buffer(pos, 6), peer_address); pos = pos + 6

		t:append_text(string.format(", Peer: %s", peer_address))
	end

	if pos < buffer_length then
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_password(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Password")
	local pos = 0

	if buffer_length >= 36 then
		local password, password_length = dissect_pascal_string(buffer(pos, 36), t, password_field, 35)

		if password ~= nil then
			t:append_text(string.format(", Password: %s", password))
			pos = pos + 36
		end
	end

	if pos < buffer_length then
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_packet_type_09(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x09")
	local pos = 0

	if buffer_length >= 6 then
		local peer_ip = buffer(pos, 4):ipv4()
		local peer_port = buffer(pos + 4, 2):uint()
		local peer_address = string.format("%s:%d", peer_ip, peer_port)
		t:add(peer_address_field, buffer(pos, 6), peer_address); pos = pos + 6

		t:append_text(string.format(", Peer: %s", peer_address))
	end

	if pos < buffer_length then
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_game_info_request(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Game Info Request")
	t:add(unknown_field, buffer())
end

function dissect_game_info(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Game Info")
	local pos = 0

	if buffer_length >= 63 then
		local map_name, map_name_length = dissect_pascal_string(buffer(pos, 36), t, map_name_field, 35)

		if map_name ~= nil then
			t:append_text(string.format(", Map: %s", map_name))
			pos = pos + 36
		else
			if pos < buffer_length then
				t:add(unknown_field, buffer(pos))
			end
			return
		end

		local host_address = buffer(pos, 4):ipv4()
		t:append_text(string.format(", Host: %s", host_address))
		t:add(host_address_field, buffer(pos, 4)); pos = pos + 4

		local start_time_mac = buffer(pos, 4):uint()
		local start_time = convert_time_from_mac(start_time_mac)
		local start_time_string = os.date("%c", start_time)
		t:add(start_time_field, buffer(pos, 4), start_time_string); pos = pos + 4

		t:add(game_type_field, buffer(pos, 1)); pos = pos + 1

		t:add(game_flags_field, buffer(pos, 1))
		t:add(mines_visible_field, buffer(pos, 1)); pos = pos + 1

		t:add(allow_computer_field, buffer(pos, 1)); pos = pos + 1
		t:add(computer_advantage_field, buffer(pos, 1)); pos = pos + 1

		local start_delay = buffer(pos, 4):le_uint()
		if start_delay ~= 0 then start_delay = (start_delay / 50) + 1 end
		t:add(start_delay_field, buffer(pos, 4), start_delay); pos = pos + 4

		local time_limit = buffer(pos, 4):le_uint()
		if time_limit ~= 0 then time_limit = (time_limit / 50 / 60) + 1 end
		t:add(time_limit_field, buffer(pos, 4), time_limit); pos = pos + 4

		t:add_le(num_players_field, buffer(pos, 2)); pos = pos + 2
		t:add_le(free_pills_field, buffer(pos, 2)); pos = pos + 2
		t:add_le(free_bases_field, buffer(pos, 2)); pos = pos + 2

		t:add(has_password_field, buffer(pos, 1)); pos = pos + 1
	end

	if pos < buffer_length then
		t:add(unknown_field, buffer(pos))
	end
end

local packet_type_dissectors =
{
	[0x00] = dissect_packet_type_00,
	[0x01] = dissect_packet_type_01,
	[0x02] = dissect_game_state,
	[0x03] = dissect_packet_type_03,
	[0x04] = dissect_game_state_acknowledge,
	[0x05] = dissect_packet_type_05,
	[0x06] = dissect_packet_type_06,
	[0x07] = dissect_packet_type_07,
	[0x08] = dissect_password,
	[0x09] = dissect_packet_type_09,
	[0x0d] = dissect_game_info_request,
	[0x0e] = dissect_game_info
}

local packet_type_names =
{
	[0x02] = "Game State",
	[0x04] = "Game State Acknowledge",
	[0x08] = "Password",
	[0x0d] = "Game Info Request",
	[0x0e] = "Game Info"
}

------ Opcode Dissectors ------

function dissect_opcode_game_info(buffer, pinfo, tree) -- 0xf1
	local buffer_length = buffer:len()
	local pos = 0

	local opcode = buffer(pos, 1):uint()
	local t = tree:add(opcode_field, buffer(), opcode); pos = pos + 1
	t:append_text(" (Game Info)")

	local subcode = buffer(pos, 1):uint()

	if subcode == 0x01 then
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local map_name, map_name_length = dissect_pascal_string(buffer(pos), t, map_name_field, 35)

		if map_name ~= nil then
			t:append_text(string.format(", Map: %s", map_name))
			pos = pos + 36
		end
		
		local host_address = buffer(pos, 4):ipv4()
		t:append_text(string.format(", Host: %s", host_address))
		t:add(host_address_field, buffer(pos, 4)); pos = pos + 4

		local start_time_mac = buffer(pos, 4):uint()
		local start_time = convert_time_from_mac(start_time_mac)
		local start_time_string = os.date("%c", start_time)
		t:add(start_time_field, buffer(pos, 4), start_time_string); pos = pos + 4

		t:add(game_type_field, buffer(pos, 1)); pos = pos + 1

		t:add(game_flags_field, buffer(pos, 1))
		t:add(mines_visible_field, buffer(pos, 1)); pos = pos + 1

		t:add(allow_computer_field, buffer(pos, 1)); pos = pos + 1
		t:add(computer_advantage_field, buffer(pos, 1)); pos = pos + 1
		t:add_le(start_delay_field, buffer(pos, 4)); pos = pos + 4
		t:add_le(time_limit_field, buffer(pos, 4)); pos = pos + 4

		t:add(unknown_field, buffer(pos, 32)); pos = pos + 32
	elseif subcode == 0x02 then
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local pillbox_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Pillbox Count: %d", pillbox_count))
		t:add(map_pillbox_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, pillbox_count - 1 do
			t:add(map_pillbox_data_field, buffer(pos, 5)); pos = pos + 5
		end
	elseif subcode == 0x03 then
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local base_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Base Count: %d", base_count))
		t:add(map_base_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, base_count - 1 do
			t:add(map_base_data_field, buffer(pos, 6)); pos = pos + 6
		end
	elseif subcode == 0x04 then
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local start_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Start Count: %d", start_count))
		t:add(map_start_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, start_count - 1 do
			t:add(map_start_data_field, buffer(pos, 3)); pos = pos + 3
		end
	else
		t:add(unknown_field, buffer(pos, buffer_length - pos)); pos = buffer_length
	end

	if pos < buffer_length then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_opcode_map_data(buffer, pinfo, tree) -- 0xf3
	local buffer_length = buffer:len()
	local pos = 0

	local opcode = buffer(pos, 1):uint()
	local t = tree:add(opcode_field, buffer(), opcode); pos = pos + 1
	t:append_text(" (Map Data)")

	if buffer_length >= 3 then
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		return
	end

	local length = buffer(pos, 1):uint() - 1
	t:add(unknown_field, buffer(pos, 1)); pos = pos + 1

	if length > buffer_length - pos then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
		return
	end

	t:add(unknown_field, buffer(pos, length)); pos = pos + length

	if pos < buffer_length then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_opcode_player_name(buffer, pinfo, tree) -- 0xf8
	local buffer_length = buffer:len()
	local pos = 0

	local opcode = buffer(pos, 1):uint()
	local t = tree:add(opcode_field, buffer(), opcode); pos = pos + 1
	t:append_text(" (Player Name)")

	local player_name, player_name_length = dissect_pascal_string(buffer(pos), t, player_name_field)

	if player_name ~= nil then
		t:append_text(string.format(", Player Name: %s", player_name))
		pos = pos + player_name_length + 1
	else
		t:add_proto_expert_info()
		t:add(unknown_field, buffer(pos))
		return
	end

	if pos < buffer_length then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_opcode_send_message(buffer, pinfo, tree) -- 0xfa
	local buffer_length = buffer:len()
	local pos = 0

	local opcode = buffer(pos, 1):uint()
	local t = tree:add(opcode_field, buffer(), opcode); pos = pos + 1
	t:append_text(" (Send Message)")

	if buffer_length >= 3 then
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		return
	end

	local message, message_length = dissect_pascal_string(buffer(pos), t, message_field)

	if message ~= nil then
		t:append_text(string.format(", Message: %s", message))
		pos = pos + message_length + 1
	else
		t:add_proto_expert_info()
		t:add(unknown_field, buffer(pos))
		return
	end

	if pos < buffer_length then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
	end
end

function dissect_opcode_disconnect(buffer, pinfo, tree) -- 0xfff0
	local buffer_length = buffer:len()
	local pos = 0

	local opcode = buffer(pos, 1):uint()
	local t = tree:add(opcode_field, buffer(), opcode); pos = pos + 1
	t:append_text(" (Disconnect)")

	t:add(address_length_field, buffer(pos, 1)); pos = pos + 1

	local upstream_ip = buffer(pos, 4):ipv4()
	local upstream_port = buffer(pos + 4, 2):uint()
	local upstream_address = string.format("%s:%d", upstream_ip, upstream_port)
	t:add(upstream_address_field, buffer(pos, 6), upstream_address); pos = pos + 6
	t:append_text(string.format(", Upstream: %s", upstream_address))

	local sender_ip = buffer(pos, 4):ipv4()
	local sender_port = buffer(pos + 4, 2):uint()
	local sender_address = string.format("%s:%d", sender_ip, sender_port)
	t:add(sender_address_field, buffer(pos, 6), sender_address); pos = pos + 6
	t:append_text(string.format(", Sender: %s", sender_address))

	local downstream_ip = buffer(pos, 4):ipv4()
	local downstream_port = buffer(pos + 4, 2):uint()
	local downstream_address = string.format("%s:%d", downstream_ip, downstream_port)
	t:add(downstream_address_field, buffer(pos, 6), downstream_address); pos = pos + 6
	t:append_text(string.format(", Downstream: %s", downstream_address))

	if pos < buffer_length then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
	end
end

local opcode_dissectors =
{
	[0xf1] = dissect_opcode_game_info,
	[0xf3] = dissect_opcode_map_data,
	[0xf8] = dissect_opcode_player_name,
	[0xfa] = dissect_opcode_send_message,
	[0xf0] = dissect_opcode_disconnect
}

------ Block Dissector ------

local opcode_lengths =
{
	    4,     6,     8,    10,     4,     1,     3,     3,
	    1,     1,     1,     1,     1,     1,     1,     1,
	    2, 0x1f1,     3, 0x1f3,     2,     3,     1,     1,
	0x1f8,     2, 0x1fa,     4,     2,     1,     1,     3,
	    1,     1,     1,     1,     1,     3,     1,     1,
	    3,     1,     1,     1,     1,     1,     1,     1,
	0x1f0,     1,     3,     3,     1,     1,     1,     1,
	    1,     1,     1,     1,     1,     1,     1,     1,
}

function get_opcode_length_game_info(buffer) -- 0xf1
	local subcode = buffer(1, 1):uint()
	local count = buffer(2, 1):uint()

	if subcode == 1 then -- Game Info
		return 0x5a -- 90
	elseif subcode == 2 then -- Pillbox Info
		return (count * 5) + 3
	elseif subcode == 3 then -- Base Info
		return (count * 6) + 3
	elseif subcode == 4 then -- Start Info
		return (count * 3) + 3
	else
		return 0x2a -- 42
	end
end

function get_opcode_length_map_data(buffer) -- 0xf3
	local map_data_length = buffer(3, 1):uint()
	return map_data_length + 3
end

function get_opcode_length_player_name(buffer) -- 0xf8
	local player_name_length = buffer(1, 1):uint()
	return player_name_length + 2
end

function get_opcode_length_send_message(buffer) -- 0xfa
	local message_length = buffer(3, 1):uint()
	return message_length + 4
end

function get_opcode_length_disconnect(buffer) -- 0xfff0
	local address_length = buffer(1, 1):uint()
	return (address_length * 3) + 2
end

local opcode_length_calculators =
{
	[0xf1] = get_opcode_length_game_info,
	[0xf3] = get_opcode_length_map_data,
	[0xf8] = get_opcode_length_player_name,
	[0xfa] = get_opcode_length_send_message,
	[0xf0] = get_opcode_length_disconnect
}

function dissect_block(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return 0 end
	local pos = 0

	-- block_length includes length byte but not CRC
	local block_length = bit.band(buffer(pos, 1):uint(), 0x7f)

	if (block_length < 4) then
		tree:add(unknown_field, buffer(pos))
		return buffer_length
	end

	local t = tree:add(block_field, buffer(pos, block_length + 2), block_length); pos = pos + 1

	local sequence = buffer(pos, 1):uint()
	t:append_text(string.format(", Sequence: 0x%02x", sequence))
	t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

	local sender = bit.band(buffer(pos, 1):uint(), 0x0f)
	t:append_text(string.format(", Sender: 0x%02x", sender))
	t:add(sender_flags_field, buffer(pos, 1))
	t:add(sender_field, buffer(pos, 1)); pos = pos + 1

	local flags = buffer(pos, 1):uint()
	t:add(block_flags_field, buffer(pos, 1)); pos = pos + 1

	-- pos should be 4 at this point
	if bit.band(flags, 0x80) ~= 0 then
		t:add(unknown_field, buffer(pos, 9 - pos))
		pos = 9
	end
	if bit.band(sender, 0xe0) ~= 0 then
		t:add(unknown_field, buffer(pos, 3))
		pos = pos + 3
	end

	while (pos < block_length) do
		local opcode = buffer(pos, 1):uint()

		local offset = 0
		if opcode == 0xff then
			pos = pos + 1
			opcode = buffer(pos, 1):uint()
			offset = 0x20
		end

		if (opcode < 0xf0) then
			opcode = bit.rshift(opcode, 4)
		else
			opcode = bit.band(opcode, 0x1f)
		end

		local index = opcode + offset
		local opcode_length = opcode_lengths[index + 1] -- Lua arrays start at 1

		if opcode_length > 0xff then
			index = bit.band(opcode_length, 0xff)
			local opcode_length_calculator = opcode_length_calculators[index]
			opcode_length = opcode_length_calculator(buffer(pos))

			-- dissect
			local opcode_dissector = opcode_dissectors[index]
			opcode_dissector(buffer(pos, opcode_length), pinfo, t)
		else
			opcode = buffer(pos, 1):uint() -- get opcode again because it may have been mangled above
			local opcode_tree = t:add(opcode_field, buffer(pos, opcode_length), opcode)
			opcode_tree:append_text(string.format(" (Unknown), Length: %d", opcode_length))
			if opcode_length > 1 then
				opcode_tree:add(unknown_field, buffer(pos + 1, opcode_length - 1))
			end
		end

		pos = pos + opcode_length
	end

	t:add(checksum_field, buffer(pos, 2)); pos = pos + 2

	return pos
end

------ Header Dissector ------

function bolo_protocol.dissector(buffer, pinfo, tree)
	pinfo.cols.protocol = bolo_protocol.name
	local t = tree:add(bolo_protocol, buffer(), "Bolo Protocol")

	t:add(signature_field, buffer(0, 4))

	local version_major = buffer(4, 1):uint()
	local version_minor = buffer(5, 1):uint()
	local version_build = buffer(6, 1):uint()
	local version_string = string.format("%x.%x.%x", version_major, version_minor, version_build)
	t:append_text(string.format(", Version: %s", version_string))
	t:add(version_field, buffer(4, 3), version_string)

	local packet_type = buffer(7, 1):uint()
	local packet_type_name = packet_type_names[packet_type]
	if packet_type_name == nil then packet_type_name = "Unknown" end
	t:append_text(string.format(", Packet Type: %s (0x%02x)", packet_type_name, packet_type))
	t:add(packet_type_field, buffer(7, 1)):append_text(string.format(" (%s)", packet_type_name))

	local packet_type_dissector = packet_type_dissectors[packet_type]
	if packet_type_dissector ~= nil then
		packet_type_dissector(buffer(8), pinfo, tree)
	else
		t:add_proto_expert_info(unknown_packet_type_expert)
		if buffer:len() > 8 then
			t:add(unknown_field, buffer(8))
		end
	end
end

------ Utility Functions ------

function dissect_pascal_string(buffer, tree, field, fixed_length)
	fixed_length = fixed_length or 0

	local buffer_length = buffer:len()
	if buffer_length < 1 then return nil, 0 end

	local pos = 0

	local string_length = buffer(pos, 1):uint()
	local string_end = pos + string_length + 1

	if (fixed_length ~= 0 and
	    buffer_length < fixed_length + 1 and
	    string_length > fixed_length) or 
	   string_end > buffer_length then
		tree:add_proto_expert_info(invalid_string_length_expert)
		return nil, 0
	end

	local string_contents = buffer(pos + 1, string_length):string()
	if string_length == 0 then string_contents = "[empty]" end
	tree:add(field, buffer(pos, string_length + 1), string_contents)
	--	:append_text(string.format(" (%d)", string_length))

	if fixed_length == 0 then
		pos = pos + string_length + 1
	else
		local padding_length = fixed_length - string_length
		if padding_length > 0 then
			tree:add(padding_field, buffer(string_end, padding_length))
		end

		pos = pos + fixed_length + 1
	end

	return string_contents, string_length
end

function convert_time_from_mac(mac_time)
	return mac_time - 2082844800
end

------ Dissector Registration ------

local function heuristic_checker(buffer, pinfo, tree)
    if buffer:len() < 8 then return false end
    if buffer(0, 4):string() ~= "Bolo" then return false end
    bolo_protocol.dissector(buffer, pinfo, tree)
    return true
end

bolo_protocol:register_heuristic("udp", heuristic_checker)
