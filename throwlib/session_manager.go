package throwlib

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SessionManager struct {
	lock sync.RWMutex

	Duration      time.Duration
	ActiveSession *Session

	Guess      Chunk
	Confidence int
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{}
	sm.Reset()
	return sm
}

func (sm *SessionManager) Reset() {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.ActiveSession = NewSession()
}

func (sm *SessionManager) NewThrow(throw Throw) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for _, t := range sm.ActiveSession.Throws {
		if throw.Similar(t) {
			return
		}
	}

	guess, conf := sm.ActiveSession.NewThrow(throw).BestGuess()
	if len(sm.ActiveSession.Scores) > 0 {
		log.Println("consumed throw", throw, "for", len(sm.ActiveSession.Scores), "matches")
	} else {
		sm.ActiveSession = NewSession()
		guess, conf = sm.ActiveSession.NewThrow(throw).BestGuess()
		log.Println("new session for throw", throw, "found", len(sm.ActiveSession.Scores), "chunks")
	}
	sm.Guess = guess
	sm.Confidence = conf
}

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
		t := NewBlindThrow(numbers[0]*8, numbers[2]*8)
		t.Type = Nether
		return t, nil
	}
	if numbers[4] < -48 || numbers[4] > -12 {
		return NewBlindThrow(numbers[0], numbers[2]), nil
	}
	return NewThrow(numbers[0], numbers[2], numbers[3]), nil
}
