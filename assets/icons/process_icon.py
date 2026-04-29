#!/usr/bin/env python3
"""
Apply pixel-art border treatment to an icon PNG, with optional colorization.

Usage: python process_icon.py <input.png> <output.png> [#RRGGBB] [factor]

  #RRGGBB  Optional — recolors all pixels to this color before treatment
  factor   Border darkening factor, default 0.70

Semi-transparent edge pixels (5 < alpha < 250) are darkened by `factor`
and made fully opaque, giving a crisp pixel-art outline.
"""
import sys
from PIL import Image
import numpy as np

if len(sys.argv) < 3:
    sys.exit(f"Usage: {sys.argv[0]} <input.png> <output.png> [#RRGGBB] [factor]")

input_path = sys.argv[1]
output_path = sys.argv[2]
color_hex = None
factor = 0.70

for arg in sys.argv[3:]:
    if arg.startswith('#'):
        color_hex = arg
    else:
        factor = float(arg)

img = Image.open(input_path).convert("RGBA")
data = np.array(img, dtype=np.float32)

if color_hex:
    data[:, :, 0] = int(color_hex[1:3], 16)
    data[:, :, 1] = int(color_hex[3:5], 16)
    data[:, :, 2] = int(color_hex[5:7], 16)

r, g, b, a = data[:, :, 0], data[:, :, 1], data[:, :, 2], data[:, :, 3]
border = (a > 5) & (a < 250)
interior = a >= 250

out = data.copy()
out[border, 0] = np.clip(r[border] * factor, 0, 255)
out[border, 1] = np.clip(g[border] * factor, 0, 255)
out[border, 2] = np.clip(b[border] * factor, 0, 255)
out[border, 3] = 255
out[interior, 3] = 255

Image.fromarray(out.astype(np.uint8), "RGBA").save(output_path)
print(f"Wrote {output_path}")
