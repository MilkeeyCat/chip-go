package chip8

import (
	"errors"
	"io"
	"math/rand"
)

const (
	DisplayWidth  = 64
	DisplayHeight = 32
)

const (
	CPUFrequency   = 850
	ClockFrequency = 60
)

const (
	V0 uint8 = 0
	VF uint8 = 15
)

var digitSprites [5 * 16]uint8 = [...]uint8{
	0xf0, 0x90, 0x90, 0x90, 0xf0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xf0, 0x10, 0xf0, 0x80, 0xf0, // 2
	0xf0, 0x10, 0xf0, 0x10, 0xf0, // 3
	0x90, 0x90, 0xf0, 0x10, 0x10, // 4
	0xf0, 0x80, 0xf0, 0x10, 0xf0, // 5
	0xf0, 0x80, 0xf0, 0x90, 0xf0, // 6
	0xf0, 0x10, 0x20, 0x40, 0x40, // 7
	0xf0, 0x90, 0xf0, 0x90, 0xf0, // 8
	0xf0, 0x90, 0xf0, 0x10, 0xf0, // 9
	0xf0, 0x90, 0xf0, 0x90, 0x90, // a
	0xe0, 0x90, 0xe0, 0x90, 0xe0, // b
	0xf0, 0x80, 0x80, 0x80, 0xf0, // c
	0xe0, 0x90, 0x90, 0x90, 0xe0, // d
	0xf0, 0x80, 0xf0, 0x80, 0xf0, // e
	0xf0, 0x80, 0xf0, 0x80, 0x80, // f
}

var ErrUnknownOpcode = errors.New("unknown opcode")

type opcodeHandler func() error

type beep func(on bool)

type Interpreter struct {
	pc uint16    // program counter
	sp uint16    // stack pointer
	dt uint8     // delay timer
	st uint8     // sound timer
	vx [16]uint8 // general purpose 8-bit registers
	i  uint16    // 16-bit register generally used to store memory addresses

	stack  [16]uint16
	memory [4096]uint8

	display         [DisplayHeight][DisplayWidth]uint8
	keys            [16]bool
	lastReleasedKey *Key

	opcode   uint16
	dispatch [16]opcodeHandler

	clockCycleCounter uint8

	beep           beep
	isSoundPlaying bool
}

func NewInterpreter(beep beep) *Interpreter {
	interpreter := &Interpreter{
		pc: 0x200,

		beep: beep,
	}

	interpreter.dispatch = [...]opcodeHandler{
		interpreter.opHandler0, interpreter.opHandler1,
		interpreter.opHandler2, interpreter.opHandler3,
		interpreter.opHandler4, interpreter.opHandler5,
		interpreter.opHandler6, interpreter.opHandler7,
		interpreter.opHandler8, interpreter.opHandler9,
		interpreter.opHandlerA, interpreter.opHandlerB,
		interpreter.opHandlerC, interpreter.opHandlerD,
		interpreter.opHandlerE, interpreter.opHandlerF,
	}

	return interpreter
}

func (i *Interpreter) Load(reader io.Reader) error {
	copy(i.memory[:], digitSprites[:])

	_, err := reader.Read(i.memory[0x200:])

	return err
}

func (i *Interpreter) Display() [DisplayHeight][DisplayWidth]uint8 {
	return i.display
}

func (i *Interpreter) SetKeyState(key Key, pressed bool) {
	i.keys[key] = pressed

	if !pressed {
		i.lastReleasedKey = &key
	}
}

func (i *Interpreter) Cycle() error {
	i.opcode = uint16(i.memory[i.pc])<<8 | uint16(i.memory[i.pc+1])

	if err := i.dispatch[i.opcode>>12](); err != nil {
		return err
	}

	if i.clockCycleCounter == CPUFrequency/ClockFrequency {
		if i.dt > 0 {
			i.dt -= 1
		}

		if i.st > 0 {
			if !i.isSoundPlaying {
				i.beep(true)
				i.isSoundPlaying = true
			}

			i.st -= 1
		} else {
			if i.isSoundPlaying {
				i.beep(false)
				i.isSoundPlaying = false
			}
		}

		i.clockCycleCounter = 0
	} else {
		i.clockCycleCounter += 1
	}

	return nil
}

func (i *Interpreter) opHandler0() error {
	switch i.opcode & 0x00ff {
	// 00E0 - CLS
	// Clear the display.
	case 0x00e0:
		clear(i.display[:])

	// 00EE - RET
	// Return from a subroutine.
	//
	// The interpreter sets the program counter to the address at the top of the
	// stack, then subtracts 1 from the stack pointer.
	case 0x00ee:
		i.sp -= 1
		i.pc = i.stack[i.sp]

	default:
		return ErrUnknownOpcode
	}

	i.pc += 2

	return nil
}

