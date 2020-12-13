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

func main() {
	MonitorForever()
}

var name = lns(`ThrowPro Minecraft Assistant`, `Version 0.4`)

type Monitor struct {
	timer      *time.Timer
	sm         *throwlib.SessionManager
	timeout    time.Duration
	clipTicker *time.Ticker

	*Display
	*File
}

func MonitorForever() {
	m := Monitor{timeout: time.Minute * 9}
	m.clipTicker = time.NewTicker(50 * time.Millisecond)
	m.sm = throwlib.NewSessionManager()
	m.Display = NewDisplay(m.sm)
	m.File = NewFile(m.Display)

	go m.Block()
	m.display.Block()
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
		m.File.Refresh()
		m.ExtendTimer()
	}
}

func (m *Monitor) ExtendTimer() {
	if m.timer != nil {
		m.timer.Stop()
	}
	m.timer = time.AfterFunc(m.timeout, func() { m.Reset() })
}

type File struct {
	file    *os.File
	display *Display
}

func NewFile(d *Display) *File {
	file := new(File)
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Println("error", err.Error())
		return nil
	}

	path := filepath.FromSlash(dir + "/.throwlib.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("error", err.Error())
		f = nil
	}
	file.file = f
	file.display = d
	return file
}

func (f *File) Refresh() {
	if f == nil {
		return
	}
	f.file.Truncate(0)
	f.file.Seek(0, 0)
	status := f.display.top.Text
	if _, err := f.file.WriteString(status); err != nil {
		log.Println("error writing file", err.Error())
	}
}

type Display struct {
	top    *widget.Label
	bottom *widget.Label
	window fyne.Window
	sm     *throwlib.SessionManager
}

func NewDisplay(sm *throwlib.SessionManager) *Display {
	d := new(Display)

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
	sm := d.sm
	throw := sm.ActiveSession.Throws[len(sm.ActiveSession.Throws)-1]
	guess := sm.Guess
	conf := sm.Confidence

	log.Println("current layer set", sm.ActiveSession.LayerSet)

	x, y := guess.Staircase()
	distPlayer := guess.Dist(throw.X, throw.Y)

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
	status := replacer.Replace(FORMATS[sm.ActiveSession.LayerSet.Code])
	mode := "Mode: " + sm.ActiveSession.LayerSet.Name

	log.Println("updating ui...", status, mode)
	d.top.SetText(status)
	d.bottom.SetText(mode)
}

func (d *Display) Reset() {
	d.top.SetText(name)
	d.bottom.SetText("Look at ender eye and press F3+C.")
}

func lns(ss ...string) string {
	return strings.Join(ss, "\r\n")
}

var FORMATS = map[string]string{
	"blind":         `{nether} nether to go {distance} blocks {line} ({coords} overworld)`,
	"educated":      `{nether} nether to go {distance} blocks {line} ({coords} overworld)`,
	"triangulation": `{coords} is {confidence} likely {line} ({distance} away, {nether} nether)`,
}
