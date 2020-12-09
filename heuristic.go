package throwpro

import (
	"fmt"
	"log"
	"sort"
)

type ChunkList []Chunk

type Chunk [2]int

func (t Chunk) ChunkDist(other Chunk) float64 {
	ax, ay := t.Center()
	bx, by := other.Center()
	return dist(float64(ax), float64(ay), float64(bx), float64(by))
}

type Throw struct {
	X, Y, A float64
	Blind   bool
}

func NewThrowFromArray(arr [3]float64) Throw {
	return NewThrow(arr[0], arr[1], arr[2])
}

func NewThrow(x, y, a float64) Throw {
	return Throw{X: x, Y: y, A: radsFromDegs(a)}
}

func (t Throw) Similar(other Throw) bool {
	if dist(t.X, t.Y, other.X, other.Y) < 12 {
		return true
	}
	return false
}

type Session struct {
	Scores map[Chunk]int
	Throws []Throw
}

func NewSession(t Throw) *Session {
	s := &Session{Throws: []Throw{t}, Scores: make(map[Chunk]int)}
	chunks := ChunksInThrow(t)
	for _, c := range chunks {
		score := c.Score(t.A, t.X, t.Y)
		if score == 0 {
			continue
		}
		s.Scores[c] = score
	}
	return s
}

func (s *Session) IsThrowUseful(t Throw) bool {
	for _, existing := range s.Throws {
		if t.Similar(existing) {
			return false
		}
	}
	return true
}

func (s *Session) AddThrow(t Throw) (int, int) {
	s.Throws = append(s.Throws, t)
	chunks := ChunksInThrow(t)
	matches := 0
	discarded := 0

	activeChunks := make(map[Chunk]bool, 0)
	for _, c := range chunks {
		activeChunks[c] = true
		_, found := s.Scores[c]
		if !found {
			continue
		}
		score := c.Score(t.A, t.X, t.Y)
		if score < 3 {
			delete(s.Scores, c)
			discarded++
			continue
		}
		matches++
		s.Scores[c] += score
	}

	for c := range s.Scores {
		if _, found := activeChunks[c]; found {
			if s.Scores[c] == 0 {
				delete(s.Scores, c)
				discarded++
			}
		} else {
			delete(s.Scores, c)
			discarded++
		}
	}
	return matches, discarded
}

type Guess struct {
	Chunk
	Confidence int
}

type Guesses []Guess

func (g Guesses) String() string {
	central := g.Central()
	x, y := central.Staircase()
	return fmt.Sprintf("%d,%d with %.1f%% confidence", x, y, float64(central.Confidence)/10.0)
}

func (g Guesses) Central() Guess {
	if len(g) == 0 {
		return Guess{Chunk{}, 1000}
	}
	sx, sy := g[0].Chunk.Center()
	totalScore := g[0].Confidence
	sx *= totalScore
	sy *= totalScore

	for _, c := range g[1:] {
		if c.Confidence < g[0].Confidence*9/10 {
			break
		}
		x, y := c.Chunk.Center()
		totalScore += c.Confidence
		sx += x * c.Confidence
		sy += y * c.Confidence
	}
	average := ChunkFromPosition(float64(sx)/float64(totalScore), float64(sy)/float64(totalScore))
	closest := g[0]
	closestDistance := average.ChunkDist(closest.Chunk)
	for _, c := range g[1:] {
		dist := average.ChunkDist(c.Chunk)
		if dist < closestDistance {
			closest = c
			closestDistance = dist
		}
	}

	return closest
}

func (s Session) Sorted() Guesses {
	chunks := make(ChunkList, 0, len(s.Scores))
	for chunk := range s.Scores {
		chunks = append(chunks, chunk)
	}
	sorter := SessionSorter{s, chunks}
	sort.Sort(sorter)

	guesses := make(Guesses, 0, len(chunks))
	sumScore := 0
	topScore := s.Scores[chunks[0]]
	for _, c := range chunks {
		if s.Scores[c] < topScore*7/10 {
			break
		}
		sumScore += s.Scores[c]
	}
	log.Println("total confidence", sumScore)

	for _, c := range chunks {
		confidence := (1000 * s.Scores[c]) / sumScore
		if len(chunks) == 1 {
			confidence = 1000
		}
		if confidence < 1 {
			confidence = 1
		}
		guesses = append(guesses, Guess{c, confidence})
	}

	return guesses
}

type SessionSorter struct {
	Session
	ChunkList
}

func (s SessionSorter) Len() int {
	return len(s.ChunkList)
}

func (s SessionSorter) Less(a, b int) bool {
	return s.Scores[s.ChunkList[a]] > s.Scores[s.ChunkList[b]]
}

func (s SessionSorter) Swap(a, b int) {
	s.ChunkList[a], s.ChunkList[b] = s.ChunkList[b], s.ChunkList[a]
}

func GetBlindGuess(t Throw) Guess {
	d := dist(0, 0, t.X, t.Y)
	x, y := t.X/d, t.Y/d
	return Guess{Chunk: ChunkFromPosition(x*111*16, y*111*16), Confidence: 76}
}
