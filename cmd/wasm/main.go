//go:build js && wasm

package main

import (
	"bytes"
	"sync"
	"syscall/js"
	"time"

	chip8 "github.com/MilkeeyCat/chip-go"
)

func main() {
	const fps = 60
	const scale = 15
	keyMap := map[string]chip8.Key{
		"Digit1": chip8.Key1,
		"Digit2": chip8.Key2,
		"Digit3": chip8.Key3,
		"Digit4": chip8.KeyC,
		"KeyQ":   chip8.Key4,
		"KeyW":   chip8.Key5,
		"KeyE":   chip8.Key6,
		"KeyR":   chip8.KeyD,
		"KeyA":   chip8.Key7,
		"KeyS":   chip8.Key8,
		"KeyD":   chip8.Key9,
		"KeyF":   chip8.KeyE,
		"KeyZ":   chip8.KeyA,
		"KeyX":   chip8.Key0,
		"KeyC":   chip8.KeyB,
		"KeyV":   chip8.KeyF,
	}

	document := js.Global().Get("document")
	canvas := document.Call("querySelector", "#canvas")

	canvas.Set("width", chip8.DisplayWidth)
	canvas.Set("height", chip8.DisplayHeight)

	clampedArray := js.Global().Get("Uint8ClampedArray").New(chip8.DisplayWidth * chip8.DisplayHeight * 4)
	js.Global().Set("imageData", js.Global().Get("ImageData").New(clampedArray, chip8.DisplayWidth))

	ctx := canvas.Call("getContext", "2d")
	keyboard := newKeyboard()
	defer keyboard.Close()
	var interpreter *chip8.Interpreter
	var mu sync.Mutex
	start := make(chan struct{})
	beeper := func(on bool) {
		beeper := js.Global().Get("window").Get("beeper")

		if on {
			beeper.Call("on")
		} else {
			beeper.Call("off")
		}
	}

	setupROMLoader(&interpreter, beeper, start, &mu)

	<-start

	drawingCycleCounter := 0
	imageData := js.Global().Get("imageData")
	imageDataBuf := imageData.Get("data")
	var buf [chip8.DisplayWidth * chip8.DisplayHeight * 4]byte

	for {
		mu.Lock()

		if err := interpreter.Cycle(); err != nil {
			panic(err)
		}

		mu.Unlock()

		for event := keyboard.pollEvent(); event != nil; event = keyboard.pollEvent() {
			if key, ok := keyMap[event.key]; ok {
				interpreter.SetKeyState(key, event.pressed)
			}
		}

		if drawingCycleCounter == chip8.CPUFrequency/fps {
			for j, rows := range interpreter.Display() {
				for i, value := range rows {
					if value != 0 {
						value = 255
					}

					base := (j*chip8.DisplayWidth + i) * 4

					buf[base+0] = value
					buf[base+1] = value
					buf[base+2] = value
					buf[base+3] = 255
				}
			}

			js.CopyBytesToJS(imageDataBuf, buf[:])
			ctx.Call("putImageData", imageData, 0, 0)

			drawingCycleCounter = 0
		} else {
			drawingCycleCounter += 1
		}

		time.Sleep(time.Second / chip8.CPUFrequency)
	}
}

func setupROMLoader(interpreter **chip8.Interpreter, beep func(on bool), start chan struct{}, mu *sync.Mutex) {
	document := js.Global().Get("document")
	input := document.Call("querySelector", "#rom")

	var once sync.Once

	input.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
		args[0].Get("target").Get("files").Index(0).Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			var buf [4096]byte

			js.CopyBytesToGo(buf[:], js.Global().Get("Uint8ClampedArray").New(args[0]))
			mu.Lock()

			*interpreter = chip8.NewInterpreter(beep)
			if err := (*interpreter).Load(bytes.NewReader(buf[:])); err != nil {
				panic(err)
			}

			mu.Unlock()
			once.Do(func() {
				close(start)
			})

			return nil
		}))

		return nil
	}))
}
