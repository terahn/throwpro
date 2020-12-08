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
		return Throw{X: numbers[2] * 8, Y: numbers[3] * 8, Blind: true}, nil
	}
	if numbers[4] < -55 || numbers[4] > -15 {
		return Throw{X: numbers[2], Y: numbers[3], Blind: true}, nil
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

func NewSessionManager(d time.Duration) *SessionManager {
	sm := &SessionManager{
		Status:   make(chan string, 10),
		Guess:    make(chan string, 10),
		Duration: d,
	}

	sm.timer = time.AfterFunc(0, func() { sm.Message("Awaiting F3+C") })
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

	sm.Message("Awaiting F3+C")
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
		sm.Message("Blind Travel", fmt.Sprintf("Nether %d,%d", x/8, y/8))
		return
	}

	if sm.ActiveSession == nil {
		sm.ActiveSession = NewSession(throw)
		blind := sm.ActiveSession.Sorted().Central()
		x, y := blind.Center()
		sm.Message("Educated Travel", fmt.Sprintf("Near %d,%d (Nether %d,%d)", x, y, x/8, y/8))
		return
	}

	if !sm.ActiveSession.IsThrowUseful(throw) {
		return
	}
	matches, _ := sm.ActiveSession.AddThrow(throw)
	if matches == 0 {
		sm.ActiveSession = NewSession(throw)
		blind := sm.ActiveSession.Sorted().Central()
		x, y := blind.Center()
		sm.Message("Educated Travel", fmt.Sprintf("Near %d,%d (Nether %d,%d)", x, y, x/8, y/8))
		return
	}

	sm.Message("Overworld Guess", sm.ActiveSession.Sorted().String())
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
	w.SetContent(widget.NewVBox(statusUI, guessUI))

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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		statusUI.SetText("error: " + err.Error())
		log.Println("error", err.Error())
		return
	}
	sm := NewSessionManager(11 * time.Minute)

	var status, guess string
	updateUI := func() {
		log.Println("updating ui...", status, guess)
		statusUI.SetText(status)
		guessUI.SetText(guess)

		f.Truncate(0)
		f.Seek(0, 0)
		if _, err := f.WriteString(status + "\n" + guess); err != nil {
			log.Println("error writing file", err.Error())
		}
	}
	go func() {
		for {
			select {
			case status = <-sm.Status:
			case guess = <-sm.Guess:
			}
			updateUI()
		}
	}()
	go func() {
		log.Println("monitoring clipboard")
		lastText, err := clipboard.ReadAll()
		if err != nil {
			log.Println("error:", err.Error())
		}
		for {
			text, err := clipboard.ReadAll()
			if err != nil {
				log.Println("error:", err.Error())
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
