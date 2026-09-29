package tui

import (
	"math"
	"strings"

	"via-terminal/internal/defs"
)

// Cells per 1u. Neighbouring keys share a border, so a 1u key is unitW+1
// characters wide on its own.
const unitW, unitH = 6, 3

const (
	up = 1 << iota
	down
	left
	right
)

var boxRunes = [16]rune{
	0:                        ' ',
	up | down:                '│',
	left | right:             '─',
	down | right:             '┌',
	down | left:              '┐',
	up | right:               '└',
	up | left:                '┘',
	up | down | right:        '├',
	up | down | left:         '┤',
	down | left | right:      '┬',
	up | left | right:        '┴',
	up | down | left | right: '┼',
}

type rect struct{ x0, y0, x1, y1 int }

// cellRects converts each key's main and second KLE rectangle to character
// cells, shifted so the board starts at 0,0.
func cellRects(keys []defs.Key) [][2]rect {
	minX, minY := math.Inf(1), math.Inf(1)
	for _, k := range keys {
		minX = min(minX, k.X, k.X+k.X2)
		minY = min(minY, k.Y, k.Y+k.Y2)
	}
	toRect := func(x, y, w, h float64) rect {
		return rect{
			int(math.Round((x - minX) * unitW)), int(math.Round((y - minY) * unitH)),
			int(math.Round((x + w - minX) * unitW)), int(math.Round((y + h - minY) * unitH)),
		}
	}
	rects := make([][2]rect, len(keys))
	for i, k := range keys {
		rects[i] = [2]rect{toRect(k.X, k.Y, k.W, k.H), toRect(k.X+k.X2, k.Y+k.Y2, k.W2, k.H2)}
	}
	return rects
}

// keyAt returns the index of the key drawn at character x,y, or -1.
func keyAt(keys []defs.Key, x, y int) int {
	for i, rs := range cellRects(keys) {
		for _, r := range rs {
			if x > r.x0 && x < r.x1 && y > r.y0 && y < r.y1 {
				return i
			}
		}
	}
	return -1
}

// render draws the keys as a box-drawing grid and shows key sel in reverse
// video (-1 for none). Legend lines longer than a key wrap inside it.
//
// Every character sits on a lattice point between cells; a point gets a
// border segment wherever the cells on either side belong to different keys.
// That gives shared borders, the right junction characters and L-shaped ISO
// Enter without special cases.
func render(keys []defs.Key, legend func(defs.Key) string, sel int) string {
	rects := cellRects(keys)
	var width, height int
	var owner [][]int
	for i, rs := range rects {
		for _, r := range rs {
			width, height = max(width, r.x1), max(height, r.y1)
			for len(owner) < height {
				owner = append(owner, nil)
			}
			for y := r.y0; y < r.y1; y++ {
				for len(owner[y]) < width {
					owner[y] = append(owner[y], 0)
				}
				for x := r.x0; x < r.x1; x++ {
					owner[y][x] = i + 1
				}
			}
		}
	}
	at := func(x, y int) int {
		if y < 0 || y >= len(owner) || x < 0 || x >= len(owner[y]) {
			return 0
		}
		return owner[y][x]
	}

	grid := make([][]rune, height+1)
	for y := range grid {
		grid[y] = make([]rune, width+1)
		for x := range grid[y] {
			var bits int
			if at(x-1, y-1) != at(x, y-1) {
				bits |= up
			}
			if at(x-1, y) != at(x, y) {
				bits |= down
			}
			if at(x-1, y-1) != at(x-1, y) {
				bits |= left
			}
			if at(x, y-1) != at(x, y) {
				bits |= right
			}
			grid[y][x] = boxRunes[bits]
		}
	}

	for i, k := range keys {
		r := rects[i][0]
		inner := r.x1 - r.x0 - 1
		if inner < 1 {
			continue
		}
		var lines [][]rune
		for _, text := range strings.Split(legend(k), "\n") {
			lines = append(lines, wrap([]rune(text), inner)...)
		}
		// Legends put the least important line first, so drop from the top.
		lines = lines[max(0, len(lines)-(r.y1-r.y0-1)):]
		for j, line := range lines {
			copy(grid[r.y0+1+j][r.x0+1+(inner-len(line))/2:], line)
		}
	}

	lines := make([]string, len(grid))
	for y, row := range grid {
		var b strings.Builder
		on := false
		for x, c := range row {
			s := sel + 1
			inside := sel >= 0 && at(x-1, y-1) == s && at(x, y-1) == s && at(x-1, y) == s && at(x, y) == s
			if inside && !on {
				b.WriteString("\x1b[7m")
			} else if !inside && on {
				b.WriteString("\x1b[27m")
			}
			on = inside
			b.WriteRune(c)
		}
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return strings.Join(lines, "\n")
}

// wrap breaks s into lines of at most width runes, preferring to break after
// a '_', ' ', '(' or ',' so "RGB_MOD" becomes "RGB_" and "MOD".
func wrap(s []rune, width int) [][]rune {
	var lines [][]rune
	for len(s) > width {
		cut := width
		for i := width - 1; i > 0; i-- {
			if strings.ContainsRune("_ (,", s[i]) {
				cut = i + 1
				break
			}
		}
		lines = append(lines, []rune(strings.TrimRight(string(s[:cut]), " ")))
		s = s[cut:]
	}
	return append(lines, s)
}
