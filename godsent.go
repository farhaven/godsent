/* TODO:
 * - abstract slides
 * - render text
 * - white background
 * - center everything
 */

package main

import (
	"log"
	"os"
	"io/ioutil"
	"bufio"

	"image"
	"image/draw"
	_ "image/png"

	"code.google.com/p/x-go-binding/ui"
	"code.google.com/p/x-go-binding/ui/x11"

	"code.google.com/p/freetype-go/freetype"
	_ "code.google.com/p/freetype-go/freetype/raster"
	"code.google.com/p/freetype-go/freetype/truetype"

	"github.com/tmc/fonts"
)

type Slide struct {
	Text string
	Image *image.Image
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
			fh, err := os.Open(line[1:])
			if err != nil {
				return nil, err
			}
			defer fh.Close()
			img, _, err := image.Decode(fh)
			if err != nil {
				return nil, err
			}
			slides = append(slides, Slide{"", &img})
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
func drawImage(img image.Image, target draw.Image) {
	draw.Draw(target, target.Bounds(), image.White, image.ZP, draw.Src)
	draw.Draw(target, target.Bounds(), img, image.ZP, draw.Src)
}

// Draws `text` to i using font `font`
func drawText(text string, font *truetype.Font, i draw.Image) error {
	ctx := freetype.NewContext()
	ctx.SetFont(font)
	ctx.SetFontSize(30)

	ctx.SetDst(i)
	ctx.SetClip(i.Bounds())
	ctx.SetSrc(image.Black)

	draw.Draw(i, i.Bounds(), image.White, image.ZP, draw.Src)
	if _, err := ctx.DrawString(text, freetype.Pt(100, 100)); err != nil {
		return err
	}

	return nil
}

func drawSlide(s Slide, font *truetype.Font, w ui.Window) {
	rgba := image.NewRGBA(w.Screen().Bounds())
	if s.Image != nil {
		drawImage(*s.Image, rgba)
	} else {
		drawText(s.Text, font, rgba)
	}
	draw.Draw(w.Screen(), w.Screen().Bounds(), rgba, image.ZP, draw.Src)
	w.FlushImage()
}

func loadFont(name string) (*truetype.Font, error) {
	fontReader, err := fonts.Load(name)
	if err != nil {
		return nil, err
	}
	fontBytes, err := ioutil.ReadAll(fontReader)
	if err != nil {
		return nil, err
	}
	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, err
	}

	return font, nil
}

func main() {
	slides, err := loadSlides("example")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("slides: %v", slides)

	font, err := loadFont("UbuntuMono-R")
	if err != nil {
		log.Fatalf(`can't load font: %s`, err)
	}

	w, err := x11.NewWindow()
	if err != nil {
		log.Fatalf("Can't create X window: %s", err)
	}
	defer w.Close()

	slideIdx := 0
	drawSlide(slides[slideIdx], font, w)

	for e := range w.EventChan() {
		switch e := e.(type) {
		case ui.KeyEvent:
			switch e.Key {
			case 'q':
				return
			case ' ':
				if slideIdx < len(slides) - 1 {
					slideIdx += 1
				}
			case 'b':
				if slideIdx > 0 {
					slideIdx -= 1
				}
			}
			// log.Printf(`key press: %v`, e.Key)
		case ui.ConfigEvent:
			log.Printf(`config event, new screen bounds: %v`, w.Screen().Bounds())
		case ui.MouseEvent:
			/* ignored */
		default:
			log.Printf(`unhandled event: %v`, e)
		}
		drawSlide(slides[slideIdx], font, w)
	}
}
