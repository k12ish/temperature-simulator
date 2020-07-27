package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"os"
	"sync"
)

const (
	marginN = 15
	marginE = 15
	marginS = 40
	marginW = 15

	heatMapW = 500
	heatMapH = 457

	winTitle  = "Temperature Simulator"
	winHeight = marginN + heatMapH + marginS
	winWidth  = marginW + heatMapW + marginE
)

// initialized by initSDL()
var (
	window   *sdl.Window
	renderer *sdl.Renderer
	tex      *sdl.Texture
	err      error

	heatMapRectPtr  = &sdl.Rect{marginW, marginN, heatMapW, heatMapH}
)

type material byte

const (
	Metal material = 1 << iota
	Plastic
	Glass
	NumberOfMaterials = iota
)

var (
	energyArr   = [heatMapH][heatMapW]float32{}
	materialArr = [heatMapH][heatMapW]material{}

	material_HeatCap = [NumberOfMaterials]float32{}
	material_Conduct = [1 << NumberOfMaterials]float32{}
)

var (
	program_running = true
	standardBrush   = BrushConstructor(10)
	leftMouse_click = make(chan HeatMap_Event, 100)
)
type HeatMap_Event interface {
	Draw_to_Arr()
}

type point struct {
	X int32
	Y int32
}

func (p point) Draw_to_Arr() {
	// if the point is inside heatmap
	if x, y := p.X - marginW, p.Y - marginN;
	   0 <= x && x < heatMapW && 0 <= y && y < heatMapH {
	   	// calculate the minimum distance to the wall
		minumum_dist := x
		for _, v := range [3]int32{y, heatMapW - x, heatMapH - y} {
			if v < minumum_dist {
				minumum_dist = v
			}
		}
		if standardBrush.MaxRadius < minumum_dist {
			for _, list := range standardBrush.Points {
				energyArr[y+list[1]][x+list[0]] += 100
			}
		}
	}
}


func initSDL() (err error) {
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
	if tex, err = renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, heatMapW, heatMapH); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create texture: %s\n", err)
		return err
	}
	tex.SetBlendMode(sdl.BLENDMODE_NONE)
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()
	renderer.Present()
	gfx.StringColor(renderer, marginW, marginN+heatMapH+15,
		"Fourier's Law implementation bug", sdl.Color{0, 255, 0, 255})
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

	leftMousePressed := false

	var heatMap_wg sync.WaitGroup
	// HeatMap_wg prevents the program from executing deferred functions
	// Prevents texture being destroyed while runHeatMap() is running
	go runHeatMap(&heatMap_wg)
	heatMap_wg.Add(1)

	for program_running {
		switch t := sdl.PollEvent().(type) {
		case *sdl.QuitEvent:
			program_running = false
		case *sdl.MouseMotionEvent:
			if leftMousePressed {
				leftMouse_click <- point{t.X, t.Y}
			}
		case *sdl.MouseButtonEvent:
			if t.Button == sdl.BUTTON_LEFT {
				if t.State == sdl.PRESSED {
					leftMouse_click <- point{t.X, t.Y}
					leftMousePressed = true
				} else {
					leftMousePressed = false
				}
			}
		}
	}
	heatMap_wg.Wait()
}


func runHeatMap(func_wg *sync.WaitGroup) {
	defer func_wg.Done()
	var channel_empty bool
	var elem HeatMap_Event

	for program_running {
		channel_empty = true
		for channel_empty {
			select {
			case elem = <-leftMouse_click:
				elem.Draw_to_Arr()
			default:
				channel_empty = false
			}
		}
		heatFlow()
		buildHeatmap()
	}
}

func heatFlow() {
	for i := 0; i < heatMapH; i++ {
		for j := 1; j < heatMapW; j++ {
			q := (energyArr[i][j] - energyArr[i][j-1]) / 2
			energyArr[i][j] -= q
			energyArr[i][j-1] += q
		}
	}
	for i := 1; i < heatMapH; i++ {
		for j := 0; j < heatMapW; j++ {
			q := (energyArr[i][j] - energyArr[i-1][j]) / 2
			energyArr[i][j] -= q
			energyArr[i-1][j] += q
		}
	}
}

func buildHeatmap() {
	bytes, _, err := tex.Lock(nil)
	if err != nil {
		panic(err)
	}
	for i, sublist := range energyArr {
		for j, value := range sublist {
			bytes[(heatMapW*i+j)*4] = uint8(value)
		}
	}
	tex.Unlock()
	renderer.Copy(tex, nil, heatMapRectPtr)
	renderer.Present()
}

func stockAnimation() {
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
			bytes[i] = byte(minus%heatMapW + minus/heatMapW)
			bytes[i+1] = byte(plus%heatMapW + plus/heatMapW)
			bytes[i+2] = 0
			bytes[i+3] = 255
		}

		tex.Unlock()
		renderer.Copy(tex, nil, heatMapRectPtr)
		renderer.Present()
	}
}

type Brush struct {
	Points    [][2]int32
	MaxRadius int32
}

func BrushConstructor(radii ...int32) Brush {
	var max int32
	var squared int32
	var array [][2]int32
	for _, r := range radii {
		if r > max {
			max = r
		}
		squared = r * r
		for i := int32(0); i <= r; i++ {
			for j := int32(0); j <= r; j++ {
				if i*i+j*j > squared {
					continue
				}
				if j != 0 {
					if i != 0 {
						array = append(array,
							[2]int32{i, j}, [2]int32{i, -j},
							[2]int32{-i, j}, [2]int32{-i, -j})
					} else {
						array = append(array,
							[2]int32{i, j}, [2]int32{i, -j})
					}
				} else {
					if i != 0 {
						array = append(array,
							[2]int32{i, j}, [2]int32{-i, j})
					} else {
						array = append(array, [2]int32{i, j})
					}
				}
			}
		}
	}
	return Brush{array, max}
}
