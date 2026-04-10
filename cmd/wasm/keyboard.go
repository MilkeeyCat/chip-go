//go:build js && wasm

package main

import "syscall/js"

type event struct {
	pressed bool
	key     string
}

type keyboard struct {
	events []event

	onKeyDown js.Func
	onKeyUp   js.Func
}

func newKeyboard() *keyboard {
	keyboard := new(keyboard)
	window := js.Global().Get("window")

	keyboard.onKeyDown = js.FuncOf(func(this js.Value, args []js.Value) any {
		keyboard.events = append(keyboard.events, event{
			pressed: true,
			key:     args[0].Get("code").String(),
		})

		return nil
	})
	keyboard.onKeyUp = js.FuncOf(func(this js.Value, args []js.Value) any {
		keyboard.events = append(keyboard.events, event{
			pressed: false,
			key:     args[0].Get("code").String(),
		})

		return nil
	})

	window.Call("addEventListener", "keydown", keyboard.onKeyDown)
	window.Call("addEventListener", "keyup", keyboard.onKeyUp)

	return keyboard
}

func (k *keyboard) pollEvent() *event {
	eventsLen := len(k.events)

	if eventsLen > 0 {
		event := k.events[eventsLen-1]

		k.events = k.events[:eventsLen-1]

		return &event
	}

	return nil
}

func (k *keyboard) Close() {
	window := js.Global().Get("window")

	window.Call("removeEventListener", "keydown", k.onKeyDown)
	window.Call("removeEventListener", "keyup", k.onKeyUp)

	k.onKeyDown.Release()
	k.onKeyUp.Release()
}
