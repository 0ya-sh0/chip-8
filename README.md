# CHIP-8 Virtual Machine & Interpreter

A Go-based emulator and interpreter for the **CHIP-8** virtual machine, built as a modern coding exercise focusing on system emulation, bitwise instruction decoding, and software design principles.

---

## 🏛️ Architecture: Hexagonal (Ports & Adapters)

This project strictly adheres to **Hexagonal Architecture**. The core execution engine is decoupled from I/O mechanisms, allowing you to run games either directly inside a terminal via ASCII art or in a 2D window powered by Ebitengine with PCM synthesized audio.


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
       /       \              /       \              /       \
      /         \            /         \            /         \
(Terminal)      (Ebit) (Terminal)      (Ebit) (Terminal)      (Ebit)

```

* **Core Engine (`internal/vm`):** Encapsulates opcode decoding, memory layout, registers, stack execution, subroutines, delay/sound timers, and random number generation.
* **Outbound Adapters:**
  * **Terminal Adapter (`internal/terminal`):** Renders framebuffer to stdout, uses raw terminal keyboard inputs, and triggers terminal alert tones.
  * **Ebitengine Adapter (`internal/ebit`):** Provides pixel-scaled GUI rendering, window keyboard mapping, and a 440 Hz square wave audio oscillator.

---

## 📁 Project Structure


```

.
├── cmd
│   ├── cli          # CLI Terminal runner
│   │   └── main.go
│   └── ebit         # Ebitengine GUI runner
│       └── main.go
├── internal
│   ├── ebit         # Ebitengine GUI adapters (Display, Keyboard, Sound)
│   │   └── adapter.go
│   ├── terminal     # Raw terminal adapters (Display, Keyboard, Sound)
│   │   ├── display.go
│   │   ├── keyboard.go
│   │   └── sound.go
│   └── vm           # Core CHIP-8 VM engine & opcode execution
│       ├── display_logger.go
│       ├── opcode.go
│       ├── opcode_test.go
│       ├── provider.go
│       └── vm.go
├── test-games       # Public domain games & demo ROMs
└── test-roms        # Timendus CHIP-8 test suite ROMs
```

---

## 🕹️ Controls (QWERTY Mapping)

The original COSMAC VIP 16-key hex keypad is mapped to a standard $4 \times 4$ QWERTY grid:

| CHIP-8 Hex Keypad | QWERTY Key Mapping |
| :---: | :---: |
| `1` `2` `3` `C` | `1` `2` `3` `4` |
| `4` `5` `6` `D` | `Q` `W` `E` `R` |
| `7` `8` `9` `E` | `A` `S` `D` `F` |
| `A` `0` `B` `F` | `Z` `X` `C` `V` |

---

## 🚀 Getting Started

### Prerequisites

* **Go 1.20+** installed on your machine.
* **Linux Cgo Dependencies (Ebitengine GUI):** On Linux, Ebitengine requires standard X11 and audio development headers. Install them using your package manager:
```bash
# Ubuntu / Debian
sudo apt install -y libx11-dev \
        libxrandr-dev libxcursor-dev \
        libxinerama-dev libxi-dev \
        libxxf86vm-dev libgl1-mesa-dev \
        libasound2-dev
```

---

### Running a ROM

You can run any `.ch8` file in either **GUI mode** or **Terminal CLI mode**.

#### 1. Ebitengine GUI Mode (Recommended)

```bash
go run cmd/ebit/main.go test-games/cavern.ch8
```

#### 2. Terminal CLI Mode

```bash
go run cmd/cli/main.go test-games/cavern.ch8
```

---

## 🧪 Testing & Acknowledgments

* **Test Suite (`/test-roms`):** Uses [Timendus's CHIP-8 Test Suite](https://github.com/Timendus/chip8-test-suite) for verifying opcode execution, memory boundaries, keypad state, and flags.
* **Games (`/test-games`):** Public domain games and demo binaries sourced from [Matt Mikolay's CHIP-8 Repository](https://github.com/mattmikolay/chip-8).

### References & Documentation

* [Wikipedia — CHIP-8 Overview](https://en.wikipedia.org/wiki/CHIP-8)
* [Columbia University — CHIP-8 Architecture & Opcode Specification](https://www.cs.columbia.edu/~sedwards/classes/2016/4840-spring/designs/Chip8.pdf)
