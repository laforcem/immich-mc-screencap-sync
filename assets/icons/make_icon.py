from PIL import Image
import numpy as np

img = Image.open("base_32.png").convert("RGBA")
data = np.array(img, dtype=np.float32)

r, g, b, a = data[:,:,0], data[:,:,1], data[:,:,2], data[:,:,3]

border = (a > 5) & (a < 250)
interior = a >= 250

variants = [
    ("v1_subtle", 0.70),
    ("v2_medium", 0.50),
    ("v3_strong", 0.30),
]

for name, factor in variants:
    out = data.copy()
    out[border, 0] = np.clip(r[border] * factor, 0, 255)
    out[border, 1] = np.clip(g[border] * factor, 0, 255)
    out[border, 2] = np.clip(b[border] * factor, 0, 255)
    out[border, 3] = 255
    out[interior, 3] = 255

    small = Image.fromarray(out.astype(np.uint8), "RGBA")
    small.save(f"immich_32_{name}.png")

    # Scale up for preview with nearest-neighbour (no blurring)
    preview = small.resize((512, 512), Image.NEAREST)
    preview.save(f"/mnt/c/Users/malc/Downloads/PREVIEW_32_{name}.png")
    print(f"PREVIEW_32_{name}.png")
