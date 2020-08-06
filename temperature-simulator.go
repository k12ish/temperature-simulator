package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"math"
	"os"
	"sync"
)

const (
	marginN = 15
	marginE = 15
	marginS = 40
	marginW = 15

	heatMapW = 150
	heatMapH = 155

	winTitle  = "Temperature Simulator"
	winHeight = marginN + heatMapH + marginS
	winWidth  = marginW + heatMapW + marginE

	VIEW_TEMPERATURE = iota
	VIEW_MATERIAL
)

// initialized by initSDL()
var (
	window   *sdl.Window
	renderer *sdl.Renderer
	tex      *sdl.Texture
	err      error

	heatMapRectPtr = &sdl.Rect{marginW, marginN, heatMapW, heatMapH}
)

type material byte

const (
	Aluminium material = 1 << iota
	Glass
	Water
	MaxMaterials
)

var (
	energyArr   = [heatMapH][heatMapW]float32{}
	temperArr   = [heatMapH][heatMapW]float32{}
	materialArr = [heatMapH][heatMapW]material{}

	heatCapacity = [MaxMaterials]float32{}
	conductivity = [MaxMaterials]float32{}
)

var (
	program_running = true
	program_view    = VIEW_TEMPERATURE

	HeatMap_Event_chan = make(chan HeatMap_Event, 100)
	standardBrush      = BrushConstructor(10)
)

