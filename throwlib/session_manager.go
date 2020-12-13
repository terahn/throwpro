package throwlib

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
)

type Backend interface {
	BestGuess(...Throw) Guess
}

type SessionManager struct {
	lock sync.RWMutex

	Backend

	Throws []Throw
	Guess  Guess
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{Backend: NewSession()}
	return sm
}

func (sm *SessionManager) NewThrow(throw Throw) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for _, t := range sm.Throws {
		if throw.Similar(t) {
			return
		}
	}

	sm.Throws = append(sm.Throws, throw)
	guess := sm.Backend.BestGuess(sm.Throws...)
	if guess.Method == "reset" {
		sm.Throws = []Throw{throw}
		guess = sm.Backend.BestGuess(sm.Throws...)
		log.Println("new session for throw", throw)
	}
	sm.Guess = guess
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
