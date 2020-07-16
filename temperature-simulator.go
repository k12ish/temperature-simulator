package main

import (
	"fmt"
	"os"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/gfx"
)

const (
	winTitle = "Temperature Simulator"
	winHeight = 600
	winWidth = 1000
)

func run() (err error) {
	var window *sdl.Window
	var renderer *sdl.Renderer

	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SDL: %s\n", err)
		return
	}
	defer sdl.Quit()

	if window, err = sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, winWidth, winHeight, sdl.WINDOW_SHOWN); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", err)
		return
	}
	defer window.Destroy()

	if renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		return
	}
	defer renderer.Destroy()
	clear(renderer)

	running := true
	for running {
			switch t := sdl.PollEvent().(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				fmt.Println("Mouse", t.Which, "at", t.X, t.Y)
				if t.State == sdl.PRESSED {
					draw(renderer, t.X, t.Y)
				}

			case *sdl.MouseButtonEvent:
				if t.State == sdl.PRESSED {
					draw(renderer, t.X, t.Y)
				}

			case *sdl.KeyboardEvent:
				if string(t.Keysym.Sym) == "c" {
					clear(renderer)
				}
			}
		// sdl.Delay(1)
	}

	return
}

func draw(renderer *sdl.Renderer, X int32, Y int32) {
	gfx.FilledCircleColor(renderer, X, Y, 20, sdl.Color{0, 255, 0, 22})
	gfx.FilledCircleColor(renderer, X, Y, 10, sdl.Color{0, 255, 0, 22})
	renderer.Present()
}

func clear(renderer *sdl.Renderer) {
	renderer.SetDrawColor(0,0,0,255)
	renderer.Clear()
	gfx.StringColor(renderer, 16, 16, "GFX Demo", sdl.Color{0, 255, 0, 255})
	renderer.Present()
}
/*
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}


exit program on 'quit button'

running := true
for running{
	switch t := sdl.PollEvent().(type) {
	case *sdl.QuitEvent:
		running = false
	...
}
*/

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		400, 400, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}

	renderer.SetDrawColor(0, 0, 0, 0)
	renderer.Clear()

	tex, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING,
		200, 200)

	rect := sdl.Rect{
		X: 15,
		Y: 15,
		W: 200,
		H: 200,
	}

	// We'll lock the texture to draw a centered yellow square.
	bytes, pitch, err := tex.Lock(nil)
	_ = pitch
	if err != nil {
		panic(err)
	}
	tex.SetBlendMode(sdl.BLENDMODE_NONE)

	for i := 0; i < len(bytes); i += 4 {
		bytes[i] = byte(i%255)
		bytes[i+1] = 255
		bytes[i+2] = 0
		bytes[i+3] = 255
	}

	tex.Unlock()


	renderer.Copy(tex, nil, &rect)
	renderer.Present()

	sdl.Delay(5000)
}