package throwlib

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	dbscan "github.com/866/go-dbscan"
	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

var DEBUG = false

var DEBUG_CHUNK Chunk

type ChunkList []Chunk

type Chunk [2]int

func (t Chunk) ChunkDist(other Chunk) float64 {
	bx, by := other.Center()
	return t.Dist(float64(bx), float64(by))
}

func (t Chunk) Dist(x, y float64) float64 {
	ax, ay := t.Center()
	return dist(float64(ax), float64(ay), x, y)
}

func (c Chunk) Staircase() (int, int) {
	x, y := c.Center()
	return x - 4, y - 4
}

func (c Chunk) String() string {
	x, y := c.Center()
	ring := RingID(c)
	return fmt.Sprintf("chunk %d,%d \t(center %d, %d, ring %d)", c[0], c[1], x, y, ring)
}

func (c Chunk) Angle(a, sx, sy float64) float64 {
	x, y := c.Center()
	atan := math.Atan2(sx-float64(x), float64(y)-sy) + math.Pi*2
	atan = math.Mod(atan, math.Pi*2)
	diff := wrapRads(a - atan)
	return diff
}

func (c Chunk) Center() (int, int) {
	return c[0]*16 + 8, c[1]*16 + 8
}

type ThrowType int

func (t ThrowType) String() string {
	return throwNames[t]
}

const (
	Overworld ThrowType = iota
	Blind
	Nether
)

var throwNames = map[ThrowType]string{Overworld: "overworld", Blind: "blind", Nether: "nether"}

type Throw struct {
	X, Y, A float64
	Type    ThrowType
}

func NewThrowFromArray(arr [3]float64) Throw {
	return NewThrow(arr[0], arr[1], arr[2])
}

func NewThrow(x, y, a float64) Throw {
	return Throw{Type: Overworld, X: x, Y: y, A: radsFromDegs(a)}
}

func NewBlindThrow(x, y float64) Throw {
	atan := math.Atan2(-x, y) + math.Pi*2
	atan = math.Mod(atan, math.Pi*2)
	return Throw{X: x, Y: y, Type: Blind, A: atan}
}

func (t Throw) Similar(other Throw) bool {
	if dist(t.X, t.Y, other.X, other.Y) < 6 {
		return true
	}
	return false
}

type Guess struct {
	Chunk      [2]int `json:"chunk"`
	Method     string `json:"method"`
	Confidence int    `json:"confidence"`
}

func (g Guess) String() string {
	return fmt.Sprintf(`%s %d %s `, g.Method, g.Confidence, Chunk(g.Chunk))
}

type ScoredChunk struct {
	Chunk
	Score int
}

func (c ScoredChunk) Distance(other interface{}) float64 {
	o := other.(ScoredChunk)
	dx := (c.Chunk[0] - o.Chunk[0])
	dy := (c.Chunk[1] - o.Chunk[1])
	dc := (c.Score - o.Score)
	return math.Sqrt(float64(dx*dx + dy*dy + dc*dc))
}

func (c ScoredChunk) GetID() string {
	return fmt.Sprintf(`%d,%d`, c.Chunk[0], c.Chunk[1])
}

type Session struct {
	Throws      []Throw
	CustomLayer *LayerSet
	LayerSet    LayerSet

	Scores     map[Chunk]int
	TotalScore int
}

func NewSession(cl ...LayerSet) *Session {
	if len(cl) > 0 {
		return &Session{CustomLayer: &cl[0]}
	}
	return &Session{}
}

