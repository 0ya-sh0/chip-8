package asm

import (
	"fmt"
	"strconv"
	"strings"
)

// Assemble converts CHIP-8 assembly text into a executable binary ROM byte array.
type parsedLine struct {
	lineNum int
	raw     string
	tokens  []string
}

// Assemble converts CHIP-8 assembly text into an executable binary ROM byte array.
func Assemble(source string) ([]byte, error) {
	lines := strings.Split(source, "\n")

	// Pass 1: Preprocess lines, strip comments/prefixes, collect labels
	var parsedLines []parsedLine
	labels := make(map[string]uint16)
	currentAddr := uint16(0x0200) // CHIP-8 programs start at 0x0200

	for i, rawLine := range lines {
		lineNum := i + 1
		line := sanitizeLine(rawLine)
		if line == "" {
			continue
		}

		// Handle labels ending with ':' (e.g., "START:")
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			labelName := strings.TrimSuffix(line, ":")
			if _, exists := labels[labelName]; exists {
				return nil, fmt.Errorf("line %d: duplicate label %q", lineNum, labelName)
			}
			labels[labelName] = currentAddr
			continue
		}

		// Handle labels inline on the same line (e.g., "START: LD V0, 0x0A")
		if colonIdx := strings.Index(line, ":"); colonIdx != -1 {
			// Check if it's not an address prefix like "0x0200:"
			prefix := line[:colonIdx]
			if !strings.HasPrefix(prefix, "0x") && !strings.HasPrefix(prefix, "0X") {
				labels[prefix] = currentAddr
				line = strings.TrimSpace(line[colonIdx+1:])
				if line == "" {
					continue
				}
			}
		}

		tokens := tokenize(line)
		if len(tokens) == 0 {
			continue
		}

		parsedLines = append(parsedLines, parsedLine{
			lineNum: lineNum,
			raw:     line,
			tokens:  tokens,
		})

		// Track address offset: DATA ops can be 1 or 2 bytes, standard opcodes are 2 bytes
		if strings.ToUpper(tokens[0]) == "DATA" && len(tokens) > 1 {
			val, err := parseNumber(tokens[1])
			if err == nil && val <= 0xFF && len(tokens) == 2 {
				currentAddr += 1
			} else {
				currentAddr += 2
			}
		} else {
			currentAddr += 2
		}
	}

	// Pass 2: Assemble instructions into binary bytes
	var rom []byte
	for _, pl := range parsedLines {
		bytes, err := assembleInstruction(pl.tokens, labels)
		if err != nil {
			return nil, fmt.Errorf("line %d (%s): %w", pl.lineNum, pl.raw, err)
		}
		rom = append(rom, bytes...)
	}

	return rom, nil
}

