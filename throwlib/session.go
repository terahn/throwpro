package throwlib

import (
	"fmt"
	"log"
	"math"
	"sort"

	dbscan "github.com/866/go-dbscan"
	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

var DEBUG = false

var DEBUG_CHUNK Chunk

const CHUNK_GROUP = 2000

var ZeroEyeSet = LayerSet{
	Code: "blind",
	Name: "Blind Travel",

	AnglePref:       radsFromDegs(0.22),
	RingMod:         107,
	AverageDistance: 0.4,
	MathFactor:      54,
}

var OneEyeSet = LayerSet{
	Code: "educated",
	Name: "Educated Travel",

	AnglePref:       radsFromDegs(0.17),
	RingMod:         107,
	AverageDistance: 0.51,
	MathFactor:      47,
}

var TwoEyeSet = LayerSet{
	Code: "triangulation",
	Name: "Gradual Triangulation",

	AnglePref:       radsFromDegs(0.09),
	RingMod:         100,
	AverageDistance: 0.27,
	MathFactor:      40,
}

type ChunkList []Chunk

type Chunk [2]int

func (t Chunk) ChunkDist(other Chunk) float64 {
	ax, ay := t.Center()
	bx, by := other.Center()
	return dist(float64(ax), float64(ay), float64(bx), float64(by))
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
	atan := math.Atan2(x, y) + math.Pi*2
	atan = math.Mod(atan, math.Pi*2)
	return Throw{Type: Blind, A: atan}
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

func (s *Session) NewThrow(t Throw) *Session {
	s.Throws = append(s.Throws, t)
	s.Scores, s.TotalScore = s.Layers().SumScores(s.Throws)
	return s
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
			// continue
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

		totalScore := 0
		for _, point := range c {
			chunk := point.(Chunk)

			score := s.Scores[chunk]
			x, y := chunk.Center()
			group.Center[0] += float64(x * score)
			group.Center[1] += float64(y * score)
			totalScore += score

			kobs := clusters.Coordinates([]float64{float64(x), float64(y)})
			group.Observations = append(group.Observations, kobs)
		}
		group.Center[0] /= float64(totalScore)
		group.Center[1] /= float64(totalScore)

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
		l := 20
		scored := s.ByScore()
		if len(scored) < l {
			l = len(scored)
		}
		for _, chunk := range s.ByScore()[:l] {
			log.Println("chunk", chunk, "score", s.Scores[chunk])
		}
	}

	return closest, s.Scores[closest] * 1000 / (s.TotalScore + 2)
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
