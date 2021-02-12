bolo_protocol = Proto("Bolo",  "Bolo Protocol")

local boolean_values =
{
	[0] = "False",
	[1] = "True"
}

unknown_field = ProtoField.bytes("bolo.unknown", "unknown", base.SPACE)
padding_field = ProtoField.bytes("bolo.padding", "padding", base.SPACE)

signature_field = ProtoField.string("bolo.signature", "signature", base.ASCII)
version_field = ProtoField.bytes("bolo.version", "version")
packet_type_field = ProtoField.uint8("bolo.packet_type", "packet_type", base.HEX)

-- Packet Type 0x02
sequence_field = ProtoField.uint8("bolo.sequence", "sequence", base.HEX)
opcode_field = ProtoField.uint8("bolo.opcode", "opcode", base.HEX)
subcode_field = ProtoField.uint8("bolo.subcode", "subcode", base.HEX)

host_address_field = ProtoField.ipv4("bolo.host_address", "host_address")

-- Op Code 0xfa
message_length_field = ProtoField.uint8("bolo.message_length", "message_length", base.DEC)
message_field = ProtoField.string("bolo.message", "message", base.ASCII)

map_pillbox_count_field = ProtoField.uint8("bolo.map_pillbox_count", "map_pillbox_count", base.DEC)
map_pillbox_data_field = ProtoField.bytes("bolo.map_pillbox_data", "map_pillbox_data", base.SPACE)
map_base_count_field = ProtoField.uint8("bolo.map_base_count", "map_base_count", base.DEC)
map_base_data_field = ProtoField.bytes("bolo.map_base_data", "map_base_data", base.SPACE)
map_start_count_field = ProtoField.uint8("bolo.map_start_count", "map_start_count", base.DEC)
map_start_data_field = ProtoField.bytes("bolo.map_start_data", "map_start_data", base.SPACE)

user_name_length_field = ProtoField.uint8("bolo.user_name_length", "user_name_length", base.DEC)
user_name_field = ProtoField.string("bolo.user_name", "user_name", base.ASCII)

map_name_length_field = ProtoField.uint8("bolo.map_name_length", "map_name_length", base.DEC)
map_name_field = ProtoField.string("bolo.map_name", "map_name", base.ASCII)

start_time_field = ProtoField.uint32("bolo.start_time", "start_time", base.HEX)

local game_type_values =
{
	[1] = "Open Game",
	[2] = "Tournament",
	[3] = "Strict Tournament"
}
game_type_field = ProtoField.uint8("bolo.game_type", "game_type", base.HEX, game_type_values)

game_flags_field = ProtoField.uint8("bolo.game_flags", "game_flags", base.HEX)
mines_visible_field = ProtoField.uint8("bolo.mines_visible", "mines_visible", base.HEX, boolean_values, 0x40)
allow_computer_field = ProtoField.bool("bolo.allow_computer", "allow_computer")
computer_advantage_field = ProtoField.bool("bolo.computer_advantage", "computer_advantage")
start_delay_field = ProtoField.uint32("bolo.start_delay", "start_delay", base.UNIT_STRING, {" Second", " Seconds"})
time_limit_field = ProtoField.uint32("bolo.time_limit", "time_limit", base.UNIT_STRING, {" Minute", " Minutes"})

-- Packet Type 0x07
peer_address_field = ProtoField.ipv4("bolo.peer_address", "peer_address")
peer_port_field = ProtoField.uint16("bolo.peer_port", "peer_port")

-- Packet Type 0x08 Password
password_length_field = ProtoField.uint8("bolo.password_length", "password_length", base.DEC)
password_field = ProtoField.string("bolo.password", "password", base.ASCII)

-- Packet Type 0x0E
num_players_field = ProtoField.uint16("bolo.num_players", "num_players", base.DEC)
free_pills_field = ProtoField.uint16("bolo.free_pills", "free_pills", base.DEC)
free_bases_field = ProtoField.uint16("bolo.free_bases", "free_bases", base.DEC)
has_password_field = ProtoField.bool("bolo.has_password", "has_password")

