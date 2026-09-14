# Render the existing outlined vector logo as a transparent purple asset.
# Run from the repository root; requires Pillow and svgpathtools.
from pathlib import Path
import xml.etree.ElementTree as ET
from svgpathtools import parse_path
from PIL import Image, ImageDraw, ImageChops
source = Path('internal/app/startup_logo.svg')
paths = [parse_path(node.attrib['d']) for node in ET.parse(source).iter('{http://www.w3.org/2000/svg}path')]
# The two separate paths are the counters (holes) inside the e letters.
path = paths[0]
x0,x1,y0,y1 = path.bbox()
scale = 1200/(x1-x0)
size = (1200, round((y1-y0)*scale))
mask = Image.new('1', size)
outer = None
for sub in [sub for path in paths for sub in path.continuous_subpaths()]:
    pts=[]
    for seg in sub:
        n=max(2, int(seg.length()*scale/2))
        pts.extend(((seg.point(i/n).real-x0)*scale,(y1-seg.point(i/n).imag)*scale) for i in range(n))
    part=Image.new('1',size)
    ImageDraw.Draw(part).polygon(pts,fill=1)
    if outer is None: outer=part.copy()
    mask=ImageChops.logical_xor(mask,part)
img=Image.new('RGBA',size,'#1d1330')
img.paste((162,147,199,255),(0,0),mask.convert('L'))
img.putalpha(outer.convert('L'))
# Rasterize once at the actual display size; the app never resizes this asset.
img.resize((160,106),Image.Resampling.LANCZOS).save('internal/app/startup_logo.png')
