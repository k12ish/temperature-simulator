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

	heatMapW = 400
	heatMapH = 400

	winTitle  = "Temperature Simulator"
	winHeight = marginN + heatMapH + marginS
	winWidth  = marginW + heatMapW + marginE
)

var (
	// initialized by initSDL()
	window   *sdl.Window
	renderer *sdl.Renderer
	tex      *sdl.Texture
	err      error

	// misc
	heatMapRectPtr     = &sdl.Rect{marginW, marginN, heatMapW, heatMapH}
	program_running    = true
	HeatMap_Event_chan = make(chan HeatMap_Event, 100)
	standardBrush      = BrushConstructor(10)
)

type material byte

const (
	Aluminium material = 1 << iota
	Glass
	Water
	MaxMaterials
)

var selected_material = material(1)

func (m material) Draw_to_Arr() {
	selected_material = m
}

type Element struct {
	material material
	energy   float32
}

func (E Element) Temperature() float32 {
	return E.energy * recip_heatCapacity[E.material]
}

var (
	elementArr = [heatMapH][heatMapW]Element{}

	heatCapacity       = [MaxMaterials]float32{}
	recip_heatCapacity = [MaxMaterials]float32{}
	conductivity       = [MaxMaterials]float32{}
)

func initMaterials() {
	// Isobaric (volumetric) Heatcapacities
	heatCapacity[Aluminium] = 2.422 // J / cm^3 K
	heatCapacity[Glass] = 2.1       // J / cm^3 K
	heatCapacity[Water] = 4.179     // J / cm^3 K

	for i := range heatCapacity {
		// storing calculated values reduces FLOPS
		recip_heatCapacity[i] = 1 / heatCapacity[i]
	}

	// Thermal conductivies
	conductivity[Aluminium] = 205.0 // W / cm K
	conductivity[Glass] = 0.8       // W / cm K
	conductivity[Water] = 0.6       // W / cm K

	// iterate through all possible pairs of material elements
	for i := material(1); i < MaxMaterials; i <<= 1 {
		for j := material(i) << 1; j < MaxMaterials; j <<= 1 {
			// the material conductivity between i and j is equal to
			// the average conductivities of i and j
			conductivity[i|j] = conductivity[i] + conductivity[j]
			conductivity[i|j] /= 2
		}
	}

	for i := range conductivity {
		// premultiply by sidelength of cubic element
		conductivity[i] *= 0.01
	}

	for i := range elementArr {
		for j := range elementArr[i] {
			if j < int(heatMapW/2) {
				elementArr[i][j].material = Aluminium
			} else {
				elementArr[i][j].material = Glass
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
	var channel_empty bool
	var event HeatMap_Event
	var main_thread sync.WaitGroup
	// Delay destruction of texture until texture not needed
	go parseUserInput(&main_thread)
	main_thread.Add(1)

	initMaterials()
	event = <-HeatMap_Event_chan

	for program_running {
		channel_empty = true
		for channel_empty {
			select {
			case event = <-HeatMap_Event_chan:
				event.Draw_to_Arr()
			default:
				channel_empty = false
			}
		}
		switch program_view {
		case TEMPERATURE_VIEW:
			heatFlow()
			showTemperature()
		case MATERIAL_VIEW:
			showMaterial()
		}
	}
	main_thread.Done()
}

func parseUserInput(main_thread *sync.WaitGroup) {
	if initSDL() != nil {
		return
	}
	defer sdl.Quit()
	defer window.Destroy()
	defer renderer.Destroy()
	defer tex.Destroy()

	var leftMousePressed bool
	var rightMousePreviousX int32
	var rightMousePreviousY int32
	HeatMap_Event_chan <- nil

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
					rightMousePreviousX = t.X
					rightMousePreviousY = t.Y
				} else {
					HeatMap_Event_chan <- Set_Rect{
						X1:    rightMousePreviousX,
						Y1:    rightMousePreviousY,
						X2:    t.X,
						Y2:    t.Y,
						Value: selected_material}
				}
			}
		case *sdl.KeyboardEvent:
			switch string(t.Keysym.Sym) {
			case " ":
				// switch between MATERIAL and TEMPERATURE views
				if t.State == sdl.PRESSED {
					HeatMap_Event_chan <- SWITCH_VIEW
				}
			case "r":
				// clear the screen
				HeatMap_Event_chan <- Set_Rect{
					X1:    marginW,
					Y1:    marginN,
					X2:    marginW + heatMapW - 1,
					Y2:    marginN + heatMapH - 1,
					Value: float32(0)}
			case "1":
				if material(1) < MaxMaterials {
					HeatMap_Event_chan <- material(1)
				}
			case "2":
				if material(2) < MaxMaterials {
					HeatMap_Event_chan <- material(2)
				}
			case "3":
				if material(4) < MaxMaterials {
					HeatMap_Event_chan <- material(4)
				}
			case "4":
				if material(8) < MaxMaterials {
					HeatMap_Event_chan <- material(8)
				}
			case "5":
				if material(16) < MaxMaterials {
					HeatMap_Event_chan <- material(16)
				}
			}
		}
	}
	main_thread.Wait()
}