// 1nnn - JP addr
// Jump to location nnn.
//
// The interpreter sets the program counter to nnn.
func (i *Interpreter) opHandler1() error {
	i.pc = addr(i.opcode)

	return nil
}

// 2nnn - CALL addr
// Call subroutine at nnn.
//
// The interpreter increments the stack pointer, then puts the current PC on the
// top of the stack. The PC is then set to nnn.
func (i *Interpreter) opHandler2() error {
	i.stack[i.sp] = i.pc
	i.sp += 1
	i.pc = addr(i.opcode)

	return nil
}

// 3xkk - SE Vx, byte
// Skip next instruction if Vx = kk.
//
// The interpreter compares register Vx to kk, and if they are equal, increments
// the program counter by 2.
func (i *Interpreter) opHandler3() error {
	i.pc += 2

	if i.vx[x(i.opcode)] == byte(i.opcode) {
		i.pc += 2
	}

	return nil
}

// 4xkk - SNE Vx, byte
// Skip next instruction if Vx != kk.
//
// The interpreter compares register Vx to kk, and if they are not equal,
// increments the program counter by 2.
func (i *Interpreter) opHandler4() error {
	i.pc += 2

	if i.vx[x(i.opcode)] != byte(i.opcode) {
		i.pc += 2
	}

	return nil
}

// 5xy0 - SE Vx, Vy
// Skip next instruction if Vx = Vy.
//
// The interpreter compares register Vx to register Vy, and if they are equal,
// increments the program counter by 2.
func (i *Interpreter) opHandler5() error {
	if i.opcode&0x000f != 0 {
		return ErrUnknownOpcode
	}

	i.pc += 2

	if i.vx[x(i.opcode)] == i.vx[y(i.opcode)] {
		i.pc += 2
	}

	return nil
}

// 6xkk - LD Vx, byte
// Set Vx = kk.
//
// The interpreter puts the value kk into register Vx.
func (i *Interpreter) opHandler6() error {
	i.vx[x(i.opcode)] = byte(i.opcode)
	i.pc += 2

	return nil
}

// 7xkk - ADD Vx, byte
// Set Vx = Vx + kk.
//
// Adds the value kk to the value of register Vx, then stores the result in Vx.
func (i *Interpreter) opHandler7() error {
	i.vx[x(i.opcode)] += byte(i.opcode)
	i.pc += 2

	return nil
}

