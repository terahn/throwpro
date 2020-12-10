package throwpro

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
	"github.com/atotto/clipboard"
)

// /execute in minecraft:overworld run tp @s -214.79 104.61 386.16 76.50 -32.40
func NewThrowFromString(s string) (Throw, error) {
	parts := strings.Split(s, " ")
	numbers := make([]float64, 0)
	for _, p := range parts {
		num, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			continue
		}
		numbers = append(numbers, num)
	}
	if len(numbers) != 5 {
		return Throw{}, fmt.Errorf("invalid format for command, has %d numbers", len(numbers))
	}
	if parts[2] != "minecraft:overworld" {
		return Throw{X: numbers[0] * 8, Y: numbers[2] * 8, Blind: true}, nil
	}
	if numbers[4] < -48 || numbers[4] > -12 {
		return Throw{X: numbers[0], Y: numbers[2], Blind: true}, nil
	}
	return NewThrow(numbers[0], numbers[2], numbers[3]), nil
}

type SessionManager struct {
	lock  sync.RWMutex
	timer *time.Timer

	Duration      time.Duration
	ActiveSession *Session

	Status chan string
	Guess  chan string
}

var waiting = name
var waiting2 = "Look at ender eye and press F3+C."

func NewSessionManager(d time.Duration) *SessionManager {
	sm := &SessionManager{
		Status:   make(chan string, 10),
		Guess:    make(chan string, 10),
		Duration: d,
	}

	sm.timer = time.AfterFunc(0, func() { sm.Message(waiting, waiting2) })
	sm.Reset()
	return sm
}

func (sm *SessionManager) Message(status string, answer ...string) {
	sm.Status <- status
	if len(answer) > 0 {
		sm.Guess <- answer[0]
	} else {
		sm.Guess <- ""
	}
}

func (sm *SessionManager) Reset() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.Message(waiting, waiting2)
	sm.ActiveSession = NewSession()
}

func (sm *SessionManager) Input(s string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.timer.Stop()
	sm.timer = time.AfterFunc(sm.Duration, func() { sm.Reset() })

	throw, err := NewThrowFromString(s)
	if err != nil {
		sm.Message("Invalid Input")
		return
	}
	if throw.Blind {
		chunk := GetBlindGuess(throw)
		sm.ActiveSession = NewSession()
		x, y := chunk.Center()
		sm.Message(fmt.Sprintf(lns("%d,%d nether", "(%d, %d overworld)"), x/8, y/8, x, y), "Mode: Blind Travel")
		return
	}

	matches := sm.ActiveSession.NewThrow(throw)
	if matches == 0 {
		sm.ActiveSession = NewSession()
		sm.ActiveSession.NewThrow(throw)
		blind := sm.ActiveSession.Guess().Central()
		x, y := blind.Staircase()
		sm.Message(fmt.Sprintf(lns("%d,%d nether", "(%d,%d overworld)"), x/8, y/8, x, y), "Mode: Educated Travel")
		return
	}

	sm.Message(lns(sm.ActiveSession.Guess().String(), ""), "Mode: Overworld Triangulation")
}

func ClipboardReader() {
	log.Println("creating UI")
	a := app.New()
	w := a.NewWindow("ThrowPro")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(300, 50))
	w.SetPadded(true)

	mainUI := widget.NewLabel("Status")
	mainUI.TextStyle.Bold = true
	secondUI := widget.NewLabel("Guess")

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
	debugChan := make(chan string, 100)
	debug := func(l1, l2 string) {
		debugChan <- lns(l1, l2)
	}

	defer w.ShowAndRun()
	defer func() {
		err := recover()
		if err != nil {
			mainUI.SetText("Error")
			secondUI.SetText(fmt.Sprintf(`%s`, err))
		}
	}()

	dir, err := os.UserHomeDir()
	if err != nil {
		mainUI.SetText("error: " + err.Error())
		log.Println("error", err.Error())
		return
	}

	clip := w.Clipboard()
	if clip == nil {
		debug("Clipboard warning...", "Method A failure.")
	}

	path := filepath.FromSlash(dir + "/.throwpro.txt")
	debug("Writing to file...", dir+"/.throwpro.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		debugChan <- "Warn: " + err.Error()
		log.Println("error", err.Error())
		f = nil
	}
	sm := NewSessionManager(11 * time.Minute)

	var status, guess string
	go func() {
		for {
			select {
			case status = <-sm.Status:
			case guess = <-sm.Guess:
			}

			log.Println("updating ui...", status, guess)
			mainUI.SetText(status)
			secondUI.SetText(guess)

			if f == nil {
				continue
			}
			f.Truncate(0)
			f.Seek(0, 0)
			if _, err := f.WriteString(lns(status, guess)); err != nil {
				log.Println("error writing file", err.Error())
			}
		}
	}()
	go func() {
		for i := range debugChan {
			debugUI.SetText(i)
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		log.Println("monitoring clipboard")
		getClip := func() string {
			if clip != nil {
				return clip.Content()
			}
			text, err := clipboard.ReadAll()
			if err != nil {
				log.Println("error:", err.Error())
				debug("Clipboard warning...", err.Error())
				return ""
			}
			return text
		}

		lastText := getClip()
		for {
			text := getClip()
			if text == lastText {
				time.Sleep(120 * time.Millisecond)
				continue
			}
			lastText = text

			if _, err := NewThrowFromString(text); err != nil {
				log.Println("skipping an invalid clipboard:", err.Error())
				log.Println(text)
				continue
			}
			sm.Input(text)
		}
	}()
	log.Println("starting UI")
}

func Begin() {
	sm := NewSessionManager(10 * time.Second)
	go func() {
		for s := range sm.Guess {
			log.Println(s)
		}
	}()

	for {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		if text == "" {
			return
		}
		sm.Input(text)

	}
}

func lns(ss ...string) string {
	return strings.Join(ss, "\r\n")
}
