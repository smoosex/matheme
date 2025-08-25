local file_path = arg[1]
local theme_name = arg[2]
local home_dir = os.getenv("HOME") or os.getenv("USERPROFILE")
local info = dofile(home_dir .. "/.config/matheme/themes/" .. theme_name .. ".lua")
local base_16 = info.base_16

local new_active_color = string.format("0xff%s", string.sub(base_16["base08"], 2))

local lines = {}
for line in io.lines(file_path) do
	if line:match("^%s*active_color%s*=") then
		local indent = line:match("^(%s*)active_color")
		line = string.format("%sactive_color=%s", indent or "", new_active_color)
	end
	table.insert(lines, line)
end

local fp, err = io.open(file_path, "w")
if not fp then
	error("failed to open file: " .. err)
end
for _, l in ipairs(lines) do
	fp:write(l, "\n")
end
fp:close()