func (i *Interpreter) opHandler8() error {
	switch i.opcode & 0x000f {
	// 8xy0 - LD Vx, Vy
	// Set Vx = Vy.
	//
	// Stores the value of register Vy in register Vx.
	case 0x0000:
		i.vx[x(i.opcode)] = i.vx[y(i.opcode)]

	// 8xy1 - OR Vx, Vy
	// Set Vx = Vx OR Vy.
	//
	// Performs a bitwise OR on the values of Vx and Vy, then stores the result
	// in Vx. A bitwise OR compares the corrseponding bits from two values, and
	// if either bit is 1, then the same bit in the result is also 1. Otherwise,
	// it is 0.
	case 0x0001:
		i.vx[x(i.opcode)] |= i.vx[y(i.opcode)]

	// 8xy2 - AND Vx, Vy
	// Set Vx = Vx AND Vy.
	//
	// Performs a bitwise AND on the values of Vx and Vy, then stores the result
	// in Vx. A bitwise AND compares the corrseponding bits from two values, and
	// if both bits are 1, then the same bit in the result is also 1. Otherwise,
	// it is 0.
	case 0x0002:
		i.vx[x(i.opcode)] &= i.vx[y(i.opcode)]

	// 8xy3 - XOR Vx, Vy
	// Set Vx = Vx XOR Vy.
	//
	// Performs a bitwise exclusive OR on the values of Vx and Vy, then stores
	// the result in Vx. An exclusive OR compares the corrseponding bits from
	// two values, and if the bits are not both the same, then the corresponding
	// bit in the result is set to 1. Otherwise, it is 0.
	case 0x0003:
		i.vx[x(i.opcode)] ^= i.vx[y(i.opcode)]

	// 8xy4 - ADD Vx, Vy
	// Set Vx = Vx + Vy, set VF = carry.
	//
	// The values of Vx and Vy are added together. If the result is greater than
	// 8 bits (i.e., > 255,) VF is set to 1, otherwise 0. Only the lowest 8 bits
	// of the result are kept, and stored in Vx.
	case 0x0004:
		sum := uint16(i.vx[x(i.opcode)]) + uint16(i.vx[y(i.opcode)])

		i.vx[x(i.opcode)] = uint8(sum)

		if sum > 255 {
			i.vx[VF] = 1
		} else {
			i.vx[VF] = 0
		}

	// 8xy5 - SUB Vx, Vy
	// Set Vx = Vx - Vy, set VF = NOT borrow.
	//
	// If Vx > Vy, then VF is set to 1, otherwise 0. Then Vy is subtracted from
	// Vx, and the results stored in Vx.
	//
	// NOTE: the docs say that the condition has to be Vx > Vy, but it's
	// actually Vx >= Vy
	case 0x0005:
		var flag uint8

		if i.vx[x(i.opcode)] >= i.vx[y(i.opcode)] {
			flag = 1
		}

		i.vx[x(i.opcode)] -= i.vx[y(i.opcode)]
		i.vx[VF] = flag

	// 8xy6 - SHR Vx {, Vy}
	// Set Vx = Vx SHR 1.
	//
	// If the least-significant bit of Vx is 1, then VF is set to 1, otherwise
	// 0. Then Vx is divided by 2.
	case 0x0006:
		flag := i.vx[x(i.opcode)] & 0x0001

		i.vx[x(i.opcode)] >>= 1
		i.vx[VF] = flag

	// 8xy7 - SUBN Vx, Vy
	// Set Vx = Vy - Vx, set VF = NOT borrow.
	//
	// If Vy > Vx, then VF is set to 1, otherwise 0. Then Vx is subtracted from
	// Vy, and the results stored in Vx.
	//
	// NOTE: the docs say that the condition has to be Vy > Vx, but it's
	// actually Vy >= Vx
	case 0x0007:
		var flag uint8

		if i.vx[y(i.opcode)] >= i.vx[x(i.opcode)] {
			flag = 1
		}

		i.vx[x(i.opcode)] = i.vx[y(i.opcode)] - i.vx[x(i.opcode)]
		i.vx[VF] = flag

	// 8xyE - SHL Vx {, Vy}
	// Set Vx = Vx SHL 1.
	//
	// If the most-significant bit of Vx is 1, then VF is set to 1, otherwise to
	// 0. Then Vx is multiplied by 2.
	case 0x000E:
		flag := i.vx[x(i.opcode)] >> 7

		i.vx[x(i.opcode)] <<= 1
		i.vx[VF] = flag

	default:
		return ErrUnknownOpcode
	}

	i.pc += 2

	return nil
}

// 9xy0 - SNE Vx, Vy
// Skip next instruction if Vx != Vy.
//
// The values of Vx and Vy are compared, and if they are not equal, the program
// counter is increased by 2.
func (i *Interpreter) opHandler9() error {
	i.pc += 2

	if i.vx[x(i.opcode)] != i.vx[y(i.opcode)] {
		i.pc += 2
	}

	return nil
}

// Annn - LD I, addr
// Set I = nnn.
//
// The value of register I is set to nnn.
func (i *Interpreter) opHandlerA() error {
	i.i = addr(i.opcode)
	i.pc += 2

	return nil
}

// Bnnn - JP V0, addr
// Jump to location nnn + V0.
//
// The program counter is set to nnn plus the value of V0.
func (i *Interpreter) opHandlerB() error {
	i.pc = addr(i.opcode) + uint16(i.vx[V0])

	return nil
}

// Cxkk - RND Vx, byte
// Set Vx = random byte AND kk.
//
// The interpreter generates a random number from 0 to 255, which is then ANDed
// with the value kk. The results are stored in Vx. See instruction 8xy2 for
// more information on AND.
func (i *Interpreter) opHandlerC() error {
	i.vx[x(i.opcode)] = uint8(rand.Intn(256)) & byte(i.opcode)
	i.pc += 2

	return nil
}

// Dxyn - DRW Vx, Vy, nibble
// Display n-byte sprite starting at memory location I at (Vx, Vy),
// set VF = collision.
//
// The interpreter reads n bytes from memory, starting at the address stored in
// I. These bytes are then displayed as sprites on screen at coordinates
// (Vx, Vy). Sprites are XORed onto the existing screen. If this causes any
// pixels to be erased, VF is set to 1, otherwise it is set to 0. If the sprite
// is positioned so part of it is outside the coordinates of the display, it
// wraps around to the opposite side of the screen.
func (i *Interpreter) opHandlerD() error {
	startX := i.vx[x(i.opcode)]
	startY := i.vx[y(i.opcode)]
	height := n(i.opcode)

	i.vx[VF] = 0

	for y := range height {
		for x := range uint8(8) {
			target := &i.display[(startY+y)%DisplayHeight][(startX+x)%DisplayWidth]
			old := *target

			*target ^= (i.memory[i.i+uint16(y)] >> (7 - x)) & 1

			if old == 1 && *target == 0 {
				i.vx[VF] = 1
			}
		}
	}

	i.pc += 2

	return nil
}

