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

var name = lns(`ThrowPro Minecraft Assistant`, `Version 0.5`)

var BLURB = lns(
	`Basic Instructions`,
	`1. ADD data by throwing an ender eye, staring right at it, and pressing F3+C.`,
	`2. ADD once to get an educated travel guess, twice to begin triangulation.`,
	`Tip: The more you ADD, the more confident the guess can be.`,
	``,
	`Advanced Use Tips`,
	`1. Predict inside nether to remember your portal.`,
	`2. Don't look up at the sky if you want a blind guess.`,
	``,
	`For further help, message @Cudduw.`,
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

	file.path = filepath.FromSlash(dir + "/throwpro.txt")
	file.wpath = filepath.FromSlash(dir + "/.throwpro.txt")
	log.Println("writing to", file.wpath)
	return file
}

func (f *FileWriter) WriteScratch(status string) error {
	file, err := os.OpenFile(f.wpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("error", err.Error())
		return err
	}

	file.Truncate(0)
	file.Seek(0, 0)
	if _, err := file.WriteString(status); err != nil {
		log.Println("error writing file", err.Error())
		return err
	}
	return file.Sync()
}

func (f *FileWriter) Write(status string) error {
	f.WriteScratch(status)
	if err := os.Rename(f.wpath, f.path); err != nil {
		log.Println("error swapping file", err.Error())
		return err
	}
	return f.WriteScratch(status)
}

type Display struct {
	top    *widget.Label
	bottom *widget.Label
	debug  func(error)

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

	help := a.NewWindow("ThrowPro Help")
	help.SetPadded(true)
	help.Hide()
	help.SetCloseIntercept(func() {
		help.Hide()
	})
	w.SetOnClosed(func() {
		help.Close()
	})

	d.window = w

	mainUI := widget.NewLabel("Status")
	mainUI.TextStyle.Bold = true
	d.top = mainUI

	secondUI := widget.NewLabel("Guess")
	d.bottom = secondUI

	d.Reset()

	infoUI := widget.NewLabel("Info")
	debugUI := widget.NewLabel("Debug")
	debugUI.SetText("Writing results to " + f.path)
	d.debug = func(e error) {
		if e == nil {
			return
		}
		debugUI.SetText("Error: " + e.Error())
	}

	var toggle func()
	showButton := widget.NewButton("Open Help Window", func() { toggle() })

	var iconData, _ = base64.StdEncoding.DecodeString(icon)
	w.SetIcon(fyne.NewStaticResource("eye.png", iconData))

	toggle = func() {
		throwlib.DEBUG = true
		help.Show()
		return
	}

	w.SetContent(widget.NewVBox(mainUI, secondUI, showButton))
	help.SetContent(widget.NewVBox(infoUI, debugUI))
	infoUI.SetText(BLURB)

	d.Refresh()
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
	if len(d.sm.Throws) == 0 {
		d.Reset()
		return
	}
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
		`{line}`, "\n",
	)
	status := replacer.Replace(FORMATS[guess.Method])
	mode := "Mode: " + METHODS[guess.Method]
	if portal := d.sm.Portal; portal != nil {
		mode = fmt.Sprintf("Portal: %d,%d", portal[0], portal[1])
	}

	log.Println("updating ui...", status, mode)
	d.top.SetText(status)
	d.bottom.SetText(mode)
	d.debug(d.f.Write(status))
}

func (d *Display) Reset() {
	d.top.SetText(name)
	d.sm.Throws = nil
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
