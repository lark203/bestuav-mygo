package effort

import (
	"github.com/egoist/mygo/ui"

	"bestuav-mygo/internal/ui/canvas"
)

// The brain of Lucide 0.4, whose hemispheres close, so that they fill: 24
// units square, stroked 2 units wide with round caps and joins.
var brainPaths = canvas.ParsePaths(
	"M12 5a3 3 0 1 0-5.997.125 4 4 0 0 0-2.526 5.77 4 4 0 0 0 .556 6.588A4 4 0 1 0 12 18Z",
	"M12 5a3 3 0 1 1 5.997.125 4 4 0 0 1 2.526 5.77 4 4 0 0 1-.556 6.588A4 4 0 1 1 12 18Z",
	"M15 13a4.5 4.5 0 0 1-3-4 4.5 4.5 0 0 1-3 4",
	"M17.599 6.5a3 3 0 0 0 .399-1.375",
	"M6.003 5.125A3 3 0 0 0 6.401 6.5",
	"M3.477 10.896a4 4 0 0 1 .585-.396",
	"M19.938 10.5a4 4 0 0 1 .585.396",
	"M6 18a4 4 0 0 1-1.967-.516",
	"M19.967 17.484A4 4 0 0 1 18 18",
)

// The help button is a question mark in a circle of 1 DIP.
var helpFont = ui.Font{Family: "system-ui", Size: 8.8, Weight: 600, Antialiased: true}

// drawHelp draws the help button, centered on (cx, cy).
func drawHelp(p *ui.Painter, cx, cy float32, c ui.Color) {
	var ring ui.Path
	ring.Circle(cx, cy, helpRadius-0.5)
	p.StrokePath(&ring, 1, c)
	// The mark, centered in its line as CSS centers it.
	g := ui.Shape("?", helpFont)
	m := helpFont.Metrics()
	p.Glyphs(g, cx+helpMarkDX-g[0].Advance/2, cy+(m.Ascent-m.Descent)/2, c)
}

// drawBrain draws the brain filled with fill and outlined in stroke.
func drawBrain(p *ui.Painter, cx, cy, size float32, fill, stroke ui.Color) {
	p.FillPath(canvas.Icon(brainPaths[:2], cx, cy, size), fill)
	p.StrokePath(canvas.Icon(brainPaths, cx, cy, size), 2*size/24, stroke)
}