bolo_protocol.fields = {
	unknown_field, padding_field,

	signature_field, version_field, packet_type_field,

	-- Packet Type 0x02 Game State
	sequence_field, opcode_field, subcode_field,
	host_address_field,
	message_length_field, message_field,
	map_pillbox_count_field, map_pillbox_data_field,
	map_base_count_field, map_base_data_field,
	map_start_count_field, map_start_data_field,

	user_name_length_field, user_name_field,
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

unknown_opcode_expert = ProtoExpert.new("bolo.unknown_opcode.expert", "Unknown opcode", expert.group.UNDECODED, expert.severity.WARN)
bolo_protocol.experts = { unknown_opcode_expert }

local operand_count =
{
	[0x04] = 5,
	[0x06] = 7,
	[0x21] = 3,
	[0x24] = 3,
	[0x25] = 3,
	[0x27] = 3,
	[0x2b] = 3,
	[0x5e] = 3,
	[0x81] = 1,
	[0x82] = 3,
	[0x84] = 5,
	[0x85] = 1,
	[0x86] = 7,
	[0x87] = 8,
	[0x89] = 10,
	[0x8c] = 13,
	[0x93] = 3,
	[0x94] = 3,
	[0x95] = 3,
	[0x99] = 4,
	[0xaf] = 3,
	[0xb7] = 3,
	[0xc9] = 3,
	[0xd7] = 3,
	[0xde] = 3,
	[0xf8] = 36, -- user name
	[0xfa] = 38 -- message
}

function bolo_protocol.dissector(buffer, pinfo, tree)
	length = buffer:len()
	if length == 0 then return end

	pinfo.cols.protocol = bolo_protocol.name

	local t = tree:add(bolo_protocol, buffer(), "Bolo Protocol Header")

	t:add(signature_field, buffer(0, 4))

	version = string.format(" (%x.%x.%x)", buffer(4, 1):uint(), buffer(5, 1):uint(), buffer(6, 1):uint())
	t:add(version_field, buffer(4, 3)):append_text(version)

	packet_type = buffer(7, 1):uint()
	packet_type_name = get_packet_type_name(packet_type)
	t:add(packet_type_field, buffer(7, 1)):append_text(" (" .. packet_type_name .. ")")

	if packet_type == 0x00 then
		local t0 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x00")
		local pos = 8

		t0:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x01 then
		local t1 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x01")
		local pos = 8

		t1:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x02 then -- Game State
		dissect_game_state(buffer(8), tree)
	end

	if packet_type == 0x03 then
		local t3 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x03")
		local pos = 8

		t3:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x04 then -- acknowledge 0x02
		local t4 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x04")
		local pos = 8

		t4:add(sequence_field, buffer(pos, 1)); pos = pos + 1
	end

	if packet_type == 0x05 then -- Request 0x07
		local t5 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x05")
		local pos = 8

		t5:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x06 then
		local t6 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x06")
		local pos = 8

		t6:add(unknown_field, buffer(pos, 4)); pos = pos + 4

		t6:add(peer_address_field, buffer(pos, 4)); pos = pos + 4
		t6:add(peer_port_field, buffer(pos, 2)); pos = pos + 2

		t6:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x07 then -- Reply to 0x05
		local t7 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x07")

		t7:add(unknown_field, buffer(8, 4))

		t7:add(peer_address_field, buffer(12, 4))
		t7:add(peer_port_field, buffer(16, 2))

		t7:add(unknown_field, buffer(18, 4))
	end

	if packet_type == 0x08 then
		local t8 = tree:add(bolo_protocol, buffer(8), "Bolo Password")
		local pos = 8

		local password_length = buffer(pos, 1):uint()
		t8:add(password_length_field, buffer(pos, 1)); pos = pos + 1

		local password_end = pos + password_length
		local padding_length = 35 - password_length
		t8:add(password_field, buffer(pos, password_length))
		if padding_length > 0 then t8:add(padding_field, buffer(password_end, padding_length)) end
		pos = pos + 35
	end

	if packet_type == 0x09 then
		local t9 = tree:add(bolo_protocol, buffer(8), "Bolo Packet Type 0x09")
		local pos = 8

		t9:add(peer_address_field, buffer(pos, 4)); pos = pos + 4
		t9:add(peer_port_field, buffer(pos, 2)); pos = pos + 2
	end

	if packet_type == 0x0d then
		local td = tree:add(bolo_protocol, buffer(8), "Bolo Request Game Info")
		local pos = 8

		td:add(unknown_field, buffer(pos))
	end

	if packet_type == 0x0e then
		local te = tree:add(bolo_protocol, buffer(8), "Bolo Game Info")
		local pos = 8

		local map_name_length = buffer(pos, 1):uint()
		te:add(map_name_length_field, buffer(pos, 1)); pos = pos + 1

		local map_name_end = pos + map_name_length
		local padding_length = 35 - map_name_length
		te:add(map_name_field, buffer(pos, map_name_length))
		if padding_length > 0 then te:add(padding_field, buffer(map_name_end, padding_length)) end
		pos = pos + 35

		te:add(peer_address_field, buffer(pos, 4)); pos = pos + 4
		te:add_le(start_time_field, buffer(pos, 4)); pos = pos + 4

		te:add(game_type_field, buffer(pos, 1)); pos = pos + 1

		local game_flags_tree = te:add(game_flags_field, buffer(pos, 1))
		game_flags_tree:add(mines_visible_field, buffer(pos, 1)); pos = pos + 1

		te:add(allow_computer_field, buffer(pos, 1)); pos = pos + 1
		te:add(computer_advantage_field, buffer(pos, 1)); pos = pos + 1

		local start_delay = buffer(pos, 4):le_uint()
		if start_delay ~= 0 then start_delay = (start_delay / 50) + 1 end
		te:add(start_delay_field, buffer(pos, 4), start_delay); pos = pos + 4

		local time_limit = buffer(pos, 4):le_uint()
		if time_limit ~= 0 then time_limit = (time_limit / 50 / 60) + 1 end
		te:add(time_limit_field, buffer(pos, 4), time_limit); pos = pos + 4

		te:add_le(num_players_field, buffer(pos, 2)); pos = pos + 2
		te:add_le(free_pills_field, buffer(pos, 2)); pos = pos + 2
		te:add_le(free_bases_field, buffer(pos, 2)); pos = pos + 2

		te:add(has_password_field, buffer(pos, 1)); pos = pos + 1

		te:add(unknown_field, buffer(pos))
	end
end

function get_packet_type_name(packet_type)
	local packet_type_name = "UNKNOWN"

	if packet_type == 0x02 then packet_type_name = "Game State"
	elseif packet_type == 0x08 then packet_type_name = "Password"
	elseif packet_type == 0x0d then packet_type_name = "Request Game Info"
	elseif packet_type == 0x0e then packet_type_name = "Game Info" end

	return packet_type_name
end

function dissect_game_state(buffer, tree)
	local pos = 0
	local t = tree:add(bolo_protocol, buffer(pos), "Bolo Game State")

	t:add(sequence_field, buffer(pos, 1)); pos = pos + 1

	while pos < buffer:len() do
		local opcode = buffer(pos, 1):uint()
		local dissected = dissect_opcode(opcode, buffer(pos), t)
		pos = pos + dissected
		if (dissected == 0) then
			local count = operand_count[opcode]
			if count == nil then
				t:add_proto_expert_info(unknown_opcode_expert)
				t:add(unknown_field, buffer(pos))
				return
			end
			local opcode_tree = t:add(opcode_field, buffer(pos, 1)); pos = pos + 1
			opcode_tree:add(unknown_field, buffer(pos, count)); pos = pos + count
		end
	end
end

function dissect_opcode(opcode, buffer, tree)
	local pos = 0

	if opcode == 0xf0 then
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
		t:add(unknown_field, buffer(pos, 1)); pos = pos + 1

		for x = 0, 2 do
			t:add(peer_address_field, buffer(pos, 4)); pos = pos + 4
			t:add(peer_port_field, buffer(pos, 2)); pos = pos + 2
		end

		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	elseif opcode == 0xf1 then
		local subcode = buffer(pos + 1, 1):uint()
		if subcode == 0x01 then
			local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
			t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

			local map_name_length = buffer(pos, 1):uint()
			t:add(map_name_length_field, buffer(pos, 1)); pos = pos + 1

			local map_name_end = pos + map_name_length
			local padding_length = 35 - map_name_length
			t:add(map_name_field, buffer(pos, map_name_length))
			if padding_length > 0 then t:add(padding_field, buffer(map_name_end, padding_length)) end
			pos = pos + 35

			t:add(host_address_field, buffer(pos, 4)); pos = pos + 4

			t:add_le(start_time_field, buffer(pos, 4)); pos = pos + 4
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
			t:add(map_pillbox_count_field, buffer(pos, 1)); pos = pos + 1
			for x = 0, pillbox_count - 1 do
				t:add(map_pillbox_data_field, buffer(pos, 5)); pos = pos + 5
			end
			t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
		elseif subcode == 0x03 then
			local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
			t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

			local base_count = buffer(pos, 1):uint()
			t:add(map_base_count_field, buffer(pos, 1)); pos = pos + 1
			for x = 0, base_count - 1 do
				t:add(map_base_data_field, buffer(pos, 6)); pos = pos + 6
			end
			t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
		elseif subcode == 0x04 then
			local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1
			t:add(subcode_field, buffer(pos, 1)); pos = pos + 1

			local start_count = buffer(pos, 1):uint()
			t:add(map_start_count_field, buffer(pos, 1)); pos = pos + 1
			for x = 0, start_count - 1 do
				t:add(map_start_data_field, buffer(pos, 3)); pos = pos + 3
			end
			t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
		end
	elseif opcode == 0xf8 then -- user name
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

		user_name_length = buffer(pos, 1):uint()
		t:add(user_name_length_field, buffer(pos, 1)); pos = pos + 1
		t:add(user_name_field, buffer(pos, user_name_length)); pos = pos + user_name_length

		t:add(unknown_field, buffer(pos, 4)); pos = pos + 4
	elseif opcode == 0xfa then -- message
		local t = tree:add(opcode_field, buffer(pos, 1)); pos = pos + 1

		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2

		message_length = buffer(pos, 1):uint()
		t:add(message_length_field, buffer(pos, 1)); pos = pos + 1
		t:add(message_field, buffer(pos, message_length)); pos = pos + message_length

		t:add(unknown_field, buffer(pos, 2)); pos = pos + 2
	end

	return pos
end

DissectorTable.get("udp.port"):add(50000, bolo_protocol)