func (i *Interpreter) opHandlerE() error {
	i.pc += 2

	// Ex9E - SKP Vx
	// Skip next instruction if key with the value of Vx is pressed.
	//
	// Checks the keyboard, and if the key corresponding to the value of Vx is
	// currently in the down position, PC is increased by 2.
	switch i.opcode & 0x00ff {
	case 0x009e:
		if i.keys[i.vx[x(i.opcode)]&0x0f] {
			i.pc += 2
		}

	// ExA1 - SKNP Vx
	// Skip next instruction if key with the value of Vx is not pressed.
	//
	// Checks the keyboard, and if the key corresponding to the value of Vx is
	// currently in the up position, PC is increased by 2.
	case 0x00a1:
		if !i.keys[i.vx[x(i.opcode)]&0x0f] {
			i.pc += 2
		}

	default:
		return ErrUnknownOpcode
	}

	return nil
}

func (i *Interpreter) opHandlerF() error {
	switch i.opcode & 0x00ff {
	// Fx07 - LD Vx, DT
	// Set Vx = delay timer value.
	//
	// The value of DT is placed into Vx.
	case 0x0007:
		i.vx[x(i.opcode)] = i.dt

	// Fx0A - LD Vx, K
	// Wait for a key press, store the value of the key in Vx.
	//
	// All execution stops until a key is pressed, then the value of that key is
	// stored in Vx.
	//
	// NOTE: it’s incorrect that the instruction waits for a key to be pressed,
	// it waits for one to be released
	case 0x000a:
		if i.lastReleasedKey == nil {
			return nil
		}

		i.vx[x(i.opcode)] = uint8(*i.lastReleasedKey)
		i.lastReleasedKey = nil

	// Fx15 - LD DT, Vx
	// Set delay timer = Vx.
	//
	// DT is set equal to the value of Vx.
	case 0x0015:
		i.dt = i.vx[x(i.opcode)]

	// Fx18 - LD ST, Vx
	// Set sound timer = Vx.
	//
	// ST is set equal to the value of Vx.
	case 0x0018:
		i.st = i.vx[x(i.opcode)]

	// Fx1E - ADD I, Vx
	// Set I = I + Vx.
	//
	// The values of I and Vx are added, and the results are stored in I.
	case 0x001e:
		i.i += uint16(i.vx[x(i.opcode)])

	// Fx29 - LD F, Vx
	// Set I = location of sprite for digit Vx.
	//
	// The value of I is set to the location for the hexadecimal sprite
	// corresponding to the value of Vx.
	case 0x0029:
		i.i = uint16((i.vx[x(i.opcode)] & 0x0f) * 5)

	// Fx33 - LD B, Vx
	// Store BCD representation of Vx in memory locations I, I+1, and I+2.
	//
	// The interpreter takes the decimal value of Vx, and places the hundreds
	// digit in memory at location in I, the tens digit at location I+1, and the
	// ones digit at location I+2.
	case 0x0033:
		value := i.vx[x(i.opcode)]

		i.memory[i.i] = value / 100
		i.memory[i.i+1] = (value / 10) % 10
		i.memory[i.i+2] = value % 10

	// Fx55 - LD [I], Vx
	// Store registers V0 through Vx in memory starting at location I.
	//
	// The interpreter copies the values of registers V0 through Vx into memory,
	// starting at the address in I.
	case 0x0055:
		for n := range x(i.opcode) + 1 {
			i.memory[i.i+uint16(n)] = i.vx[n]
		}

	// Fx65 - LD Vx, [I]
	// Read registers V0 through Vx from memory starting at location I.
	//
	// The interpreter reads values from memory starting at location I into
	// registers V0 through Vx.
	case 0x0065:
		for n := range x(i.opcode) + 1 {
			i.vx[n] = i.memory[i.i+uint16(n)]
		}

	default:
		return ErrUnknownOpcode
	}

	i.pc += 2

	return nil
}

func addr(word uint16) uint16 {
	return word & 0x0fff
}

func x(word uint16) uint8 {
	return uint8((word & 0x0f00) >> 8)
}

func y(word uint16) uint8 {
	return uint8((word & 0x00f0) >> 4)
}

func n(word uint16) uint8 {
	return uint8(word & 0x000f)
}

func byte(word uint16) uint8 {
	return uint8(word & 0x00ff)
}