// assembleInstruction encodes tokenized assembly mnemonics into opcode bytes.
func assembleInstruction(tokens []string, labels map[string]uint16) ([]byte, error) {
	cmd := strings.ToUpper(tokens[0])

	switch cmd {
	case "CLS":
		return []byte{0x00, 0xE0}, nil

	case "RET":
		return []byte{0x00, 0xEE}, nil

	case "SYS":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing address for SYS")
		}
		addr, err := resolveAddress(tokens[1], labels)
		if err != nil {
			return nil, err
		}
		return encodeNNN(0x0, addr), nil

	case "JP":
		if len(tokens) == 3 {
			// JP V0, addr
			if strings.ToUpper(tokens[1]) == "V0" {
				addr, err := resolveAddress(tokens[2], labels)
				if err != nil {
					return nil, err
				}
				return encodeNNN(0xB, addr), nil
			}
			return nil, fmt.Errorf("invalid register for JP, expected V0")
		}
		if len(tokens) == 2 {
			// JP addr
			addr, err := resolveAddress(tokens[1], labels)
			if err != nil {
				return nil, err
			}
			return encodeNNN(0x1, addr), nil
		}
		return nil, fmt.Errorf("invalid syntax for JP")

	case "CALL":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing address for CALL")
		}
		addr, err := resolveAddress(tokens[1], labels)
		if err != nil {
			return nil, err
		}
		return encodeNNN(0x2, addr), nil

	case "SE":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid syntax for SE")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		if isRegister(tokens[2]) {
			// SE Vx, Vy
			y, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0x50 | x, y << 4}, nil
		}
		// SE Vx, byte
		val, err := resolveByte(tokens[2], labels)
		if err != nil {
			return nil, err
		}
		return []byte{0x30 | x, val}, nil

	case "SNE":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid syntax for SNE")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		if isRegister(tokens[2]) {
			// SNE Vx, Vy
			y, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0x90 | x, y << 4}, nil
		}
		// SNE Vx, byte
		val, err := resolveByte(tokens[2], labels)
		if err != nil {
			return nil, err
		}
		return []byte{0x40 | x, val}, nil

	case "SKP":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing register for SKP")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		return []byte{0xE0 | x, 0x9E}, nil

	case "SKNP":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing register for SKNP")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		return []byte{0xE0 | x, 0xA1}, nil

	case "LD":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid syntax for LD")
		}
		dest := strings.ToUpper(tokens[1])
		src := strings.ToUpper(tokens[2])

		// LD I, addr
		if dest == "I" {
			addr, err := resolveAddress(tokens[2], labels)
			if err != nil {
				return nil, err
			}
			return encodeNNN(0xA, addr), nil
		}

		// LD DT, Vx
		if dest == "DT" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x15}, nil
		}

		// LD ST, Vx
		if dest == "ST" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x18}, nil
		}

		// LD F, Vx
		if dest == "F" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x29}, nil
		}

		// LD B, Vx
		if dest == "B" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x33}, nil
		}

		// LD [I], Vx
		if dest == "[I]" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x55}, nil
		}

		// Destination is Vx
		if isRegister(dest) {
			x, err := parseRegister(dest)
			if err != nil {
				return nil, err
			}

			if src == "DT" {
				return []byte{0xF0 | x, 0x07}, nil
			}
			if src == "K" {
				return []byte{0xF0 | x, 0x0A}, nil
			}
			if src == "[I]" {
				return []byte{0xF0 | x, 0x65}, nil
			}
			if isRegister(src) {
				y, err := parseRegister(src)
				if err != nil {
					return nil, err
				}
				return []byte{0x80 | x, y << 4}, nil
			}

			// LD Vx, byte
			val, err := resolveByte(tokens[2], labels)
			if err != nil {
				return nil, err
			}
			return []byte{0x60 | x, val}, nil
		}

		return nil, fmt.Errorf("invalid operands for LD")

	case "ADD":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid syntax for ADD")
		}
		dest := strings.ToUpper(tokens[1])

		// ADD I, Vx
		if dest == "I" {
			x, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0xF0 | x, 0x1E}, nil
		}

		// ADD Vx, ...
		x, err := parseRegister(dest)
		if err != nil {
			return nil, err
		}

		if isRegister(tokens[2]) {
			// ADD Vx, Vy
			y, err := parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
			return []byte{0x80 | x, (y << 4) | 0x04}, nil
		}

		// ADD Vx, byte
		val, err := resolveByte(tokens[2], labels)
		if err != nil {
			return nil, err
		}
		return []byte{0x70 | x, val}, nil

	case "OR":
		return encodeALU(0x1, tokens, labels)
	case "AND":
		return encodeALU(0x2, tokens, labels)
	case "XOR":
		return encodeALU(0x3, tokens, labels)
	case "SUB":
		return encodeALU(0x5, tokens, labels)
	case "SUBN":
		return encodeALU(0x7, tokens, labels)

	case "SHR":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing register for SHR")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		y := byte(0)
		if len(tokens) >= 3 && isRegister(tokens[2]) {
			y, err = parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
		}
		return []byte{0x80 | x, (y << 4) | 0x06}, nil

	case "SHL":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing register for SHL")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		y := byte(0)
		if len(tokens) >= 3 && isRegister(tokens[2]) {
			y, err = parseRegister(tokens[2])
			if err != nil {
				return nil, err
			}
		}
		return []byte{0x80 | x, (y << 4) | 0x0E}, nil

	case "RND":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid syntax for RND")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		val, err := resolveByte(tokens[2], labels)
		if err != nil {
			return nil, err
		}
		return []byte{0xC0 | x, val}, nil

	case "DRW":
		if len(tokens) < 4 {
			return nil, fmt.Errorf("invalid syntax for DRW")
		}
		x, err := parseRegister(tokens[1])
		if err != nil {
			return nil, err
		}
		y, err := parseRegister(tokens[2])
		if err != nil {
			return nil, err
		}
		n, err := parseNumber(tokens[3])
		if err != nil || n > 0xF {
			return nil, fmt.Errorf("invalid nibble for DRW: %s", tokens[3])
		}
		return []byte{0xD0 | x, (y << 4) | byte(n)}, nil

	case "DATA":
		if len(tokens) < 2 {
			return nil, fmt.Errorf("missing data value")
		}
		val, err := parseNumber(tokens[1])
		if err != nil {
			return nil, fmt.Errorf("invalid data value: %s", tokens[1])
		}
		if val <= 0xFF && len(tokens) == 2 {
			return []byte{byte(val)}, nil
		}
		return []byte{byte(val >> 8), byte(val & 0xFF)}, nil

	default:
		return nil, fmt.Errorf("unknown opcode or mnemonic %q", cmd)
	}
}

