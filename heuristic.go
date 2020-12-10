package throwpro

import (
	"fmt"
	"sort"
	"strings"
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
	if dist(t.X, t.Y, other.X, other.Y) < 6 {
		return true
	}
	return false
}

type ThrowResults struct {
	Throw
	Scores map[Chunk]int
}

func (t ThrowResults) Matches(any []ThrowResults) int {
	matches := 0

	for _, other := range any {
		myMatches := 0
		avgScore := 0
		total := 0
		maxScore := 0

		for _, score := range other.Scores {
			avgScore += score
			total++
			if score > maxScore {
				maxScore = score
			}
		}

		for tChunk, score := range t.Scores {
			if score*total < avgScore {
				continue
			}
			if score < maxScore*9/10 {
				continue
			}
			for _, other := range any {
				if _, found := other.Scores[tChunk]; found {
					myMatches++
				}
			}
		}
		if myMatches == 0 {
			return 0
		}
		matches += myMatches
	}

	return matches
}

func SumScores(t Throw, layers []func(Throw, Chunk) int) ThrowResults {
	res := ThrowResults{t, make(map[Chunk]int)}
	chunks := ChunksInThrow(t)
	for _, c := range chunks {
		score := 0
		for _, l := range layers {
			score += l(t, c)
		}
		res.Scores[c] = score
	}
	return res
}

func MergeScores(throws ...ThrowResults) Guesses {
	combined := make(map[Chunk]int)
	average := 0
	total := 0
	max := 0
	for n, t := range throws {
		for chunk, score := range t.Scores {
			combined[chunk] += score + n
			average += score
			total++
			if score > max {
				max = score
			}
		}
	}

	guesses := make(Guesses, 0, len(combined))
	for chunk, score := range combined {
		if score*total < average {
			continue
		}
		if score < max*8/10 {
			continue
		}
		guesses = append(guesses, Guess{chunk, score})
	}
	return guesses
}

type Session struct {
	Throws      []ThrowResults
	CustomLayer *LayerSet
}

func NewSession(cl ...LayerSet) *Session {
	if len(cl) > 0 {
		return &Session{CustomLayer: &cl[0]}
	}
	return &Session{}
}

func (s *Session) Explain(t Throw, goal Chunk, guess Chunk) string {
	chunks := ChunksInThrow(t)
	logs := []string{}
	for _, c := range chunks {
		if c.ChunkDist(goal) > 300 && c.ChunkDist(guess) > 300 {
			continue
		}
		logs = append(logs, fmt.Sprintf("\n%s angle %f, ring %d, scores", c, c.Angle(t.A, t.X, t.Y), RingID(c)))

		for _, l := range TwoEyeSet().Layers() {
			logs = append(logs, fmt.Sprintf(`l1(%d)`, l(t, c)))
		}
		logs = append(logs, fmt.Sprintf("total %d", s.Score(t, c)))
	}
	return strings.Join(logs, ",")
}

func (s *Session) Score(t Throw, c Chunk) int {
	score := 0
	for _, l := range TwoEyeSet().Layers() {
		score += l(t, c)
	}
	return score
}

func (s *Session) NewThrow(t Throw) int {
	newScores := SumScores(t, TwoEyeSet().Layers())
	matches := newScores.Matches(s.Throws)
	s.Throws = append(s.Throws, newScores)
	return matches
}

func (s *Session) Guess() Guesses {
	if len(s.Throws) == 1 {
		set := OneEyeSet()
		if s.CustomLayer != nil {
			set = *s.CustomLayer
		}

		newScores := SumScores(s.Throws[0].Throw, set.Layers())
		guesses := MergeScores(newScores)
		sort.Sort(guesses)
		return guesses
	}
	guesses := MergeScores(s.Throws...)
	sort.Sort(guesses)
	return guesses
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

	highestConfidence := 0
	for _, c := range g[1:] {
		if c.Confidence < g[0].Confidence*9/10 {
			break
		}
		x, y := c.Chunk.Center()
		totalScore += c.Confidence
		sx += x * c.Confidence
		sy += y * c.Confidence
		if c.Confidence > highestConfidence {
			highestConfidence = c.Confidence
		}
	}
	average := ChunkFromPosition(float64(sx)/float64(totalScore), float64(sy)/float64(totalScore))
	closest := g[0]
	closestDistance := average.ChunkDist(closest.Chunk)
	for _, c := range g[1:] {
		if c.Confidence < highestConfidence*8/10 {
			continue
		}
		dist := average.ChunkDist(c.Chunk)
		if dist < closestDistance {
			closest = c
			closestDistance = dist
		}
	}

	return closest
}

func (s Guesses) Len() int {
	return len(s)
}

func (s Guesses) Less(a, b int) bool {
	return s[a].Confidence > s[b].Confidence
}

func (s Guesses) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func GetBlindGuess(t Throw) Guess {
	d := dist(0, 0, t.X, t.Y)
	x, y := t.X/d, t.Y/d
	return Guess{Chunk: ChunkFromPosition(x*111*16, y*111*16), Confidence: 76}
}
