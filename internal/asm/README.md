# `internal/asm` — CHIP-8 Assembler & Disassembler

The `internal/asm` package provides bi-directional transformation between raw binary CHIP-8 ROM files (`.ch8`) and human-readable CHIP-8 assembly code (`.asm`).

```
          +-----------------------+
          |  Assembly Source      |
          |  (my_game.asm)        |
          +-----------------------+
             /                 ^
            /                   \
   Assemble()                  Disassemble()
          /                       \
         v                         \
  +-----------------------------------+
  |  Binary CHIP-8 ROM (.ch8)         |
  +-----------------------------------+

```

---

## 📝 Syntax & Conventions

### 1. Registers & Special Symbols

| Symbol | Description | Example Usage |
| --- | --- | --- |
| `V0` – `VF` | 16 General-purpose 8-bit registers | `LD V0, 0x0A` |
| `I` | 16-bit Index / Memory address register | `LD I, 0x0200` |
| `DT` | Delay Timer register | `LD DT, V0` |
| `ST` | Sound Timer register | `LD ST, V1` |
| `K` | Blocking Keypad Input (waits for press) | `LD V0, K` |
| `F` | Built-in 5-byte font sprite address | `LD F, V0` |
| `B` | Binary-Coded Decimal (BCD) conversion | `LD B, V0` |
| `[I]` | Memory buffer pointed to by index register `I` | `LD [I], V3` |

### 2. Numbers & Literals

* **Hexadecimal:** Prefixed with `0x` (e.g., `0x0200`, `0x1F`, `0x0A`).
* **Decimal:** Standard numbers (e.g., `512`, `31`, `10`).

### 3. Comments & Formatting

* Lines starting with `;` or `//` are treated as comments and ignored.
* Trailing comments on the same line are supported.
* Instruction keywords and register names are case-insensitive (`ld v0, 0x05` is equivalent to `LD V0, 0x05`).

---

## 📖 Complete Instruction Set Reference (35 Opcodes)

Below is the complete mapping between CHIP-8 binary opcodes and the assembly syntax used by `internal/asm`.

### System & Control Flow

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `00E0` | `CLS` | Clears the display screen. |
| `00EE` | `RET` | Returns from a subroutine (pops PC from stack). |
| `0NNN` | `SYS addr` | Jumps to machine code at `NNN`. *(Ignored in modern ROMs)* |
| `1NNN` | `JP addr` | Jumps to memory address `NNN`. |
| `2NNN` | `CALL addr` | Calls subroutine at address `NNN` (pushes PC to stack). |
| `BNNN` | `JP V0, addr` | Jumps to memory address `NNN + V0`. |

---

### Conditional Branching (Skips)

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `3XNN` | `SE Vx, byte` | Skips next instruction if `Vx == NN`. |
| `4XNN` | `SNE Vx, byte` | Skips next instruction if `Vx != NN`. |
| `5XY0` | `SE Vx, Vy` | Skips next instruction if `Vx == Vy`. |
| `9XY0` | `SNE Vx, Vy` | Skips next instruction if `Vx != Vy`. |
| `EX9E` | `SKP Vx` | Skips next instruction if key stored in `Vx` is pressed. |
| `EXA1` | `SKNP Vx` | Skips next instruction if key stored in `Vx` is **not** pressed. |

---

### Load & Constants

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `6XNN` | `LD Vx, byte` | Sets `Vx = NN`. |
| `8XY0` | `LD Vx, Vy` | Sets `Vx = Vy`. |
| `ANNN` | `LD I, addr` | Sets Index register `I = NNN`. |

---

### Arithmetic & Logic

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `7XNN` | `ADD Vx, byte` | Sets `Vx = Vx + NN` *(VF flag is not changed)*. |
| `8XY1` | `OR Vx, Vy` | Sets `Vx = Vx OR Vy` (bitwise OR). |
| `8XY2` | `AND Vx, Vy` | Sets `Vx = Vx AND Vy` (bitwise AND). |
| `8XY3` | `XOR Vx, Vy` | Sets `Vx = Vx XOR Vy` (bitwise XOR). |
| `8XY4` | `ADD Vx, Vy` | Sets `Vx = Vx + Vy`, sets `VF = 1` if carry occurs. |
| `8XY5` | `SUB Vx, Vy` | Sets `Vx = Vx - Vy`, sets `VF = 1` if no borrow occurs (`Vx >= Vy`). |
| `8XY6` | `SHR Vx` | Shifts `Vx` right by 1 bit. Sets `VF` to the dropped bit. |
| `8XY7` | `SUBN Vx, Vy` | Sets `Vx = Vy - Vx`, sets `VF = 1` if no borrow occurs (`Vy >= Vx`). |
| `8XYE` | `SHL Vx` | Shifts `Vx` left by 1 bit. Sets `VF` to the dropped bit. |
| `CXNN` | `RND Vx, byte` | Sets `Vx = (random 0-255) AND NN`. |

---

### Graphics & Display

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `DXYN` | `DRW Vx, Vy, nibble` | Draws an `N`-byte sprite at screen coordinate `(Vx, Vy)`. Sets `VF = 1` if any set pixels are flipped to unset (collision). |

---

### Timers, Keypad & Memory I/O

| Opcode Hex | Assembly Syntax | Operation / Description |
| --- | --- | --- |
| `FX07` | `LD Vx, DT` | Sets `Vx` to current Delay Timer value. |
| `FX0A` | `LD Vx, K` | Waits for a keypress, then stores the pressed key in `Vx` (blocking). |
| `FX15` | `LD DT, Vx` | Sets Delay Timer = `Vx`. |
| `FX18` | `LD ST, Vx` | Sets Sound Timer = `Vx`. |
| `FX1E` | `ADD I, Vx` | Sets Index register `I = I + Vx`. |
| `FX29` | `LD F, Vx` | Sets `I` to address of font character sprite for hex digit in `Vx`. |
| `FX33` | `LD B, Vx` | Stores BCD representation of `Vx` in memory at `I` (hundreds), `I+1` (tens), `I+2` (ones). |
| `FX55` | `LD [I], Vx` | Dumps registers `V0` through `Vx` into RAM starting at address `I`. |
| `FX65` | `LD Vx, [I]` | Loads RAM starting at address `I` into registers `V0` through `Vx`. |

---

## 💡 Example Assembly Program

Here is an example program demonstrating basic initialization, drawing a sprite, waiting for input, and looping:

```assembly
; Initialize screen & registers
CLS                  ; Clear display
LD V0, 10            ; V0 = X position (10)
LD V1, 15            ; V1 = Y position (15)
LD V2, 0x00          ; V2 = Character '0'

; Set up font sprite location
LD F, V2             ; Point I to font sprite for character in V2 ('0')
DRW V0, V1, 5        ; Draw 5-byte sprite at (10, 15)

; Wait for key press
LD V3, K             ; Halt until any key is pressed, store key in V3

; Infinite Loop
JP 0x0200            ; Jump back to program start (address 0x200)

```

---

## 🧪 Disassembly Output Format

When disassembling a binary ROM, addresses start at `0x0200` (512) where CHIP-8 programs are loaded in RAM.

Disassembler output format:

```text
0x200:  00E0  CLS
0x202:  600A  LD  V0, 0x0A
0x204:  610F  LD  V1, 0x0F
0x206:  F229  LD  F, V2
0x208:  D015  DRW V0, V1, 5
0x20A:  1200  JP  0x0200

```