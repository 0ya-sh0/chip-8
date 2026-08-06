This is a coding exercise to implement a interpreter for chip 8 VM.

https://en.wikipedia.org/wiki/CHIP-8
https://www.cs.columbia.edu/~sedwards/classes/2016/4840-spring/designs/Chip8.pdf

The test roms under /e2e are taken from https://github.com/Timendus/chip8-test-suite



The project structure will follow hexagonal architecture.
This means the core will be the VM implementation that manages
- memory
- registers
- stack
- handles instruction set

Input - its a outbound port!

Sound - its a outbound port!

Display - its a outbound port!

Timer - its a outbound port?
