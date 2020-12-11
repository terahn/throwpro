package throwpro

import (
	"fmt"
	"log"
	"sort"
	"strings"

	dbscan "github.com/866/go-dbscan"
	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

var DEBUG = false

var DEBUG_CHUNK Chunk

const CHUNK_GROUP = 1500

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

type Session struct {
	Throws      []Throw
	CustomLayer *LayerSet

	Scores     map[Chunk]int
	TotalScore int
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

		for _, l := range s.Layers() {
			logs = append(logs, fmt.Sprintf(`l1(%d)`, l(t, c)))
		}
		logs = append(logs, fmt.Sprintf("total %d", s.Score(t, c)))
	}
	return strings.Join(logs, ",")
}

func (s *Session) SumScores(layers []func(Throw, Chunk) int) (map[Chunk]int, int) {
	scores := make(map[Chunk]int)
	reject := make(map[Chunk]bool)
	count := make(map[Chunk]int)

	highest := 0
	for _, t := range s.Throws {
		chunks := ChunksInThrow(t)
		for _, c := range chunks {
			count[c]++
			score := s.Score(t, c)
			if score == 0 {
				reject[c] = true
				continue
			}
			scores[c] += score
			if scores[c] > highest {
				highest = scores[c]
			}
		}
	}
	// clear all totally rejected chunk scores
	rejects := 0
	total := 0
	for c, score := range scores {
		if reject[c] || count[c] < len(s.Throws) || score < highest*8/10 {
			rejects++
			delete(scores, c)
			continue
		}
		total += score
	}

	if DEBUG {
		log.Println("summed scores, matched", len(scores), "rejected", rejects, "highscore", highest)
	}
	return scores, total
}

func (s *Session) Score(t Throw, c Chunk) int {
	score := 0
	for _, l := range s.Layers() {
		s := l(t, c)
		if s == 0 {
			// log.Println("chunk", c, "failed test", n)
			return 0
		}
		score += s
	}
	return score
}

func (s *Session) Chunks() []Chunk {
	chunks := make([]Chunk, 0, len(s.Scores))
	for chunk := range s.Scores {
		chunks = append(chunks, chunk)
	}
	sort.SliceStable(chunks, func(i int, j int) bool {
		if chunks[i][0] < chunks[j][0] {
			return true
		}
		if chunks[i][1] < chunks[j][1] {
			return true
		}
		if chunks[i] == chunks[j] {
			panic("equal chunks")
		}
		return false
	})
	return chunks
}

func (s *Session) ByScore() []Chunk {
	chunks := s.Chunks()
	sort.Slice(chunks, func(i int, j int) bool {
		return s.Scores[chunks[i]] > s.Scores[chunks[j]]
	})
	return chunks
}

func (s *Session) NewThrow(t Throw) {
	s.Throws = append(s.Throws, t)
	s.Scores, s.TotalScore = s.SumScores(s.Layers())
}

func (s *Session) Layers() []func(Throw, Chunk) int {
	if s.CustomLayer != nil {
		return s.CustomLayer.Layers()
	}
	if len(s.Throws) == 1 {
		return OneEyeSet().Layers()
	}
	return TwoEyeSet().Layers()
}

func (s *Session) BestGuess() (Chunk, int) {
	if len(s.Scores) == 0 {
		return Chunk{}, 1000
	}

	chunks := s.Chunks()
	averageScore := s.TotalScore / len(chunks)
	pts := make([]dbscan.Clusterable, 0, len(chunks))
	counted := 0
	for _, c := range chunks {
		if s.Scores[c] < averageScore {
			continue
		}
		counted++
		pts = append(pts, c)
	}
	// log.Println("observing", counted, "/", len(chunks), "above average", averageScore)
	clustered := dbscan.Clusterize(pts, 1, CHUNK_GROUP)
	clusterGroups := make(map[int]clusters.Cluster)
	allowOutliers := true
	for id, c := range clustered {
		group := clusters.Cluster{}
		group.Center = []float64{0, 0}

		for _, point := range c {
			x, y := point.(Chunk).Center()
			group.Center[0] += float64(x)
			group.Center[1] += float64(y)

			kobs := clusters.Coordinates([]float64{float64(x), float64(y)})
			group.Observations = append(group.Observations, kobs)
		}
		group.Center[0] /= float64(len(group.Observations))
		group.Center[1] /= float64(len(group.Observations))

		clusterGroups[id] = group
		if len(c) > 1 {
			allowOutliers = false
		}
		if DEBUG {
			log.Println("cluster", id, "size", len(c))
		}
	}

	display := make(clusters.Clusters, 0, len(clusterGroups))
	for id, c := range clusterGroups {
		if len(c.Observations) == 1 && !allowOutliers {
			continue
		}
		if DEBUG {
			log.Println("cluster", id, "size", len(c.Observations), "center", c.Center)
		}
		display = append(display, c)
	}
	if DEBUG {
		log.Println("chunks", len(pts), "clusters", len(display))
	}

	if len(display) == 0 {
		panic(fmt.Sprintf(`%t, %#v = %#v`, allowOutliers, display, clusterGroups))
	}

	t := s.Throws[len(s.Throws)-1]
	throwCoords := clusters.Coordinates{t.X, t.Y}
	leastFar := display[0]
	leastFound := leastFar.Center.Distance(throwCoords)
	if len(display) > 1 {
		for _, c := range display[1:] {
			dist := c.Center.Distance(throwCoords)
			if dist < leastFound {
				leastFound = dist
				leastFar = c
			}
		}
	}

	if DEBUG {
		kmeans.SimplePlotter{}.Plot(display, 0)
		log.Println("closest cluster", leastFar.Center, leastFound)
	}

	sx, sy := leastFar.Center[0], leastFar.Center[1]
	average := ChunkFromPosition(sx, sy)
	closest := chunks[0]
	closestDistance := average.ChunkDist(closest)
	for _, c := range chunks {
		dist := average.ChunkDist(c)
		if dist < closestDistance {
			closest = c
			closestDistance = dist
		}
	}

	if DEBUG {
		log.Println("total score", s.TotalScore)
		l := 10
		scored := s.ByScore()
		if len(scored) < 10 {
			l = len(scored)
		}
		for _, chunk := range s.ByScore()[:l] {
			log.Println("chunk", chunk, "score", s.Scores[chunk])
		}
	}

	return closest, s.Scores[closest] * 1000 / s.TotalScore
}

func GetBlindGuess(t Throw) Chunk {
	d := dist(0, 0, t.X, t.Y)
	x, y := t.X/d, t.Y/d
	return ChunkFromPosition(x*111*16, y*111*16)
}

func Must(e error) {
	if e != nil {
		panic(e)
	}
}