func initMaterials() {
	// Isobaric (volumetric) Heatcapacities
	heatCapacity[Aluminium] = 2.422 // J / cm^3 K
	heatCapacity[Glass] = 2.1       // J / cm^3 K
	heatCapacity[Water] = 4.179     // J / cm^3 K

	conductivity[Aluminium] = 205.0 / 100 // W / m K
	conductivity[Glass] = 0.8 / 100       // W / m K
	conductivity[Water] = 0.6 / 100       // W / m K

	// iterate through all possible pairs of materials
	for i := material(1); i < MaxMaterials; i <<= 1 {
		for j := material(i) << 1; j < MaxMaterials; j <<= 1 {
			// the material conductivity between i and j is equal to
			conductivity[i|j] = conductivity[i] + conductivity[j]
			conductivity[i|j] /= 2
			// the average conductivities of i and j
		}
	}

	for i := range materialArr {
		for j := range materialArr[i] {
			if j > int(heatMapW/2) {
				materialArr[i][j] = Glass
			} else {
				materialArr[i][j] = Aluminium
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

	initMaterials()

	var heatMap_wg sync.WaitGroup
	go runHeatMap(&heatMap_wg)
	heatMap_wg.Add(1)
	// Prevents texture being destroyed while runHeatMap() is running

	leftMousePressed := false
	var rightMousePrevious point

	for program_running {
		switch t := sdl.PollEvent().(type) {
		case *sdl.QuitEvent:
			program_running = false
		case *sdl.MouseMotionEvent:
			if leftMousePressed {
				HeatMap_Event_chan <- point{t.X, t.Y}
			}
		case *sdl.MouseButtonEvent:
			if t.Button == sdl.BUTTON_LEFT {
				if t.State == sdl.PRESSED {
					HeatMap_Event_chan <- point{t.X, t.Y}
					leftMousePressed = true
				} else {
					leftMousePressed = false
				}
			} else if t.Button == sdl.BUTTON_RIGHT {
				if t.State == sdl.PRESSED {
					rightMousePrevious.X = t.X
					rightMousePrevious.Y = t.Y
				} else {
					HeatMap_Event_chan <- Set_Rect{
						X1:    rightMousePrevious.X,
						Y1:    rightMousePrevious.Y,
						X2:    t.X,
						Y2:    t.Y,
						Value: Aluminium,
					}
				}
			}
		case *sdl.KeyboardEvent:
			keyCode := t.Keysym.Sym
			fmt.Println(string(keyCode))
			switch string(keyCode) {
			case " ":
				switch program_view {
				case VIEW_TEMPERATURE:
					program_view = VIEW_MATERIAL
				case VIEW_MATERIAL:
					program_view = VIEW_TEMPERATURE
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
			case elem = <-HeatMap_Event_chan:
				elem.Draw_to_Arr()
			default:
				channel_empty = false
			}
		}
		switch program_view {
		case VIEW_TEMPERATURE:
			heatFlow()
			showTemperature()
		case VIEW_MATERIAL:
			showMaterial()
		}
	}
}

func heatFlow() {
	for i := 0; i < heatMapH; i++ {
		for j := 1; j < heatMapW; j++ {
			q := (temperArr[i][j] - temperArr[i][j-1]) *
				conductivity[materialArr[i][j]|materialArr[i][j-1]] *
				0.1
			if math.IsNaN(float64(q)) {
				fmt.Println("NaN error")
				return
			}
			energyArr[i][j] -= q
			energyArr[i][j-1] += q
		}
	}
	for i := 1; i < heatMapH; i++ {
		for j := 0; j < heatMapW; j++ {
			q := (temperArr[i][j] - temperArr[i-1][j]) *
				conductivity[materialArr[i][j]|materialArr[i-1][j]] *
				0.1
			if math.IsNaN(float64(q)) {
				fmt.Println("NaN error")
				return
			}
			energyArr[i][j] -= q
			energyArr[i-1][j] += q
		}
	}
	for i := range energyArr {
		for j := range energyArr[i] {
			temperArr[i][j] = energyArr[i][j] / heatCapacity[materialArr[i][j]]
		}
	}
}

func showTemperature() {
	bytes, _, err := tex.Lock(nil)
	if err != nil {
		panic(err)
	}
	for i, sublist := range temperArr {
		for j, value := range sublist {
			bytes[(heatMapW*i+j)*4] = uint8(value)
		}
	}
	tex.Unlock()
	renderer.Copy(tex, nil, heatMapRectPtr)
	renderer.Present()
}

func showMaterial() {
	bytes, _, err := tex.Lock(nil)
	if err != nil {
		panic(err)
	}
	for i, sublist := range temperArr {
		for j, value := range sublist {
			bytes[(heatMapW*i+j)*4+1] = uint8(value)
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
			// byte() takes the last 8 bits  integer, eg. mod 256
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

type HeatMap_Event interface {
	Draw_to_Arr()
}

func rel_to_heatMap(X int32, Y int32) (x, y int32, inside bool) {
	// inlined by compiler -- no function call overhead
	x, y = X-marginW, Y-marginN
	inside = 0 <= x && x < heatMapW && 0 <= y && y < heatMapH
	return
}

type point struct {
	X int32
	Y int32
}

func (p point) Draw_to_Arr() {
	// if the point is inside heatmap
	if x, y, inside := rel_to_heatMap(p.X, p.Y); inside {
		// calculate the minimum distance to the wall
		minumum_dist := x
		for _, v := range [3]int32{y, heatMapW - x, heatMapH - y} {
			if v < minumum_dist {
				minumum_dist = v
			}
		}
		if standardBrush.MaxRadius < minumum_dist {
			for _, list := range standardBrush.Points {
				energyArr[y+list[1]][x+list[0]] += heatCapacity[Water] * 10
			}
		}
	}
}

type Set_Rect struct {
	X1    int32
	Y1    int32
	X2    int32
	Y2    int32
	Value interface{}
}

func (r Set_Rect) Draw_to_Arr() {
	x1, y1, inside1 := rel_to_heatMap(r.X1, r.Y1)
	x2, y2, inside2 := rel_to_heatMap(r.X2, r.Y2)
	if !inside1 || !inside2 {
		return
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	switch r.Value.(type) {
	case material:
		for y := y1; y <= y2; y++ {
			for x := x1; x <= x2; x++ {
				materialArr[y][x] = r.Value.(material)
			}
		}
	default:
		fmt.Println("Set_Rect recieved unsupported type")
	}
}
