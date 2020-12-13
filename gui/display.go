package main

//go:generate sh -c "(printf 'package main\nvar icon string=`'; base64 eye.png; printf '`') >Icon.go"

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
	"github.com/atotto/clipboard"
	"github.com/dantoye/throwpro/throwlib"
)

var FORMATS = map[string]string{
	"blind":         `{nether} nether to go {distance} blocks {line}({coords} overworld)`,
	"educated":      `{nether} nether to go {distance} blocks {line}({coords} overworld)`,
	"triangulation": `{coords} is {confidence} likely {line}({distance} away, {nether} nether)`,
}

var METHODS = map[string]string{
	"blind":         `Blind Guess`,
	"educated":      `Educated Travel`,
	"triangulation": `Gradual Triangulation`,
}
var name = lns(`ThrowPro Minecraft Assistant`, `Version 0.4`)

type Monitor struct {
	timer      *time.Timer
	sm         *throwlib.SessionManager
	timeout    time.Duration
	clipTicker *time.Ticker

	*Display
	*FileWriter
}

func StartClipboardMonitor(d *Display, sm *throwlib.SessionManager) {
	m := Monitor{timeout: time.Minute * 9}
	m.clipTicker = time.NewTicker(50 * time.Millisecond)
	m.Display = d
	m.sm = sm
	go m.Block()
}

func (m *Monitor) Block() {
	log.Println("monitoring clipboard")
	getClip := func() string {
		text, err := clipboard.ReadAll()
		if err != nil {
			log.Println("error:", err.Error())
			return ""
		}
		return text
	}

	lastText := getClip()
	for range m.clipTicker.C {
		text := getClip()
		if text == lastText {
			continue
		}
		lastText = text
		throw, err := throwlib.NewThrowFromString(text)
		if err != nil {
			log.Println("skipping an invalid clipboard:", err.Error())
			log.Println(text)
			continue
		}
		m.sm.NewThrow(throw)
		m.Display.Refresh()
		m.ExtendTimer()
	}
}

func (m *Monitor) ExtendTimer() {
	if m.timer != nil {
		m.timer.Stop()
	}
	m.timer = time.AfterFunc(m.timeout, func() { m.Reset() })
}

type FileWriter struct {
	file  *os.File
	path  string
	wpath string
}

func NewFileWriter() *FileWriter {
	file := new(FileWriter)
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Println("error", err.Error())
		return nil
	}

	file.path = filepath.FromSlash(dir + "/throwlib.txt")
	file.wpath = filepath.FromSlash(dir + "/.throwlib.txt")
	log.Println("writing to", file.wpath)
	return file
}

func (f *FileWriter) WriteScratch(status string) {
	file, err := os.OpenFile(f.wpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("error", err.Error())
		return
	}

	file.Truncate(0)
	file.Seek(0, 0)
	if _, err := file.WriteString(status); err != nil {
		log.Println("error writing file", err.Error())
	}
	file.Sync()
}

func (f *FileWriter) Write(status string) {
	if f.file == nil {
		return
	}

	f.WriteScratch(status)
	if err := os.Rename(f.wpath, f.path); err != nil {
		log.Println("error swapping file", err.Error())
	}
	f.WriteScratch(status)
}

type Display struct {
	top    *widget.Label
	bottom *widget.Label
	window fyne.Window

	sm *throwlib.SessionManager
	f  *FileWriter
}

func NewDisplay(sm *throwlib.SessionManager, f *FileWriter) *Display {
	d := new(Display)
	d.sm = sm
	d.f = f

	log.Println("creating UI")
	a := app.New()

	w := a.NewWindow("ThrowPro")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(300, 50))
	w.SetPadded(true)

	d.window = w

	mainUI := widget.NewLabel("Status")
	mainUI.TextStyle.Bold = true
	d.top = mainUI

	secondUI := widget.NewLabel("Guess")
	d.bottom = secondUI

	d.Reset()

	infoUI := widget.NewLabel("Info")
	debugUI := widget.NewLabel("Debug")
	debugUI.SetText(lns("Writing results to", f.path))

	var toggle func()
	showButton := widget.NewButton("Show Secret Details", func() { toggle() })

	var iconData, _ = base64.StdEncoding.DecodeString(icon)
	w.SetIcon(fyne.NewStaticResource("eye.png", iconData))

	toggle = func() {
		if infoUI.Hidden {
			infoUI.Show()
			debugUI.Show()
			mainUI.Hide()
			secondUI.Hide()
			showButton.SetText("Hide Secret Details")
			throwlib.DEBUG = true
			return
		}
		infoUI.Hide()
		debugUI.Hide()
		mainUI.Show()
		secondUI.Show()
		showButton.SetText("Show Secret Details")
	}
	infoUI.Hide()
	debugUI.Hide()

	w.SetContent(widget.NewVBox(mainUI, secondUI, infoUI, debugUI, showButton))

	infoUI.SetText("For help, message @Cudduw")
	return d
}

func (d *Display) Block() {
	log.Println("starting UI")
	d.window.ShowAndRun()
	d.window = nil
}

func (d *Display) Stop() {
	if d.window != nil {
		d.window.Close()
	}
}

func (d *Display) Refresh() {
	guess := d.sm.Guess
	throw := d.sm.Throws[len(d.sm.Throws)-1]
	chunk := throwlib.Chunk(guess.Chunk)
	conf := guess.Confidence

	x, y := chunk.Staircase()
	distPlayer := chunk.Dist(throw.X, throw.Y)

	distStr := fmt.Sprintf(`%.1fk`, distPlayer/1000)
	confStr := fmt.Sprintf(`%.1f%%`, float64(conf)/10)
	coords := fmt.Sprintf(`%d,%d`, x, y)
	nether := fmt.Sprintf(`%d,%d`, x/8, y/8)

	replacer := strings.NewReplacer(
		`{distance}`, distStr,
		`{confidence}`, confStr,
		`{coords}`, coords,
		`{nether}`, nether,
		`{line}`, "\n\r",
	)
	status := replacer.Replace(FORMATS[guess.Method])
	mode := "Mode: " + METHODS[guess.Method]

	log.Println("updating ui...", status, mode)
	d.top.SetText(status)
	d.bottom.SetText(mode)
	d.f.Write(status)
}

func (d *Display) Reset() {
	d.top.SetText(name)
	d.bottom.SetText("Look at ender eye and press F3+C.")
}

func lns(ss ...string) string {
	return strings.Join(ss, "\r\n")
}

func main() {
	sm := throwlib.NewSessionManager()
	file := NewFileWriter()
	display := NewDisplay(sm, file)

	StartClipboardMonitor(display, sm)
	display.Block()
}