func heatFlow() {
	for i := 0; i < heatMapH; i++ {
		for j := 1; j < heatMapW; j++ {
			a, b := elementArr[i][j], elementArr[i][j-1]
			q := (a.Temperature() - b.Temperature()) *
				conductivity[a.material|b.material] *
				0.05

			elementArr[i][j].energy -= q
			elementArr[i][j-1].energy += q
		}
	}
	for i := 1; i < heatMapH; i++ {
		for j := 0; j < heatMapW; j++ {
			a, b := elementArr[i][j], elementArr[i-1][j]
			q := (a.Temperature() - b.Temperature()) *
				conductivity[a.material|b.material] *
				0.05

			elementArr[i][j].energy -= q
			elementArr[i-1][j].energy += q
		}
	}
}

type viewType byte

var program_view viewType = TEMPERATURE_VIEW

const (
	TEMPERATURE_VIEW viewType = iota
	MATERIAL_VIEW
	SWITCH_VIEW
)

func (view viewType) Draw_to_Arr() {
	if program_view == view {
		return
	}

	if view == SWITCH_VIEW {
		switch program_view {
		case TEMPERATURE_VIEW:
			program_view = MATERIAL_VIEW
		case MATERIAL_VIEW:
			program_view = TEMPERATURE_VIEW
		}
	} else {
		program_view = view
	}
	// since the view has changed, set the texture to black
	bytes, _, err := tex.Lock(nil)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(bytes); i++ {
		bytes[i] = 0
	}
	tex.Unlock()
	renderer.Copy(tex, nil, heatMapRectPtr)
	renderer.Present()
}

func showTemperature() {
	bytes, _, err := tex.Lock(nil)
	if err != nil {
		panic(err)
	}
	for i, sublist := range elementArr {
		for j, value := range sublist {
			bytes[(heatMapW*i+j)*4] = uint8(value.Temperature())
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
	for i, sublist := range elementArr {
		heatMapW_star_i := heatMapW * i
		for j, value := range sublist {
			switch value.material {
			case material(1):
				bytes[(heatMapW_star_i+j)*4] = uint8(140)
				bytes[(heatMapW_star_i+j)*4+1] = uint8(138)
				bytes[(heatMapW_star_i+j)*4+2] = uint8(232)
			case material(2):
				bytes[(heatMapW_star_i+j)*4] = uint8(193)
				bytes[(heatMapW_star_i+j)*4+1] = uint8(198)
				bytes[(heatMapW_star_i+j)*4+2] = uint8(75)
			case material(4):
				bytes[(heatMapW_star_i+j)*4] = uint8(227)
				bytes[(heatMapW_star_i+j)*4+1] = uint8(103)
				bytes[(heatMapW_star_i+j)*4+2] = uint8(189)
			case material(8):
				bytes[(heatMapW_star_i+j)*4] = uint8(106)
				bytes[(heatMapW_star_i+j)*4+1] = uint8(208)
				bytes[(heatMapW_star_i+j)*4+2] = uint8(122)
			default:
				bytes[(heatMapW_star_i+j)*4] = uint8(226)
				bytes[(heatMapW_star_i+j)*4+1] = uint8(124)
				bytes[(heatMapW_star_i+j)*4+2] = uint8(75)
			}
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
		array = append(array, [2]int32{0, 0})
		for j := int32(1); j <= r; j++ {
			array = append(array, [2]int32{0, j}, [2]int32{0, -j})
		}
		for i := int32(1); i <= r; i++ {
			array = append(array, [2]int32{i, 0}, [2]int32{-i, 0})
		}
		for i := int32(1); i <= r; i++ {
			for j := int32(1); j <= r; j++ {
				if i*i+j*j > squared {
					continue
				}
				array = append(array,
					[2]int32{i, j}, [2]int32{i, -j},
					[2]int32{-i, j}, [2]int32{-i, -j})
			}
		}
	}
	return Brush{array, max}
}

func rel_to_heatMap(X int32, Y int32) (x, y int32, inside bool) {
	// inlined by compiler -- no function call overhead
	x, y = X-marginW, Y-marginN
	inside = 0 <= x && x < heatMapW && 0 <= y && y < heatMapH
	return
}

type HeatMap_Event interface {
	Draw_to_Arr()
}

type point struct {
	X int32
	Y int32
}

func (p point) Draw_to_Arr() {
	// if the point is inside heatmap
	if x, y, inside := rel_to_heatMap(p.X, p.Y); inside {
		// calculate the minimum distance to the wall
		for _, v := range [...]int32{x, y, heatMapW - x, heatMapH - y} {
			if standardBrush.MaxRadius >= v {
				return
			}
		}
		switch program_view {
		case TEMPERATURE_VIEW:
			for _, l := range standardBrush.Points {
				elementArr[y+l[1]][x+l[0]].energy += heatCapacity[Water] * 5
			}
		case MATERIAL_VIEW:
			for _, l := range standardBrush.Points {
				elementArr[y+l[1]][x+l[0]].material = selected_material
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
				elementArr[y][x].material = r.Value.(material)
			}
		}
	case float32:
		for y := y1; y <= y2; y++ {
			for x := x1; x <= x2; x++ {
				elementArr[y][x].energy = r.Value.(float32)
			}
		}
	default:
		panic("Set_Rect recieved unsupported type")
	}
}
