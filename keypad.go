package chip8

// The computers which originally used the Chip-8 Language had a 16-key
// hexadecimal keypad with the following layout:
//
// +---+---+---+---+
// | 1 | 2 | 3 | C |
// +---+---+---+---+
// | 4 | 5 | 6 | D |
// +---+---+---+---+
// | 7 | 8 | 9 | E |
// +---+---+---+---+
// | A | 0 | B | F |
// +---+---+---+---+

type Key uint8

const (
	Key0 Key = iota
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
)
