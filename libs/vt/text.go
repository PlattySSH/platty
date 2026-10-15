package vt

import (
	"strings"
	"unicode"
)

type Pos struct {
	Line, Col int
}

func (p Pos) Before(q Pos) bool {
	return p.Line < q.Line || (p.Line == q.Line && p.Col < q.Col)
}

type Match struct {
	Start, End Pos
}

func (m Match) Contains(p Pos) bool {
	return !p.Before(m.Start) && !m.End.Before(p)
}

func (t *Terminal) visibleSB() int {
	if t.screen != t.primary {
		return 0
	}

	return t.sb.len()
}

func (t *Terminal) lineAt(id int) (*line, bool) {
	rel := id - t.sb.dropped
	sb := t.visibleSB()

	if rel < 0 || rel >= sb+t.rows {
		return nil, false
	}

	if rel < sb {
		return t.sb.at(rel), true
	}

	return &t.screen.lines[rel-sb], true
}

func (t *Terminal) lineRange() (first, last int) {
	first = t.sb.dropped
	return first, first + t.visibleSB() + t.rows - 1
}

func (t *Terminal) TextRange(a, b Pos) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if b.Before(a) {
		a, b = b, a
	}

	first, last := t.lineRange()
	a.Line = max(a.Line, first)
	b.Line = min(b.Line, last)

	var sb strings.Builder

	for id := a.Line; id <= b.Line; id++ {
		l, ok := t.lineAt(id)

		if !ok {
			continue
		}

		from, to := 0, len(l.cells)-1

		if id == a.Line {
			from = a.Col
		}

		if id == b.Line {
			to = min(b.Col, len(l.cells)-1)
		}

		var row strings.Builder

		for x := max(from, 0); x <= to; x++ {
			c := l.cells[x]

			if c.Width() == 0 {
				continue
			}

			if c.Rune == 0 {
				row.WriteByte(' ')
				continue
			}

			row.WriteRune(c.Rune)
		}

		text := row.String()

		if !l.wrapped || id == b.Line {
			text = strings.TrimRight(text, " ")
		}

		sb.WriteString(text)

		if id != b.Line && !l.wrapped {
			sb.WriteString("\r\n")
		}
	}

	return sb.String()
}

func (t *Terminal) Find(query string, matchCase bool) []Match {
	if query == "" {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	q := []rune(query)

	if !matchCase {
		q = []rune(strings.ToLower(query))
	}

	var out []Match
	first, last := t.lineRange()
	runes := make([]rune, 0, t.cols)
	pos := make([]Pos, 0, t.cols)

	flush := func() {
		for i := 0; i+len(q) <= len(runes); i++ {
			if !runesEqual(runes[i:i+len(q)], q) {
				continue
			}

			out = append(out, Match{Start: pos[i], End: pos[i+len(q)-1]})
			i += len(q) - 1
		}

		runes, pos = runes[:0], pos[:0]
	}

	for id := first; id <= last; id++ {
		l, ok := t.lineAt(id)

		if !ok {
			continue
		}

		for x, c := range l.cells {
			if c.Width() == 0 {
				continue
			}

			r := c.Rune

			if r == 0 {
				r = ' '
			}

			if !matchCase {
				r = unicode.ToLower(r)
			}

			runes = append(runes, r)
			pos = append(pos, Pos{Line: id, Col: x})
		}

		if !l.wrapped {
			flush()
		}
	}

	flush()

	return out
}

func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (t *Terminal) ClearScrollback() {
	t.mu.Lock()
	t.sb.clear()
	t.mu.Unlock()
	t.gen.Add(1)
}

func (t *Terminal) Clear() {
	t.mu.Lock()
	t.sb.clear()
	t.eraseDisplay(2)
	t.cur.x, t.cur.y = 0, 0
	t.mu.Unlock()
	t.gen.Add(1)
}
