"""Generate assets/wingman.ico — wing + checkmark, winget-blue on dark."""

from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parent
OUT = ROOT / "assets" / "wingman.ico"

BG = (13, 17, 23)  # #0d1117
BLUE = (0, 164, 239)  # winget-adjacent #00a4ef
GREEN = (63, 185, 80)  # #3fb950


def draw_icon(size: int) -> Image.Image:
    img = Image.new("RGBA", (size, size), (*BG, 255))
    d = ImageDraw.Draw(img)
    m = size / 256

    # Rounded tile background
    pad = int(18 * m)
    d.rounded_rectangle(
        (pad, pad, size - pad, size - pad),
        radius=int(36 * m),
        fill=(22, 27, 34),
        outline=(48, 54, 61),
        width=max(1, int(3 * m)),
    )

    # Stylized wing (paper-plane / wing shape)
    wing = [
        (int(72 * m), int(148 * m)),
        (int(200 * m), int(72 * m)),
        (int(188 * m), int(118 * m)),
        (int(128 * m), int(138 * m)),
        (int(168 * m), int(188 * m)),
        (int(96 * m), int(168 * m)),
    ]
    d.polygon(wing, fill=BLUE)
    d.polygon(
        [
            (int(88 * m), int(132 * m)),
            (int(176 * m), int(88 * m)),
            (int(168 * m), int(112 * m)),
            (int(120 * m), int(128 * m)),
        ],
        fill=(56, 189, 248),
    )

    # Checkmark badge
    cx, cy = int(188 * m), int(178 * m)
    r = int(34 * m)
    d.ellipse((cx - r, cy - r, cx + r, cy + r), fill=GREEN)
    d.line(
        [
            (int(cx - 14 * m), int(cy + 2 * m)),
            (int(cx - 2 * m), int(cy + 14 * m)),
            (int(cx + 18 * m), int(cy - 12 * m)),
        ],
        fill=BG,
        width=max(2, int(7 * m)),
        joint="curve",
    )

    return img


def main() -> None:
    OUT.parent.mkdir(parents=True, exist_ok=True)
    sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)]
    images = [draw_icon(s[0]) for s in sizes]
    images[0].save(
        OUT,
        format="ICO",
        sizes=sizes,
        append_images=images[1:],
    )
    print(f"Wrote {OUT}")


if __name__ == "__main__":
    main()
