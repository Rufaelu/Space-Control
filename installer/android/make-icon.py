#!/usr/bin/env python3
"""Draws the Android launcher icon (an adaptive icon: background + foreground).

    python3 installer/android/make-icon.py      # needs Pillow

Writes res/drawable-nodpi/ic_bg.png and ic_fg.png next to this script.
"""
import math, os, random
from PIL import Image, ImageDraw, ImageFilter

S = 432  # 108dp at xxxhdpi; the launcher shows the middle 66dp or so
HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "res", "drawable-nodpi")
os.makedirs(OUT, exist_ok=True)
random.seed(7)

# background: deep space with a purple-blue nebula and stars
bg = Image.new("RGB", (S, S), (6, 8, 22))
glow = Image.new("RGB", (S, S), (0, 0, 0))
d = ImageDraw.Draw(glow)
for cx, cy, r, col in [(120, 110, 210, (70, 20, 110)), (330, 320, 230, (20, 40, 120)), (300, 90, 140, (90, 30, 90))]:
    d.ellipse((cx - r, cy - r, cx + r, cy + r), fill=col)
glow = glow.filter(ImageFilter.GaussianBlur(70))
bg = Image.blend(bg, glow, 0.85)
d = ImageDraw.Draw(bg)
for _ in range(90):
    x, y = random.randrange(S), random.randrange(S)
    b = random.choice([120, 160, 200, 255])
    r = random.choice([1, 1, 1, 2])
    d.ellipse((x - r, y - r, x + r, y + r), fill=(b, b, min(255, b + 30)))
bg.save(os.path.join(OUT, "ic_bg.png"))

# foreground: the player's fighter from above, nose up, engines lit
fg = Image.new("RGBA", (S, S), (0, 0, 0, 0))
c = S / 2
k = 12.5  # model units to pixels; the ship spans about 15 x 16 units

def P(x, z):  # model x (right) and z (back) to pixels, ship centred
    return (c + x * k, c + (z + 1.9) * k)

def box(x0, z0, x1, z1):
    (a, b), (e, f) = P(x0, z0), P(x1, z1)
    return [min(a, e), min(b, f), max(a, e), max(b, f)]

engines = Image.new("RGBA", (S, S), (0, 0, 0, 0))
de = ImageDraw.Draw(engines)
for sx in (-1, 1):
    x, y = P(sx * 1.75, 7.6)
    de.ellipse((x - 30, y - 30, x + 30, y + 46), fill=(60, 190, 255, 255))
engines = engines.filter(ImageFilter.GaussianBlur(14))
fg.alpha_composite(engines)

d = ImageDraw.Draw(fg)
hull, dark, accent = (205, 210, 232, 255), (32, 36, 52, 255), (20, 150, 255, 255)
for sx in (-1, 1):  # wings and wingtip cannons
    d.polygon([P(sx * 1.2, -1.2), P(sx * 7.5, 2.4), P(sx * 7.5, 4.8), P(sx * 1.2, 4.2)], fill=hull)
    d.polygon([P(sx * 2.4, 3.4), P(sx * 7.0, 4.2), P(sx * 7.0, 4.5), P(sx * 2.4, 3.9)], fill=accent)
    d.rectangle(box(sx * 7.5 - 0.4, -0.6, sx * 7.5 + 0.4, 5.5), fill=dark)
    d.polygon([P(sx * 0.9, 2.0), P(sx * 2.4, 4.6), P(sx * 2.4, 5.9), P(sx * 0.9, 5.0)], fill=(170, 176, 200, 255))  # tail fins
    d.rectangle(box(sx * 1.75 - 0.45, 1.8, sx * 1.75 + 0.45, 6.2), fill=dark)
# fuselage and canopy
d.polygon([P(0, -9.5), P(0.8, -5.5), P(1.5, 2.5), P(1.05, 5.2), P(-1.05, 5.2), P(-1.5, 2.5), P(-0.8, -5.5)], fill=hull)
d.polygon([P(0, -4.8), P(0.6, -3.0), P(0.55, 0.9), P(-0.55, 0.9), P(-0.6, -3.0)], fill=(25, 140, 255, 255))
d.rectangle(box(-0.12, 1.2, 0.12, 4.2), fill=accent)
# muzzle flashes
for sx in (-1, 1):
    x, y = P(sx * 7.5, -1.0)
    d.ellipse((x - 7, y - 7, x + 7, y + 7), fill=(170, 235, 255, 255))
fg.save(os.path.join(OUT, "ic_fg.png"))
print("wrote", OUT)
