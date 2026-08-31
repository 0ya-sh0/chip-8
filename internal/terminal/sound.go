package terminal

import "fmt"

type TerminalSound struct{}

func NewTerminalSound() *TerminalSound {
	return &TerminalSound{}
}

func (s *TerminalSound) PlaySound() {
	// Prints the ASCII 'BEL' character (0x07)
	// Most terminals will emit a "ding" or system chime
	fmt.Print("\a")
}

func (s *TerminalSound) StopSound() {
	// No-op for terminal bell; the OS chime stops automatically
}