func (s *Session) Chunks() []Chunk {
	chunks := make([]Chunk, 0, len(s.Scores))
	for chunk := range s.Scores {
		chunks = append(chunks, chunk)
	}
	sort.Slice(chunks, func(i int, j int) bool {
		if chunks[i][0] == chunks[j][0] {
			return chunks[i][1] < chunks[j][1]
		}
		return chunks[i][1] < chunks[j][1]
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

func (s *Session) CalcLayerSet() LayerSet {
	if s.CustomLayer != nil {
		return *s.CustomLayer
	}
	if len(s.Throws) == 1 {
		if s.Throws[0].Type == Blind {
			return ZeroEyeSet
		}
		return OneEyeSet
	}
	return TwoEyeSet
}

func (s *Session) Layers() LayerSet {
	appropriate := s.CalcLayerSet()
	if s.LayerSet != appropriate {
		s.LayerSet = appropriate
		if DEBUG {
			log.Println("switching layer set", appropriate)
		}
	}
	return appropriate
}

func (s *Session) BestGuess(ts ...Throw) Guess {
	if len(ts) == 0 {
		panic("no throws")
	}
	t1 := time.Now() // evaluation

	s.Throws = ts
	s.Scores, s.TotalScore = s.Layers().SumScores(s.Throws)

	if len(s.Scores) == 0 {
		return Guess{Method: "reset"}
	}
	if s.TotalScore == 0 {
		panic("no score")
	}
	if s.TotalScore < 0 {
		panic("negative score")
	}

	t2 := time.Now() // clustering
	chunks := s.Chunks()
	averageScore := s.TotalScore / len(chunks)
	pts := make([]dbscan.Clusterable, 0, len(chunks))
	counted := 0
	for _, c := range chunks {
		if s.Scores[c] < averageScore {
			continue
		}
		counted++
		score := s.Scores[c]
		pts = append(pts, ScoredChunk{c, score})
	}
	// log.Println("observing", counted, "/", len(chunks), "above average", averageScore)
	clustered := dbscan.Clusterize(pts, 1, s.LayerSet.ClusterWeight)
	clusterGroups := make(map[int]clusters.Cluster)
	allowOutliers := true
	maxScore := 0
	for id, c := range clustered {
		group := clusters.Cluster{}
		group.Center = []float64{0, 0}

		totalScore := 0
		avgRing := 0

		for _, point := range c {
			scored := point.(ScoredChunk)
			score := scored.Score
			chunk := scored.Chunk

			x, y := chunk.Center()
			group.Center[0] += float64(x * score)
			group.Center[1] += float64(y * score)
			totalScore += score
			if score > maxScore {
				maxScore = score
			}
			avgRing += RingID(chunk)
			kobs := clusters.Coordinates([]float64{float64(x), float64(y)})
			group.Observations = append(group.Observations, kobs)
		}
		group.Center[0] /= float64(totalScore)
		group.Center[1] /= float64(totalScore)
		avgRing /= len(c)

		clusterGroups[id] = group
		if len(c) > 1 {
			allowOutliers = false
		}
		if DEBUG {
			log.Println("cluster", id, "size", len(c), "ringavg", avgRing, "center", group.Center)
		}
	}

	t3 := time.Now() // choosing cluster

	display := make(clusters.Clusters, 0, len(clusterGroups))
	for _, c := range clusterGroups {
		if len(c.Observations) == 1 && !allowOutliers {
			continue
		}
		display = append(display, c)
	}
	if DEBUG {
		log.Println("chunks", len(pts), "clusters", len(display))
	}

	if len(display) == 0 {
		panic(fmt.Sprintf(`%t, %#v = %#v`, allowOutliers, display, clusterGroups))
	}

	// pick cluster closest to player
	t := s.Throws[len(s.Throws)-1]
	leastFar := display[0]
	leastFound := dist(leastFar.Center[0], leastFar.Center[1], t.X, t.Y)
	if len(display) > 1 {
		for _, c := range display[1:] {
			dist := dist(c.Center[0], c.Center[1], t.X, t.Y)
			if dist < leastFound {
				leastFound = dist
				leastFar = c
			}
		}
	}

	if DEBUG {
		kmeans.SimplePlotter{}.Plot(display, 0)
		log.Println("plotting closest cluster", leastFar.Center, leastFound)
	}

	t4 := time.Now() // choosing chunk
	sx, sy := leastFar.Center[0], leastFar.Center[1]
	highest := chunks[0]
	highestScore := 0

	closest := chunks[0]
	closestDistance := closest.Dist(sx, sy) / float64(s.Scores[closest])
	for _, c := range chunks {
		dist := c.Dist(sx, sy) / float64(s.Scores[c])
		if dist < closestDistance {
			closest = c
			closestDistance = dist
		}
		score := s.Scores[c]
		if score > highestScore {
			highestScore = score
			highest = c
		}
	}

	if DEBUG {
		log.Println("total score", s.TotalScore)
		l := 20
		scored := s.ByScore()
		if len(scored) < l {
			l = len(scored)
		}
		for _, chunk := range s.ByScore()[:l] {
			log.Println("chunk", chunk, "score", s.Scores[chunk])
		}

		log.Println("highest score chunk", highest, "/", maxScore)
	}

	if DEBUG {
		log.Printf(`scoring:%s clustering:%s rendering:%s choosing:%s`, t2.Sub(t1), t3.Sub(t2), t4.Sub(t3), time.Since(t4))
	}
	return Guess{
		Chunk:      closest,
		Confidence: s.Scores[closest] * 1000 / (s.TotalScore + 2),
		Method:     s.Layers().Code,
	}
}
