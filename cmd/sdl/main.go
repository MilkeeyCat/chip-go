package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/veandco/go-sdl2/sdl"

	chip8 "github.com/MilkeeyCat/chip-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./chip8 <path/to/rom/file>")

		return
	}

	const fps = 60
	const scale = 15
	keyMap := map[sdl.Keycode]chip8.Key{
		sdl.K_1: chip8.Key1,
		sdl.K_2: chip8.Key2,
		sdl.K_3: chip8.Key3,
		sdl.K_4: chip8.KeyC,
		sdl.K_q: chip8.Key4,
		sdl.K_w: chip8.Key5,
		sdl.K_e: chip8.Key6,
		sdl.K_r: chip8.KeyD,
		sdl.K_a: chip8.Key7,
		sdl.K_s: chip8.Key8,
		sdl.K_d: chip8.Key9,
		sdl.K_f: chip8.KeyE,
		sdl.K_z: chip8.KeyA,
		sdl.K_x: chip8.Key0,
		sdl.K_c: chip8.KeyB,
		sdl.K_v: chip8.KeyF,
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("CHIP-8", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, chip8.DisplayWidth*scale, chip8.DisplayHeight*scale, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, 0)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	beeper, err := newBeeper()
	if err != nil {
		panic(err)
	}
	defer beeper.close()

	beep := func(on bool) {
		if on {
			beeper.play()
		} else {
			beeper.stop()
		}
	}

	interpreter := chip8.NewInterpreter(beep)
	f, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	if err := interpreter.Load(bytes.NewReader(f)); err != nil {
		panic(err)
	}

	drawingCycleCounter := 0

	for {
		if err := interpreter.Cycle(); err != nil {
			panic(err)
		}

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event := event.(type) {
			case *sdl.QuitEvent:
				os.Exit(0)

			case *sdl.KeyboardEvent:
				if key, ok := keyMap[event.Keysym.Sym]; ok {
					switch event.Type {
					case sdl.KEYUP:
						interpreter.SetKeyState(key, false)
					case sdl.KEYDOWN:
						interpreter.SetKeyState(key, true)
					}
				}
			}
		}

		if drawingCycleCounter == chip8.CPUFrequency/fps {
			renderer.SetDrawColor(0, 0, 0, 255)
			renderer.Clear()

			display := interpreter.Display()

			for j, rows := range display {
				for i, value := range rows {
					if value == 0 {
						renderer.SetDrawColor(0, 0, 0, 255)
					} else {
						renderer.SetDrawColor(255, 255, 255, 255)
					}

					renderer.FillRect(&sdl.Rect{
						Y: int32(j) * scale,
						X: int32(i) * scale,
						W: scale,
						H: scale,
					})
				}
			}

			drawingCycleCounter = 0
			renderer.Present()
		} else {
			drawingCycleCounter += 1
		}

		time.Sleep(time.Second / chip8.CPUFrequency)
	}
}
