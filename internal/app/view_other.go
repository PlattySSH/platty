//go:build !windows

package app

import gioapp "gioui.org/app"

func (a *App) attachView(ev gioapp.ViewEvent) {}
