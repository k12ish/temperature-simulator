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
	winWidth = 795
	heatMapX = 15
	heatMapY = 15
	heatMapW = 768
	heatMapH = 555	
)

var (
	// initialized by initSDL()
	window *sdl.Window
	renderer *sdl.Renderer
	tex *sdl.Texture
	err error
)

var (
	energyArr  = [heatMapW][heatMapH]uint8{}
)


func initSDL() (err error){
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SDL: %s\n", err)
		return err
	}
	if window, err = sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, winWidth, winHeight, sdl.WINDOW_SHOWN); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", err)
		return err
	}
	if renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		return err
	}
	if 	tex, err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, heatMapW, heatMapH); err != nil{
		fmt.Fprintf(os.Stderr, "Failed to create texture: %s\n", err)
		return err
	}
	tex.SetBlendMode(sdl.BLENDMODE_NONE)
	return nil
}


func main() {
	if initSDL() != nil {
		return
	}
	defer sdl.Quit()
	defer window.Destroy()
	defer renderer.Destroy()
	defer tex.Destroy()

	renderer.SetDrawColor(0,0,0,255)
	renderer.Clear()
	renderer.Present()
	gfx.StringColor(renderer, heatMapX, heatMapY + heatMapH + 15, 
		"Drawing pixels from an array", sdl.Color{0, 255, 0, 255})

	heatMapRect := sdl.Rect{heatMapX, heatMapY, heatMapW, heatMapH}	
	for _, element := range energyArr {
		// fmt.Println(element)
		_ = element
	}

	for offset := 0; offset < 512; offset++ {
		// iterating through offset animates pattern
		bytes, _, err := tex.Lock(nil)
		if err != nil {
			panic(err)
		}

		for i := 0; i < len(bytes); i += 4 {
			// (i/4) % heatMapW + (i/4) / heatMapW represents the
			// manhattan distance from the top left pixel
			minus := (i / 4) - offset
			plus := (i / 4) + offset
			// byte() takes the last 8 bits of integer, eg. mod 256
			bytes[i] = byte(minus % heatMapW + minus / heatMapW)
			bytes[i+1] = byte(plus % heatMapW + plus / heatMapW)
			bytes[i+2] = 0
			bytes[i+3] = 255
		}

		tex.Unlock()
		renderer.Copy(tex, nil, &heatMapRect)
		renderer.Present()		
		sdl.Delay(10)
	}
}


func run() (err error) {
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

