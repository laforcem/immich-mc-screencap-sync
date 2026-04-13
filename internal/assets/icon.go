// Package assets provides embedded resources for the tray application.
package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// IconIdle returns a 16x16 green PNG icon for the idle tray state.
func IconIdle() []byte { return iconPNG(color.RGBA{0x5a, 0xac, 0x2b, 0xff}) }

// IconSyncing returns a 16x16 yellow PNG icon for the syncing tray state.
func IconSyncing() []byte { return iconPNG(color.RGBA{0xff, 0xd7, 0x00, 0xff}) }

// IconError returns a 16x16 red PNG icon for the error tray state.
func IconError() []byte { return iconPNG(color.RGBA{0xcc, 0x33, 0x33, 0xff}) }

func iconPNG(c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
