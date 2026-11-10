package configfile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Set rewrites the values of dotted keys in a TOML document. Everything it does not have to
// touch stays byte for byte identical: comments, blank lines, ordering and indentation survive,
// which matters because the config is hand maintained.
func Set(data []byte, changes map[string]string) ([]byte, error) {
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("config file is empty")
	}

	hadTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := changes[key]
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("value for %q must be a single line", key)
		}

		section, name := splitKey(key)
		var err error
		lines, err = setValue(lines, section, name, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
	}

	out := strings.Join(lines, "\n")
	if hadTrailingNewline {
		out += "\n"
	}

	// Prove the result parses before the caller drops it over the user's file. A rewrite that
	// only looks like TOML would leave the config unusable with nothing to fall back to.
	var probe map[string]any
	if _, err := toml.Decode(out, &probe); err != nil {
		return nil, fmt.Errorf("refusing to write config that does not parse: %w", err)
	}
	return []byte(out), nil
}

func splitKey(key string) (section, name string) {
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		return key[:idx], key[idx+1:]
	}
	return "", key
}

func setValue(lines []string, section, name, value string) ([]string, error) {
	target := joinKey(section, name)
	table := ""
	inArrayTable := false

	for i, line := range lines {
		if header, ok := tableHeader(line); ok {
			table = header
			inArrayTable = false
			continue
		}
		if isArrayTableHeader(line) {
			inArrayTable = true
			continue
		}
		if inArrayTable {
			continue
		}

		// Compare full paths so that a dotted key such as `chezmoi.enable = false` written at
		// the top level is rewritten in place instead of growing a second [chezmoi] table.
		key, start, end, ok := keyValue(line)
		if !ok || joinKey(table, key) != target {
			continue
		}
		if start < len(line) && (line[start] == '[' || line[start] == '{') {
			return nil, fmt.Errorf("%s holds a list or inline table, edit the file by hand", target)
		}

		token, err := formatValue(value, line[start:end])
		if err != nil {
			return nil, err
		}
		lines[i] = line[:start] + token + line[end:]
		return lines, nil
	}

	return insertValue(lines, section, name, value)
}

func joinKey(table, key string) string {
	if table == "" {
		return key
	}
	return table + "." + key
}

func insertValue(lines []string, section, name, value string) ([]string, error) {
	token, err := formatValue(value, "")
	if err != nil {
		return nil, err
	}
	entry := name + " = " + token

	if section == "" {
		return insertAt(lines, 0, entry), nil
	}

	headerIndex := -1
	for i, line := range lines {
		if header, ok := tableHeader(line); ok && header == section {
			headerIndex = i
			break
		}
	}

	if headerIndex < 0 {
		added := []string{"[" + section + "]", entry}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			added = append([]string{""}, added...)
		}
		return append(lines, added...), nil
	}

	end := len(lines)
	for i := headerIndex + 1; i < len(lines); i++ {
		if _, ok := tableHeader(lines[i]); ok {
			end = i
			break
		}
	}

	at := end
	for at > headerIndex+1 && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	return insertAt(lines, at, entry), nil
}

func insertAt(lines []string, index int, entry string) []string {
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:index]...)
	out = append(out, entry)
	return append(out, lines[index:]...)
}

// tableHeader reports the table path of a "[table]" line. Array tables are ignored because
// matheme never writes into them.
func tableHeader(line string) (string, bool) {
	trimmed := strings.TrimSpace(stripComment(line))
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "[[") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}
	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	inner = strings.Trim(inner, `"'`)
	if inner == "" {
		return "", false
	}
	return inner, true
}

// isArrayTableHeader reports a "[[table]]" line, whose keys belong to one element of a list.
func isArrayTableHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(stripComment(line)), "[[")
}

// keyValue returns the bare key of a "key = value" line plus the bounds of the value token.
func keyValue(line string) (key string, start, end int, ok bool) {
	eq := equalsIndex(line)
	if eq < 0 {
		return "", 0, 0, false
	}

	rawKey := strings.TrimSpace(stripComment(line[:eq]))
	rawKey = strings.Trim(rawKey, `"'`)
	if rawKey == "" {
		return "", 0, 0, false
	}

	start, end = valueBounds(line, eq+1)
	return rawKey, start, end, true
}

func equalsIndex(line string) int {
	inDouble, inSingle := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case inDouble:
			if c == '\\' {
				i++
			} else if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '#':
			return -1
		case c == '=':
			return i
		}
	}
	return -1
}

func valueBounds(line string, from int) (start, end int) {
	i := from
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	start = i
	if i >= len(line) {
		return start, start
	}

	switch line[i] {
	case '"', '\'':
		quote := line[i]
		i++
		for i < len(line) {
			if quote == '"' && line[i] == '\\' {
				i += 2
				continue
			}
			if line[i] == quote {
				return start, i + 1
			}
			i++
		}
		return start, len(line)
	default:
		for i < len(line) && line[i] != '#' {
			i++
		}
		for i > start && (line[i-1] == ' ' || line[i-1] == '\t') {
			i--
		}
		return start, i
	}
}

// stripComment cuts a line at the first # that is not inside a string.
func stripComment(line string) string {
	inDouble, inSingle := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case inDouble:
			if c == '\\' {
				i++
			} else if c == '"' {
				inDouble = false
			}
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case c == '"':
			inDouble = true
		case c == '\'':
			inSingle = true
		case c == '#':
			return line[:i]
		}
	}
	return line
}

// formatValue keeps the type of the value it replaces so that toggles stay booleans and paths
// stay strings.
func formatValue(value, existing string) (string, error) {
	existing = strings.TrimSpace(existing)

	switch {
	case strings.HasPrefix(existing, `"`), strings.HasPrefix(existing, `'`):
		return quote(value)
	case existing == "true" || existing == "false":
		if value != "true" && value != "false" {
			return "", fmt.Errorf("expected true or false, got %q", value)
		}
		return value, nil
	case existing == "":
		if value == "true" || value == "false" {
			return value, nil
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil && value != "" {
			return value, nil
		}
		return quote(value)
	}

	if _, err := strconv.ParseFloat(existing, 64); err == nil {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return "", fmt.Errorf("expected a number, got %q", value)
		}
		return value, nil
	}

	return quote(value)
}

func quote(value string) (string, error) {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("value contains a control character")
		}
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`, nil
}
