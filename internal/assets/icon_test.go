package assets

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func TestIconICO_ValidHeader(t *testing.T) {
	ico := iconICO(color.RGBA{0x5a, 0xac, 0x2b, 0xff})

	if len(ico) < 22 {
		t.Fatalf("ICO too short: got %d bytes, want at least 22", len(ico))
	}

	// ICO header: reserved(2) + type(2) + count(2)
	reserved := binary.LittleEndian.Uint16(ico[0:2])
	imgType := binary.LittleEndian.Uint16(ico[2:4])
	count := binary.LittleEndian.Uint16(ico[4:6])

	if reserved != 0 {
		t.Errorf("reserved = %d, want 0", reserved)
	}
	if imgType != 1 {
		t.Errorf("image type = %d, want 1 (ICO)", imgType)
	}
	if count != 1 {
		t.Errorf("image count = %d, want 1", count)
	}

	// Directory entry: width(1) + height(1) at offset 6
	width := ico[6]
	height := ico[7]
	if width != 16 {
		t.Errorf("width = %d, want 16", width)
	}
	if height != 16 {
		t.Errorf("height = %d, want 16", height)
	}
}

func TestIconICO_BitmapInfoHeader(t *testing.T) {
	ico := iconICO(color.RGBA{0xff, 0xd7, 0x00, 0xff})

	// Data offset is at directory entry bytes 18-21 (offset 18 in file)
	if len(ico) < 22 {
		t.Fatalf("ICO too short: got %d bytes", len(ico))
	}
	dataOffset := binary.LittleEndian.Uint32(ico[18:22])

	if len(ico) < int(dataOffset)+40 {
		t.Fatalf("ICO too short for BMP header at offset %d: got %d bytes", dataOffset, len(ico))
	}

	bmpHeader := ico[dataOffset:]
	headerSize := binary.LittleEndian.Uint32(bmpHeader[0:4])
	bmpWidth := int32(binary.LittleEndian.Uint32(bmpHeader[4:8]))
	bmpHeight := int32(binary.LittleEndian.Uint32(bmpHeader[8:12]))
	planes := binary.LittleEndian.Uint16(bmpHeader[12:14])
	bitCount := binary.LittleEndian.Uint16(bmpHeader[14:16])

	if headerSize != 40 {
		t.Errorf("BMP header size = %d, want 40", headerSize)
	}
	if bmpWidth != 16 {
		t.Errorf("BMP width = %d, want 16", bmpWidth)
	}
	// ICO BMPs use double height (image + AND mask)
	if bmpHeight != 32 {
		t.Errorf("BMP height = %d, want 32 (double for ICO)", bmpHeight)
	}
	if planes != 1 {
		t.Errorf("planes = %d, want 1", planes)
	}
	if bitCount != 32 {
		t.Errorf("bit count = %d, want 32", bitCount)
	}
}

func TestIconICO_PixelData(t *testing.T) {
	c := color.RGBA{0x5a, 0xac, 0x2b, 0xff}
	ico := iconICO(c)

	dataOffset := binary.LittleEndian.Uint32(ico[18:22])
	// Pixel data starts after 40-byte BMP header
	pixelStart := int(dataOffset) + 40
	// 16x16 BGRA = 1024 bytes
	if len(ico) < pixelStart+1024 {
		t.Fatalf("ICO too short for pixel data: got %d bytes", len(ico))
	}

	// Check first pixel (bottom-left in BMP order) is BGRA
	b := ico[pixelStart]
	g := ico[pixelStart+1]
	r := ico[pixelStart+2]
	a := ico[pixelStart+3]

	if r != c.R || g != c.G || b != c.B || a != c.A {
		t.Errorf("pixel BGRA = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
			b, g, r, a, c.B, c.G, c.R, c.A)
	}
}
