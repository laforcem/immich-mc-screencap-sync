// Package assets provides embedded resources for the tray application.
package assets

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

// IconIdle returns a 16x16 green icon for the idle tray state.
func IconIdle() []byte { return iconForTray(color.RGBA{0x5a, 0xac, 0x2b, 0xff}) }

// IconSyncing returns a 16x16 yellow icon for the syncing tray state.
func IconSyncing() []byte { return iconForTray(color.RGBA{0xff, 0xd7, 0x00, 0xff}) }

// IconError returns a 16x16 red icon for the error tray state.
func IconError() []byte { return iconForTray(color.RGBA{0xcc, 0x33, 0x33, 0xff}) }

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

func iconICO(c color.RGBA) []byte {
	const (
		width      = 16
		height     = 16
		bpp        = 32
		headerSize = 6
		dirSize    = 16
		bmpHdrSize = 40
		pixelBytes = width * height * 4       // BGRA
		andBytes   = height * ((width + 31) / 32 * 4) // 1-bit mask, rows padded to 4 bytes
		dataSize   = bmpHdrSize + pixelBytes + andBytes
		dataOffset = headerSize + dirSize
	)

	var buf bytes.Buffer
	buf.Grow(dataOffset + dataSize)

	// ICO header
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: ICO
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // count

	// Directory entry
	buf.WriteByte(width)              // width
	buf.WriteByte(height)             // height
	buf.WriteByte(0)                  // color palette count
	buf.WriteByte(0)                  // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))          // color planes
	binary.Write(&buf, binary.LittleEndian, uint16(bpp))        // bits per pixel
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))   // image data size
	binary.Write(&buf, binary.LittleEndian, uint32(dataOffset)) // image data offset

	// BITMAPINFOHEADER
	binary.Write(&buf, binary.LittleEndian, uint32(bmpHdrSize))    // header size
	binary.Write(&buf, binary.LittleEndian, int32(width))          // width
	binary.Write(&buf, binary.LittleEndian, int32(height*2))       // height (doubled for ICO)
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // planes
	binary.Write(&buf, binary.LittleEndian, uint16(bpp))           // bit count
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // compression
	binary.Write(&buf, binary.LittleEndian, uint32(pixelBytes+andBytes)) // image size
	binary.Write(&buf, binary.LittleEndian, int32(0))              // x ppm
	binary.Write(&buf, binary.LittleEndian, int32(0))              // y ppm
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // colors used
	binary.Write(&buf, binary.LittleEndian, uint32(0))             // colors important

	// Pixel data (BGRA, bottom-up row order)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}

	// AND mask (all zeros = fully opaque)
	andRow := make([]byte, (width+31)/32*4)
	for y := 0; y < height; y++ {
		buf.Write(andRow)
	}

	return buf.Bytes()
}
