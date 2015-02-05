package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"

	"image"
	_ "image/png"

	"github.com/scottferg/Go-SDL/sdl"
	"github.com/ungerik/go-cairo"
)

type Command int

const (
	FirstSlide = iota
	NextSlide
	PrevSlide
	LastSlide
	ToggleFullscreen
	Quit
)

type Slide struct {
	Text  string
	Image *cairo.Surface
}

func loadSlides(fname string) ([]Slide, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var slides []Slide
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			line = " "
		}
		switch line[0] {
		case '@':
			// image slide
			fh, err := os.Open(line[1:])
			if err != nil {
				return nil, err
			}
			img, _, err := image.Decode(fh)
			if err != nil {
				return nil, err
			}
			slides = append(slides, Slide{"", cairo.NewSurfaceFromImage(img)})
		case '#':
			/* comment slide */
			log.Printf(`comment: %s`, line)
		default:
			/* regular text slide */
			slides = append(slides, Slide{line, nil})
		}
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return slides, nil
}

// Draws `img` to `target`
func drawImage(src *cairo.Surface, tgt *sdl.Surface, zoom bool) error {
	var srcrect sdl.Rect
	var dstrect sdl.Rect

	tgt.GetClipRect(&dstrect)

	if zoom {
		tgtwidth := float64(dstrect.W) * 0.9
		tgtheight := float64(dstrect.H) * 0.9

		sx := tgtwidth / float64(src.GetWidth())
		sy := tgtheight / float64(src.GetHeight())

		tgtwidth = float64(src.GetWidth()) * math.Min(sx, sy)
		tgtheight = float64(src.GetHeight()) * math.Min(sx, sy)

		newsrc := cairo.NewSurface(src.GetFormat(), int(tgtwidth), int(tgtheight))

		newsrc.SetSourceRGB(1, 1, 1)
		newsrc.Rectangle(0, 0, tgtwidth, tgtheight)
		newsrc.Fill()

		newsrc.Scale(math.Min(sx, sy), math.Min(sx, sy))
		newsrc.SetOperator(cairo.OPERATOR_SOURCE)
		newsrc.SetSourceSurface(src, 0, 0)

		newsrc.Paint()

		src = newsrc
	}

	surf := sdl.CreateRGBSurfaceFrom(src.GetData(),
		src.GetWidth(), src.GetHeight(),
		32 /* bpp */, src.GetWidth()*4, /* pitch */
		0x00FF0000, /* rmask */
		0x0000FF00, /* gmask */
		0x000000FF, /* bmask */
		0 /* amask */)

	surf.GetClipRect(&srcrect)

	dstrect.X = int16((dstrect.W / 2) - (srcrect.W / 2))
	dstrect.Y = int16((dstrect.H / 2) - (srcrect.H / 2))
	if zoom {
		srcrect.X += 1
		srcrect.Y += 1
	}
	dstrect.W = srcrect.W
	dstrect.H = srcrect.H

	if tgt.Blit(&dstrect, surf, &srcrect) != 0 {
		return fmt.Errorf(`%s`, sdl.GetError())
	}

	return nil
}

// Draws `text` to s using the largest possible font in `fonts`
func drawText(text string, s *sdl.Surface) error {
	var r sdl.Rect
	s.GetClipRect(&r)

	tgtheight := float64(r.H) * 0.9
	tgtwidth := float64(r.W) * 0.9

	surf := cairo.NewSurface(cairo.FORMAT_ARGB32, int(tgtwidth), int(tgtheight))
	surf.SetSourceRGB(1, 1, 1)
	surf.Rectangle(0, 0, float64(surf.GetWidth()), float64(surf.GetHeight()))
	surf.Fill()

	surf.SetSourceRGB(0, 0, 0)
	surf.SelectFontFace("Ubuntu Mono", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_NORMAL)
	size := float64(0)
	var ext *cairo.TextExtents
	for sz := float64(10); sz <= 800; sz += 10 {
		surf.SetFontSize(sz)
		ext = surf.TextExtents(text)
		if ext.Xadvance < tgtwidth && ext.Yadvance < tgtheight {
			size = sz
		} else {
			break
		}
	}
	surf.MoveTo(0, float64(surf.GetHeight()/2)-((ext.Height/2)+ext.Ybearing))
	surf.SetFontSize(size)
	surf.ShowText(text)

	return drawImage(surf, s, false)
}

func colorToUint(c sdl.Color) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.Unused)
}

func drawSlide(s Slide, surf *sdl.Surface) {
	var dstrect sdl.Rect
	surf.GetClipRect(&dstrect)
	surf.FillRect(&dstrect, colorToUint(sdl.Color{255, 255, 255, 255}))
	if s.Image != nil {
		drawImage(s.Image, surf, true)
	} else {
		drawText(s.Text, surf)
	}
	surf.Flip()
}

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Key(k.Sym))
}

func handleCommands(commands chan Command, done chan bool, slides []Slide) {
	defer func() {
		done <- true
	}()

	surf := sdl.GetVideoSurface()
	slideIdx := 0
	drawSlide(slides[slideIdx], surf)

	for cmd := range commands {
		switch cmd {
		case FirstSlide:
			slideIdx = 0
		case LastSlide:
			slideIdx = len(slides) - 1
		case NextSlide:
			if slideIdx < len(slides)-1 {
				slideIdx += 1
			}
		case PrevSlide:
			if slideIdx > 0 {
				slideIdx -= 1
			}
		case ToggleFullscreen:
			sdl.WM_ToggleFullScreen(surf)
		case Quit:
			return
		}
		drawSlide(slides[slideIdx], surf)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf(`Usage: %s slideset`, os.Args[0])
	}

	var slides []Slide
	for _, a := range os.Args[1:] {
		s, err := loadSlides(a)
		if err != nil {
			log.Fatalf(`can't load slides: %s`, err)
		}
		slides = append(slides, s...)
	}

	if sdl.Init(sdl.INIT_VIDEO) != 0 {
		log.Fatalf(`couldn't init sdl video: %s`, sdl.GetError())
	}
	defer sdl.Quit()
	sdl.WM_SetCaption("GodSent", "")   // title of presentation?
	sdl.SetVideoMode(1024, 768, 32, 0) // sdl.FULLSCREEN)

	done := make(chan bool)
	commandchan := make(chan Command)
	go handleCommands(commandchan, done, slides)

eventloop:
	for e := range sdl.Events {
		switch e := e.(type) {
		default:
			log.Printf(`event %T`, e)
		case sdl.MouseMotionEvent, sdl.ActiveEvent:
			/* ignore */
		case sdl.KeyboardEvent:
			if e.Type != sdl.KEYDOWN {
				break
			}
			switch getNameFromKeysym(e.Keysym) {
			case `space`:
				commandchan <- NextSlide
			case `b`:
				commandchan <- PrevSlide
			case `home`:
				commandchan <- FirstSlide
			case `end`:
				commandchan <- LastSlide
			case `f`:
				commandchan <- ToggleFullscreen
			case `q`:
				commandchan <- Quit
				break eventloop
			default:
				log.Printf(`key press: %v %s`, e.Type, getNameFromKeysym(e.Keysym))
			}
		case sdl.QuitEvent:
			commandchan <- Quit
			break eventloop
		}
	}

	// Wait for command handler to quit
	<-done
}
