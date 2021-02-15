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
player_field = ProtoField.uint8("bolo.player", "Player", base.DEC)
opcode_field = ProtoField.uint8("bolo.opcode", "Opcode", base.HEX)
subcode_field = ProtoField.uint8("bolo.subcode", "Subcode", base.HEX)

host_address_field = ProtoField.ipv4("bolo.host_address", "Host Address")

-- Opcode 0xfa
message_field = ProtoField.string("bolo.message", "Message", base.ASCII)

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

game_flags_field = ProtoField.uint8("bolo.game_flags", "Game Flags", base.HEX)
mines_visible_field = ProtoField.uint8("bolo.mines_visible", "Mines Visible", base.HEX, boolean_values, 0x40)
allow_computer_field = ProtoField.bool("bolo.allow_computer", "Allow Computer")
computer_advantage_field = ProtoField.bool("bolo.computer_advantage", "Computer Advantage")
start_delay_field = ProtoField.uint32("bolo.start_delay", "Start Delay", base.UNIT_STRING, {" Second", " Seconds"})
time_limit_field = ProtoField.uint32("bolo.time_limit", "Time Limit", base.UNIT_STRING, {" Minute", " Minutes"})

-- Packet Type 0x07
peer_address_field = ProtoField.ipv4("bolo.peer_address", "Peer Address")
peer_port_field = ProtoField.uint16("bolo.peer_port", "Peer Port")

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
	sequence_field, block_field, player_field,
	opcode_field, subcode_field,
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

	-- Packet Type 0x07
	peer_address_field, peer_port_field,

	-- Packet Type 0x08 Password
	password_length_field, password_field,

	-- Packet Type 0x0E
	num_players_field, free_pills_field, free_bases_field,
	has_password_field
}

------ Expert Definitions ------

unknown_packet_type_expert = ProtoExpert.new("bolo.unknown_packet_type_expert.expert", "Unknown packet type", expert.group.UNDECODED, expert.severity.WARN)
unknown_opcode_expert = ProtoExpert.new("bolo.unknown_opcode.expert", "Unknown opcode", expert.group.UNDECODED, expert.severity.WARN)
opcode_buffer_underrun_expert = ProtoExpert.new("bolo.opcode_buffer_underrun.expert", "Opcode buffer underrun", expert.group.MALFORMED, expert.severity.WARN)
invalid_string_length_expert = ProtoExpert.new("bolo.invalid_string_length.expert", "Invalid string length", expert.group.MALFORMED, expert.severity.WARN)

bolo_protocol.experts = {
	unknown_packet_type_expert,
	unknown_opcode_expert,
	opcode_buffer_underrun_expert,
	invalid_string_length_expert
}

------ Packet Type Dissectors ------

function dissect_packet_type_00(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x00")
	t:add(unknown_field, buffer())
end

function dissect_packet_type_01(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local t = tree:add(bolo_protocol, buffer(), "Bolo Packet Type 0x01")
	t:add(unknown_field, buffer())
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

		local peer_address = buffer(pos, 4):ipv4()
		t:add(peer_address_field, buffer(pos, 4)); pos = pos + 4

		local peer_port = buffer(pos, 2):uint()
		t:add(peer_port_field, buffer(pos, 2)); pos = pos + 2

		t:append_text(string.format(", Peer: %s:%d", peer_address, peer_port))
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

		local peer_address = buffer(pos, 4):ipv4()
		t:add(peer_address_field, buffer(pos, 4)); pos = pos + 4

		local peer_port = buffer(pos, 2):uint()
		t:add(peer_port_field, buffer(pos, 2)); pos = pos + 2

		t:append_text(string.format(", Peer: %s:%d", peer_address, peer_port))
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
		local peer_address = buffer(pos, 4):ipv4()
		t:add(peer_address_field, buffer(pos, 4)); pos = pos + 4

		local peer_port = buffer(pos, 2):uint()
		t:add(peer_port_field, buffer(pos, 2)); pos = pos + 2

		t:append_text(string.format(", Peer: %s:%d", peer_address, peer_port))
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

		local game_flags_tree = t:add(game_flags_field, buffer(pos, 1))
		game_flags_tree:add(mines_visible_field, buffer(pos, 1)); pos = pos + 1

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

function dissect_opcode_9c(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	if buffer_length >= 4 then
		t:add(unknown_field, buffer(pos, 4)); pos = pos + 4
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos, buffer_length)); pos = pos + buffer_length
	end

	return pos
end

function dissect_opcode_f0(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	if buffer_length >= 3 then
		t:add(unknown_field, buffer(pos, 3)); pos = pos + 3
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos)); pos = buffer_length
	end

	return pos
end

