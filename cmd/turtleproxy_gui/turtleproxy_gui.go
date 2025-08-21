package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("TurtleProxy")

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("TurtleProxy",
			fyne.NewMenuItem("Config", func() {
				w.Show()
			}))
		desk.SetSystemTrayMenu(m)
	}

	logger := slog.Default()

	baseDir := filepath.Dir(os.Args[0])
	go func() {
	}

	w.SetContent(widget.NewLabel("Hello World!"))
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	a.Run()
}
