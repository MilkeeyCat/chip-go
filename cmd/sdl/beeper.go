package main

// void audioCallback(void *userdata, unsigned char *stream, int len);
import "C"

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

type beeper struct {
	id sdl.AudioDeviceID
}

func newBeeper() (*beeper, error) {
	spec := sdl.AudioSpec{
		Freq:     44100,
		Format:   sdl.AUDIO_S8,
		Channels: 2,
		Samples:  100,
		Callback: sdl.AudioCallback(C.audioCallback),
	}
	id, err := sdl.OpenAudioDevice(sdl.GetAudioDeviceName(0, false), false, &spec, nil, 0)
	if err != nil {
		return nil, err
	}

	return &beeper{id}, nil
}

func (b *beeper) play() {
	sdl.PauseAudioDevice(b.id, false)
}

func (b *beeper) stop() {
	sdl.PauseAudioDevice(b.id, true)
}

func (b *beeper) close() {
	sdl.CloseAudioDevice(b.id)
}

//export audioCallback
func audioCallback(_ unsafe.Pointer, stream *C.uchar, length C.int) {
	const amplitude = 1
	n := int(length)
	buf := unsafe.Slice((*C.schar)(unsafe.Pointer(stream)), n)

	for i := 0; i < n; i += 2 {
		var sample C.schar

		if i > n/2 {
			sample = C.schar(amplitude)
		} else {
			sample = C.schar(-amplitude)
		}

		buf[i] = sample
		buf[i+1] = sample
	}
}