// --- Helper Functions ---

func encodeNNN(prefix byte, addr uint16) []byte {
	return []byte{
		(prefix << 4) | byte((addr>>8)&0x0F),
		byte(addr & 0xFF),
	}
}

func encodeALU(subCode byte, tokens []string, labels map[string]uint16) ([]byte, error) {
	if len(tokens) < 3 {
		return nil, fmt.Errorf("invalid syntax for %s", tokens[0])
	}
	x, err := parseRegister(tokens[1])
	if err != nil {
		return nil, err
	}
	y, err := parseRegister(tokens[2])
	if err != nil {
		return nil, err
	}
	return []byte{0x80 | x, (y << 4) | subCode}, nil
}

func isRegister(s string) bool {
	s = strings.ToUpper(s)
	if len(s) != 2 || s[0] != 'V' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 8)
	return err == nil
}

func parseRegister(s string) (byte, error) {
	s = strings.ToUpper(s)
	if !isRegister(s) {
		return 0, fmt.Errorf("invalid register %q", s)
	}
	val, _ := strconv.ParseUint(s[1:], 16, 8)
	return byte(val), nil
}

func parseNumber(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err := strconv.ParseUint(s[2:], 16, 16)
		return uint16(val), err
	}
	val, err := strconv.ParseUint(s, 10, 16)
	return uint16(val), err
}

func resolveAddress(s string, labels map[string]uint16) (uint16, error) {
	if addr, exists := labels[s]; exists {
		return addr, nil
	}
	val, err := parseNumber(s)
	if err != nil {
		return 0, fmt.Errorf("unknown label or invalid address %q", s)
	}
	return val, nil
}

func resolveByte(s string, labels map[string]uint16) (byte, error) {
	val, err := resolveAddress(s, labels)
	if err != nil {
		return 0, err
	}
	if val > 0xFF {
		return 0, fmt.Errorf("value 0x%04X exceeds 8-bit byte limit", val)
	}
	return byte(val), nil
}

func sanitizeLine(line string) string {
	// Strip comments starting with ';' or '//'
	if idx := strings.Index(line, ";"); idx != -1 {
		line = line[:idx]
	}
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	line = strings.TrimSpace(line)

	// If line begins with disassembler address & hex output (e.g. "0x0200:  00E0  CLS"),
	// strip "0x0200:  00E0" prefix so disassembled files re-assemble directly!
	if len(line) >= 15 && strings.HasPrefix(line, "0x") && line[6] == ':' {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			line = strings.Join(parts[2:], " ")
		}
	}

	return line
}

func tokenize(line string) []string {
	// Replace commas and tabs with spaces for uniform tokenization
	line = strings.ReplaceAll(line, ",", " ")
	line = strings.ReplaceAll(line, "\t", " ")
	fields := strings.Fields(line)
	var tokens []string
	for _, f := range fields {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			tokens = append(tokens, trimmed)
		}
	}
	return tokens
}
