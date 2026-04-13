//go:build !windows

package assets

import "image/color"

func iconForTray(c color.RGBA) []byte { return iconPNG(c) }
