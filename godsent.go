/* TODO:
 * - abstract slides
 * - render text
 * - parse sent-style source file
 * - white background
 * - center everything
 */

package main

import (
	"log"
	_ "os"
	"io/ioutil"

	"image"
	"image/draw"
	_ "image/png"

	"code.google.com/p/x-go-binding/ui"
	"code.google.com/p/x-go-binding/ui/x11"

	"code.google.com/p/freetype-go/freetype"
	_ "code.google.com/p/freetype-go/freetype/raster"

	"github.com/tmc/fonts"
)

func main() {
	/*
	f, err := os.Open("nyan.png")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	m, _, err := image.Decode(f)
	if err != nil {
		log.Fatalln(err)
	}
	*/

	ctx := freetype.NewContext()
	fontReader, err := fonts.Load("Ubuntu-R")
	if err != nil {
		log.Fatalln(err)
	}
	fontBytes, err := ioutil.ReadAll(fontReader)
	if err != nil {
		log.Fatalln(err)
	}
	font, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Fatalln(err)
	}
	ctx.SetFont(font)
	ctx.SetFontSize(30)

	w, err := x11.NewWindow()
	if err != nil {
		log.Fatalln(err)
	}
	rgba := image.NewRGBA(w.Screen().Bounds())
	draw.Draw(rgba, rgba.Bounds(), image.White, image.ZP, draw.Src)
	ctx.SetDst(rgba)
	ctx.SetClip(rgba.Bounds())
	ctx.SetSrc(image.Black)

	if _, err := ctx.DrawString("Foo", freetype.Pt(100, 100)); err != nil {
		log.Fatalln(err)
	}

	draw.Draw(w.Screen(), w.Screen().Bounds(), rgba, image.ZP, draw.Src)

	// draw.Draw(w.Screen(), w.Screen().Bounds(), m, image.ZP, draw.Src)
	w.FlushImage()

	for e := range w.EventChan() {
		switch e := e.(type) {
		case ui.KeyEvent:
			if e.Key == ' ' || e.Key == 'q' {
				return
			}
			log.Printf(`key press: %v`, e.Key)
		}
	}
}
