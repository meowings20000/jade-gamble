
from PIL import Image
from collections import deque
src = r"C:\jade-gamble\frontend\catcoin.png"
im = Image.open(src).convert("RGBA")
w, h = im.size
px = im.load()
def isbg(c):
    r, g, b = c[0], c[1], c[2]
    return (max(r,g,b) - min(r,g,b)) < 28 and min(r,g,b) > 180
seen = set(); q = deque()
for x in range(w):
    q.append((x,0)); q.append((x,h-1))
for y in range(h):
    q.append((0,y)); q.append((w-1,y))
while q:
    x, y = q.popleft()
    if (x,y) in seen or x < 0 or y < 0 or x >= w or y >= h:
        continue
    if not isbg(px[x,y]):
        continue
    seen.add((x,y))
    px[x,y] = (0,0,0,0)
    q.extend([(x+1,y),(x-1,y),(x,y+1),(x,y-1)])
# 邊緣柔化：貼著透明區的半白像素調低 alpha
for (x,y) in list(seen):
    for dx,dy in ((1,0),(-1,0),(0,1),(0,-1)):
        nx,ny = x+dx, y+dy
        if 0 <= nx < w and 0 <= ny < h and (nx,ny) not in seen:
            r,g,b,a = px[nx,ny]
            if isbg((r,g,b)):
                px[nx,ny] = (r,g,b,90)
# 裁掉多餘空白
bbox = im.getbbox()
if bbox:
    im = im.crop(bbox)
# 統一放大到 128x128（保持比例）
side = max(im.size)
canvas = Image.new("RGBA", (side, side), (0,0,0,0))
canvas.paste(im, ((side - im.size[0])//2, (side - im.size[1])//2), im)
canvas = canvas.resize((128,128), Image.LANCZOS)
canvas.save(r"C:\jade-gamble\frontend\catcoin.png")
# 檢查
chk = Image.open(r"C:\jade-gamble\frontend\catcoin.png").convert("RGBA")
alpha = chk.getchannel("A")
corner = alpha.getpixel((2,2)); center = alpha.getpixel((64,64))
print("尺寸:", chk.size, "| 角落 alpha:", corner, "(0=透明) | 中心 alpha:", center, "(255=不透明)")
