package vm

import (
	"fmt"
)

type LoggerDisplay struct {
}

// Close implements [vm.DisplayProvider].
func (l *LoggerDisplay) Close() {

}

// Draw implements [vm.DisplayProvider].
func (l *LoggerDisplay) Draw(data FrameBuffer) {
	fmt.Printf("Display: Draw\n")
}

// Clear implements [vm.DisplayProvider].
func (l *LoggerDisplay) Clear() {
	fmt.Printf("Display: Clear\n")
}

var _ (DisplayProvider) = (*LoggerDisplay)(nil)
