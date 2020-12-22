package throwlib

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Request struct {
	Clips   []string `json:"clips"`
	Options struct {
		Hyper bool `json:"hyper"`
	} `json:"options"`
}

type Response struct {
	Chunk  *[2]int `json:"chunk"`
	Coords *[2]int `json:"coords"`
	Player *[2]int `json:"player"`
	Portal *[2]int `json:"portal"`

	Method     string `json:"method"`
	Confidence int    `json:"confidence"`
	Reset      bool   `json:"reset"`
}

func NewResponse(req Request) Response {
	b, _ := json.Marshal(req)
	log.Println("request", string(b))

	res := Response{}
	sess := NewSession()
	if req.Options.Hyper {
		sess.Options.Hyper = true
	}

	log.Println("handling request with", len(req.Clips), "clips")

	for _, text := range req.Clips {
		throw, err := NewThrowFromString(text)
		if err != nil {
			log.Println("skipping an invalid clipboard:", err.Error())
			log.Println(text)
			continue
		}
		if throw.Type == Nether {
			if res.Portal == nil {
				res.Portal = &[2]int{int(throw.X / 8), int(throw.Y / 8)}
			}
			if len(sess.Throws) > 0 {
				continue
			}
		}

		similar := false
		for _, t := range sess.Throws {
			if throw.Similar(t) {
				similar = true
			}
		}
		if similar {
			continue
		}

		if throw.Type == Blind {
			throw.X = 0
			throw.Y = 0
		}

		sess.Throws = append(sess.Throws, throw)
	}
	lastThrow := sess.Throws[len(sess.Throws)-1]
	guess := sess.BestGuess(sess.Throws...)
	if guess.Method == "reset" {
		sess.Throws = []Throw{lastThrow}
		res.Reset = true
		guess = sess.BestGuess(sess.Throws...)
		log.Println("new session for throw", lastThrow)
	}
	x, y := Chunk(guess.Chunk).Staircase()

	res.Chunk = &guess.Chunk
	res.Coords = &[2]int{x, y}
	res.Player = &[2]int{int(lastThrow.X), int(lastThrow.Y)}
	res.Confidence = guess.Confidence
	res.Method = guess.Method

	c, _ := json.Marshal(res)
	log.Println("response", string(c))

	return res
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