function dissect_opcode_f1(buffer, pinfo, tree)
	local pos = 0
	local subcode = buffer(pos + 1, 1):uint()
	if subcode == 0x01 then
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
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

		local game_flags_tree = t:add(game_flags_field, buffer(pos, 1));
		game_flags_tree:add(mines_visible_field, buffer(pos, 1)); pos = pos + 1

		t:add(allow_computer_field, buffer(pos, 1)); pos = pos + 1
		t:add(computer_advantage_field, buffer(pos, 1)); pos = pos + 1
		t:add_le(start_delay_field, buffer(pos, 4)); pos = pos + 4
		t:add_le(time_limit_field, buffer(pos, 4)); pos = pos + 4

		t:add(unknown_field, buffer(pos, 32)); pos = pos + 32
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	elseif subcode == 0x02 then
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local pillbox_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Pillbox Count: %d", pillbox_count))
		t:add(map_pillbox_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, pillbox_count - 1 do
			t:add(map_pillbox_data_field, buffer(pos, 5)); pos = pos + 5
		end
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	elseif subcode == 0x03 then
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local base_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Base Count: %d", base_count))
		t:add(map_base_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, base_count - 1 do
			t:add(map_base_data_field, buffer(pos, 6)); pos = pos + 6
		end
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	elseif subcode == 0x04 then
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
		t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

		local start_count = buffer(pos, 1):uint()
		t:append_text(string.format(", Map Start Count: %d", start_count))
		t:add(map_start_count_field, buffer(pos, 1)); pos = pos + 1
		for x = 0, start_count - 1 do
			t:add(map_start_data_field, buffer(pos, 3)); pos = pos + 3
		end
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	end

	return pos
end

function dissect_opcode_f2(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	if buffer_length >= 4 then
		t:add(unknown_field, buffer(pos, 4)); pos = pos + 4
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos)); pos = buffer_length
	end

	return pos
end

function dissect_opcode_f3(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	if buffer_length >= 3 then
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		return buffer_length
	end

	local length = buffer(pos, 1):uint() - 1
	t:add(unknown_field, buffer(pos, 1)); pos = pos + 1

	if length > buffer_length - pos then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
		return buffer_length
	end

	t:add(unknown_field, buffer(pos, length)); pos = pos + length

	if buffer_length - pos < 2 then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
		return buffer_length
	end

	t:add(unknown_field, buffer(pos, 2)); pos = pos + 2

	return pos
end

function dissect_player_name(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	local player_name, player_name_length = dissect_pascal_string(buffer(pos), t, player_name_field)

	if player_name ~= nil then
		t:append_text(string.format(", Player Name: %s", player_name))
		pos = pos + player_name_length + 1
	else
		-- t:add_proto_expert_info()
		t:add(unknown_field, buffer(pos))
		return buffer_length
	end

	return pos
end

function dissect_send_message(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return end

	local pos = 0
	local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

	if buffer_length >= 3 then
		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	else
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		return buffer_length
	end

	local message, message_length = dissect_pascal_string(buffer(pos), t, message_field)

	if message ~= nil then
		t:append_text(string.format(", Message: %s", message))
		pos = pos + message_length + 1
	else
		-- t:add_proto_expert_info()
		t:add(unknown_field, buffer(pos))
		return buffer_length
	end

	if buffer_length - pos < 2 then
		t:add_proto_expert_info(opcode_buffer_underrun_expert)
		t:add(unknown_field, buffer(pos))
		return buffer_length
	end

	t:add(unknown_field, buffer(pos, 2)); pos = pos + 2

	return pos
end

local opcode_dissectors =
{
	[0x9c] = dissect_opcode_9c,
	[0xf0] = dissect_opcode_f0,
	[0xf1] = dissect_opcode_f1,
	[0xf2] = dissect_opcode_f2,
	[0xf3] = dissect_opcode_f3,
	[0xf8] = dissect_player_name,
	[0xfa] = dissect_send_message
}

------ Block Dissector ------

function dissect_block(buffer, pinfo, tree)
	local buffer_length = buffer:len()
	if buffer_length < 1 then return 0 end

	local pos = 0
	while pos < buffer_length do
		local block_length = bit.band(buffer(pos, 1):uint(), 0x7f) + 1
		local block_end = pos + block_length
		local t = tree:add(block_field, buffer(pos, block_length + 1), block_length); pos = pos + 1

		if pos + block_length <= buffer_length then
			if block_length >= 3 then
				local sequence = buffer(pos, 1):uint()
				t:append_text(string.format(", Sequence: 0x%02x", sequence))
				t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

				t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
			end
		end

		local remaining = block_length - 3
		while remaining > 0 do
			local opcode = buffer(pos, 1):uint()

			local dissected = 0
			local opcode_dissector = opcode_dissectors[opcode]
			if opcode_dissector ~= nil then
				dissected = opcode_dissector(buffer(pos, remaining), pinfo, t)
			end

			if dissected == 0 then
				-- t:add_proto_expert_info(unknown_opcode_expert)
				t:add(unknown_field, buffer(pos, remaining))
				pos = block_end + 1
				remaining = 0
			else
				pos = pos + dissected
				remaining = remaining - dissected
			end
		end
	end

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
