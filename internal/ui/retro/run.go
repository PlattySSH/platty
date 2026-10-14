package retro

import "gioui.org/f32"

func (r *Computer) OnRun(fn func(cmd string)) {
	r.onRun = fn
}

func (r *Computer) run(cmd string) {
	if r.onRun != nil {
		r.onRun(cmd)
	}
}

func (r *Computer) Typed() string {
	return string(r.typed)
}

func (r *Computer) Lines() []string {
	return r.lines
}

func (r *Computer) KeyPos(label string) (f32.Point, bool) {
	for _, kr := range r.keyRects {
		if kr.k.label == label {
			return r.window(vec3{(kr.x0 + kr.x1) / 2, keyTop(false), (kr.z0 + kr.z1) / 2}), true
		}
	}

	return f32.Point{}, false
}

func (r *Computer) ScreenPos() f32.Point {
	return r.window(vec3{0, (screenY0 + screenY1) / 2, screenZ})
}

func (r *Computer) CasePos() f32.Point {
	return r.window(vec3{0.6, 0.15, 0.62})
}

func (r *Computer) window(p vec3) f32.Point {
	return r.fit(p).Add(r.origin)
}
