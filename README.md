# CHIP-8 Virtual Machine & Interpreter

A Go-based emulator and interpreter for the **CHIP-8** virtual machine, built as a modern coding exercise focusing on system emulation, bitwise instruction decoding, and software design principles.

---

## 🏛️ Architecture: Hexagonal (Ports & Adapters)

This project follows **Hexagonal Architecture** to maintain a pure, decoupled core engine that is completely agnostic to terminal, GUI, or audio drivers.


```
              +-----------------------------------+
              |            CHIP-8 VM              |
              |             (Core)                |
              |                                   |
              |  - 4KB Memory      - Stack        |
              |  - Registers (V0-VF) - Timers     |
              |  - Opcode Decoder  - RNG          |
              +-----------------------------------+
                 /              |              \
  [Outbound Port]        [Outbound Port]       [Outbound Port]
         |                      |                     |
         v                      v                     v
 DisplayProvider        KeyboardProvider        SoundProvider
         |                      |                     |
 (Terminal Display)    (Terminal Keyboard)     (Terminal Sound)

```


* **Core Engine (`internal/vm`):** Encapsulates the execution loop, instruction set decoding, memory layout, register management, execution stack, subroutines, delay/sound timers, and random number generation.
* **Outbound Ports & Adapters:**
  * **Display:** Renders framebuffer output (`DisplayProvider`).
  * **Input:** Handles non-blocking 16-key keypad input (`KeyboardProvider`).
  * **Sound:** Signals 60 Hz tone execution (`SoundProvider`).

---

## 🕹️ Controls (QWERTY Mapping)

The original COSMAC VIP 16-key hex keypad is mapped to the standard QWERTY grid:

| CHIP-8 Hex Keypad | QWERTY Key Mapping |
| :---: | :---: |
| `1` `2` `3` `C` | `1` `2` `3` `4` |
| `4` `5` `6` `D` | `Q` `W` `E` `R` |
| `7` `8` `9` `E` | `A` `S` `D` `F` |
| `A` `0` `B` `F` | `Z` `X` `C` `V` |

---

## 🚀 Getting Started

### Prerequisites
* **Go 1.20+** installed on your system.
* A POSIX-compliant terminal (Linux / macOS) for raw terminal mode handling.

### Running a ROM
To load and launch a `.ch8` binary:

```bash
go run cmd/main.go path/to/rom.ch8

```

---

## 🧪 Testing & Acknowledgments

* **Test Suite:** The end-to-end tests located under `/e2e` use [Timendus's CHIP-8 Test Suite](https://github.com/Timendus/chip8-test-suite) for verifying opcode logic, flags, and memory operations.
* **Additional Games & ROMs:** Public domain games and test binaries can be found at [Matt Mikolay's CHIP-8 Repository](https://github.com/mattmikolay/chip-8).

### References & Documentation

* [Wikipedia — CHIP-8 Overview](https://en.wikipedia.org/wiki/CHIP-8)
* [Columbia University — CHIP-8 Architecture & Opcode Specification](https://www.google.com/search?q=https://www.cs.columbia.edu/~sedwares/classes/2016/4840-spring/designs/Chip8.pdf)
