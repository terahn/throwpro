package throwpro

import (
	"bufio"
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

const waiting = name
const waiting2 = `Look at ender eye and press F3+C.`

func NewSessionManager(d time.Duration) *SessionManager {
	sm := &SessionManager{
		Status:   make(chan string, 10),
		Guess:    make(chan string, 10),
		Duration: d,
	}

	sm.timer = time.AfterFunc(0, func() { sm.Message(waiting, waiting2) })
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
	sm.ActiveSession = nil
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
		sm.ActiveSession = NewSession(throw)
		x, y := chunk.Center()
		sm.Message(fmt.Sprintf("%d,%d nether (%d, %d overworld)", x/8, y/8, x, y), "Mode: Blind Travel")
		return
	}

	if sm.ActiveSession == nil {
		sm.ActiveSession = NewSession(throw)
		blind := sm.ActiveSession.Sorted().Central()
		x, y := blind.Staircase()
		sm.Message(fmt.Sprintf("%d,%d nether (%d,%d overworld)", x/8, y/8, x, y), "Mode: Educated Travel")
		return
	}

	if !sm.ActiveSession.IsThrowUseful(throw) {
		return
	}
	matches, _ := sm.ActiveSession.AddThrow(throw)
	if matches == 0 {
		sm.ActiveSession = NewSession(throw)
		blind := sm.ActiveSession.Sorted().Central()
		x, y := blind.Staircase()
		sm.Message(fmt.Sprintf("%d,%d nether (%d,%d overworld)", x/8, y/8, x, y), "Mode: Educated Travel")
		return
	}

	sm.Message(sm.ActiveSession.Sorted().String(), "Mode: Overworld Triangulation")
}

func ClipboardReader() {
	log.Println("creating UI")
	a := app.New()
	w := a.NewWindow("ThrowPro")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(300, 50))
	w.SetPadded(true)

	statusUI := widget.NewLabel("Status")
	guessUI := widget.NewLabel("Guess")
	infoUI := widget.NewLabel("Info")
	fileUI := widget.NewLabel("File")
	w.SetContent(widget.NewVBox(statusUI, guessUI, infoUI, fileUI))
	infoUI.SetText("For help, message @Cudduw")

	defer w.ShowAndRun()
	defer func() {
		err := recover()
		if err != nil {
			statusUI.SetText("Error")
			guessUI.SetText(fmt.Sprintf(`%s`, err))
		}
	}()

	dir, err := os.UserHomeDir()
	if err != nil {
		statusUI.SetText("error: " + err.Error())
		log.Println("error", err.Error())
		return
	}
	path := filepath.FromSlash(dir + "/.throwpro.txt")
	fileUI.SetText("Writing to " + dir + "/.throwpro.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fileUI.SetText("Warn: " + err.Error())
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
			statusUI.SetText(status)
			guessUI.SetText(guess)

			if f == nil {
				continue
			}
			f.Truncate(0)
			f.Seek(0, 0)
			if _, err := f.WriteString(status + "\n" + guess); err != nil {
				log.Println("error writing file", err.Error())
			}
		}
	}()
	go func() {
		log.Println("monitoring clipboard")
		lastText, err := clipboard.ReadAll()
		if err != nil {
			log.Println("error:", err.Error())
			fileUI.SetText("Warn: " + err.Error())
		}
		for {
			text, err := clipboard.ReadAll()
			if err != nil {
				log.Println("error:", err.Error())
				fileUI.SetText("Warn: " + err.Error())
				continue
			}
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
