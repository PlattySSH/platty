package app

import (
	gioapp "gioui.org/app"

	"github.com/plattyssh/platty/internal/platform/dragdrop"
)

func (a *App) attachView(ev gioapp.ViewEvent) {
	e, ok := ev.(gioapp.Win32ViewEvent)

	if !ok || e.HWND == 0 {
		return
	}

	hwnd := e.HWND
	go a.win.Run(func() { dragdrop.Enable(hwnd, a.dropFiles) })
}

func (a *App) configured() {}
