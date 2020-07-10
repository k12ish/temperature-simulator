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
	winWidth = 800
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

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

/*
exit program on 'quit button'

running := true
for running{
	switch t := sdl.PollEvent().(type) {
	case *sdl.QuitEvent:
		running = false
	...
}
*/