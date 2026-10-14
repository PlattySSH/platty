package vt

func (t *Terminal) reflow(cols, rows int, cur *cursor) {
	b := t.primary
	oldCols := t.cols

	last := cur.y

	for y := len(b.lines) - 1; y > last; y-- {
		if !lineEmpty(b.lines[y]) {
			last = y
			break
		}
	}

	var phys []line

	for i := 0; i < t.sb.len(); i++ {
		phys = append(phys, *t.sb.at(i))
	}

	curPhys := len(phys) + cur.y
	phys = append(phys, b.lines[:last+1]...)

	type logical struct {
		cells []Cell
	}
	var logicals []logical
	curLogical, curOffset := -1, 0
	var acc []Cell

	for i, l := range phys {
		if i == curPhys {
			curLogical = len(logicals)
			curOffset = len(acc) + cur.x
		}

		cells := l.cells

		if len(cells) > oldCols {
			cells = cells[:oldCols]
		}

		acc = append(acc, cells...)

		if !l.wrapped || i == len(phys)-1 {
			keep := -1

			if curLogical == len(logicals) {
				keep = curOffset
			}

			logicals = append(logicals, logical{cells: trimBlank(acc, keep)})
			acc = nil
		}
	}

	var out []line
	curLine, curX := 0, 0

	for li, lg := range logicals {
		cells := lg.cells
		var widths []int

		if len(cells) == 0 {
			out = append(out, newLine(cols, DefaultColor))
			widths = append(widths, 0)
		}

		for len(cells) > 0 {
			n := min(cols, len(cells))

			if n < len(cells) && n > 1 && cells[n-1].Width == 2 {
				n--
			}

			l := newLine(cols, DefaultColor)
			copy(l.cells, cells[:n])
			cells = cells[n:]
			l.wrapped = len(cells) > 0
			out = append(out, l)
			widths = append(widths, n)
		}

		if li != curLogical {
			continue
		}

		first := len(out) - len(widths)
		curLine, curX = first, curOffset

		for i, w := range widths {
			if curX < w || i == len(widths)-1 {
				break
			}

			curX -= w
			curLine++
		}

		switch {
		case curX == cols:
			curX = cols - 1
		case curX > cols:
			curLine++
			curX -= cols

			if curLine >= len(out) {
				out = append(out, newLine(cols, DefaultColor))
			}
		}
	}

	startRow := max(len(out)-rows, 0)
	startRow = min(startRow, curLine)

	t.sb.clear()

	for _, l := range out[:startRow] {
		t.sb.push(l)
	}

	screen := make([]line, rows)

	for i := range screen {
		if startRow+i < len(out) {
			screen[i] = out[startRow+i]
		} else {
			screen[i] = newLine(cols, DefaultColor)
		}
	}

	b.lines = screen

	cur.y = min(curLine-startRow, rows-1)
	cur.x = min(curX, cols-1)
	cur.pendingWrap = false
}

func lineEmpty(l line) bool {
	for _, c := range l.cells {
		if c.Rune != 0 || c.Bg != DefaultColor {
			return false
		}
	}

	return true
}

func trimBlank(cells []Cell, keep int) []Cell {
	n := len(cells)

	for n > 0 && cells[n-1].Rune == 0 && cells[n-1].Bg == DefaultColor && cells[n-1].Attr == 0 {
		n--
	}

	if keep >= 0 {
		n = max(n, min(keep, len(cells)))
	}

	out := make([]Cell, n)
	copy(out, cells[:n])
	return out
}
