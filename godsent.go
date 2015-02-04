package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/scottferg/Go-SDL/sdl"
	"github.com/scottferg/Go-SDL/ttf"
)

type Command int

const (
	NextSlide = iota
	PrevSlide
	ToggleFullscreen
	Quit
)

type Slide struct {
	Text  string
	Image *sdl.Surface
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
		switch line[0] {
		case '@':
			/* image slide */
			img := sdl.Load(line[1:])
			if img == nil {
				return nil, fmt.Errorf(`%s`, sdl.GetError())
			}
			slides = append(slides, Slide{"", img})
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
func drawImage(src *sdl.Surface, tgt *sdl.Surface) error {
	var srcrect sdl.Rect
	var dstrect sdl.Rect

	src.GetClipRect(&srcrect)
	tgt.GetClipRect(&dstrect)

	dstrect.X = int16((dstrect.W / 2) - (srcrect.W / 2))
	dstrect.Y = int16((dstrect.H / 2) - (srcrect.H / 2))
	dstrect.W = srcrect.W
	dstrect.H = srcrect.H

	if tgt.Blit(&dstrect, src, &srcrect) != 0 {
		return fmt.Errorf(`%s`, sdl.GetError())
	}

	return nil
}

// Draws `text` to s using the largest possible font in `fonts`
func drawText(text string, fonts []*ttf.Font, s *sdl.Surface) error {
	var r sdl.Rect
	s.GetClipRect(&r)

	tgtheight := int(float32(r.H) * 0.9)
	tgtwidth  := int(float32(r.W) * 0.9)

	var font *ttf.Font

	for _, fnt := range fonts {
		width, height, err := fnt.SizeUTF8(text)
		if err != 0 {
			return fmt.Errorf(`%s`, sdl.GetError())
		}
		if width < tgtwidth && height < tgtheight {
			font = fnt
		}
	}

	/* If even the smallest font doesn't fit, select it anyway */
	if font == nil {
		font = fonts[len(fonts) - 1]
	}

	ts := ttf.RenderUTF8_Blended(font, text, sdl.Color{0, 0, 0, 0})
	return drawImage(ts, s)
}

func colorToUint(c sdl.Color) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.Unused)
}

func drawSlide(s Slide, fonts []*ttf.Font, surf *sdl.Surface) {
	var dstrect sdl.Rect
	surf.GetClipRect(&dstrect)
	surf.FillRect(&dstrect, colorToUint(sdl.Color{255, 255, 255, 255}))
	if s.Image != nil {
		drawImage(s.Image, surf)
	} else {
		drawText(s.Text, fonts, surf)
	}
	surf.Flip()
}

func loadFont(name string) ([]*ttf.Font, error) {
	if ttf.Init() != 0 {
		return nil, fmt.Errorf(`couldn't init ttf`)
	}

	var fonts []*ttf.Font

	for sz := 10; sz <= 200; sz += 10 {
		font := ttf.OpenFont(name, sz)
		if font == nil {
			return nil, fmt.Errorf(`couldn't load font "%s" with size=%d`, name, sz)
		}
		fonts = append(fonts, font)
	}

	return fonts, nil
}

func getNameFromKeysym(k sdl.Keysym) string {
	return sdl.GetKeyName(sdl.Key(k.Sym))
}

func handleCommands(commands chan Command, done chan bool, fonts []*ttf.Font, slides []Slide) {
	defer func() {
		done <- true
	}()

	surf := sdl.GetVideoSurface()
	slideIdx := 0
	drawSlide(slides[slideIdx], fonts, surf)

	for cmd := range commands {
		switch cmd {
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
		drawSlide(slides[slideIdx], fonts, surf)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf(`Usage: %s slideset`, os.Args[0])
	}

	slides, err := loadSlides(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	fonts, err := loadFont("UbuntuMono-R.ttf")
	if err != nil {
		log.Fatalf(`can't load font: %s`, err)
	}

	if sdl.Init(sdl.INIT_VIDEO) != 0 {
		log.Fatalf(`couldn't init sdl video: %s`, sdl.GetError())
	}
	defer sdl.Quit()
	sdl.WM_SetCaption("GodSent", "") // title of presentation?
	vi := sdl.GetVideoInfo()
	sdl.SetVideoMode(int(vi.Current_w/2), int(vi.Current_h/2), 32, 0) // sdl.FULLSCREEN)

	done := make(chan bool)
	commandchan := make(chan Command)
	go handleCommands(commandchan, done, fonts, slides)

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
