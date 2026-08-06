package vm

import (
	"fmt"
	"os"
	"testing"
)

func Test_parseOpcode(t *testing.T) {
	data, _ := os.ReadFile("/home/yash/coding/chip-8/e2e/1-chip8-logo.ch8")
	for i := 0; i < len(data); i += 2 {
		var opcode [2]uint8
		opcode[0] = data[i]
		opcode[1] = data[i+1]
		result := parseOpcode(opcode)
		fmt.Printf("%X == %+v\n", opcode, result)
	}
}
