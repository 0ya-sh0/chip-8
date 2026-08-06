package display

import (
	"fmt"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

type LoggerDisplay struct {
}

// Draw implements [vm.DisplayProvider].
func (l *LoggerDisplay) Draw(data [64][32]bool) {
	fmt.Printf("Display: Draw\n")
}

// Clear implements [vm.DisplayProvider].
func (l *LoggerDisplay) Clear() {
	fmt.Printf("Display: Clear\n")
}

var _ (vm.DisplayProvider) = (*LoggerDisplay)(nil)
