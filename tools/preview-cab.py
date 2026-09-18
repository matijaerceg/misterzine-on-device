# Assembles the harness's cab-*.png frames into sheet.png and a 2x GIF.
import glob
import sys

from PIL import Image

src, out = sys.argv[1], sys.argv[2]
fs = sorted(glob.glob(src + '/cab-*.png'))
cols = 6
rows = (len(fs) + cols - 1) // cols
sheet = Image.new('RGB', (cols * 320, rows * 240))
frames = []
for i, f in enumerate(fs):
    im = Image.open(f).convert('RGB')
    sheet.paste(im, ((i % cols) * 320, (i // cols) * 240))
    frames.append(im.resize((640, 480), Image.NEAREST))
sheet.save(out + '/sheet.png')
frames[0].save(out + '/launch-cab.gif', save_all=True, append_images=frames[1:], duration=33, loop=0)
print(len(fs), 'frames')
