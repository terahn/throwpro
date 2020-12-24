package throwlib

import (
	"log"
	"math"
	"math/rand"
)

const MAX_EYE_ANGLE = 0.85

const SELECTION_EFFECT = true

var rings = [][2]int{{1408, 2688}, {4480, 5760}, {7552, 8832}, {10624, 11904}, {13696, 14976}, {16768, 18048}, {19840, 21120}, {22912, 24192}}
var counts = []int{3, 6, 10, 15, 21, 28, 36, 9}

func ChunkFromCenter(x, y int) Chunk {
	return Chunk{(x - modLikePython(x, 16)) / 16, (y - modLikePython(y, 16)) / 16}
}

func ChunkFromPosition(x, y float64) Chunk {
	return Chunk{(int(x) - modLikePython(int(x), 16)) / 16, (int(y) - modLikePython(int(y), 16)) / 16}
}

func (c Chunk) Selectable(fromX, fromY float64) int {
	ring := RingID(c)
	if ring == -1 {
		return 0
	}

	count := counts[ring]
	x, y := c.Center()
	distPlayer := c.Dist(fromX, fromY)

	atan := math.Atan2(-float64(x), float64(y))
	inc := math.Pi * 2.0 / float64(count)
	if c == DEBUG_CHUNK {
		log.Println("-> selectable", c, "inc", inc)
	}

	score := 10
	for a := atan - inc; a <= atan+inc; a += inc * 2 {
		// a = neighboring spokes
		dx, dy := -math.Sin(a), math.Cos(a)
		for buffer := -120; buffer <= 120; buffer += 60 {
			// buffer 120 blocks around max
			d := float64(rings[ring][1] + buffer)
			ox, oy := dx*d-fromX, dy*d-fromY
			altDistPlayer := math.Sqrt(ox*ox + oy*oy)
			// every time this stronghold would be closer, subtract a point
			if distPlayer > altDistPlayer {
				if c == DEBUG_CHUNK {
					log.Println("-> distance", altDistPlayer, "<", distPlayer)
				}
				score--
				// if at max buffer, discard this chunk entirely
				if buffer == 120 {
					if DEBUG {
						log.Printf("deselected by %.0f,%.0f... %.0f < %.0f", ox, oy, altDistPlayer, distPlayer)
					}
					return 0
				}
			}
		}
	}
	return score
}

func RingID(c Chunk) int {
	cDist := c.Dist(0, 0)
	for n, ring := range rings {
		minDist, maxDist := float64(ring[0]), float64(ring[1])
		if cDist < minDist-110 {
			continue
		}
		if cDist > maxDist+110 {
			continue
		}
		return n
	}
	return -1
}

type Layer func([]Throw, Chunk) int

type LayerSet struct {
	Code string

	AnglePref       float64
	RingMod         float64
	AverageDistance float64
	MathFactor      float64
	ClusterWeight   float64

	Weights [3]int
}

var ZeroEyeSet = LayerSet{
	Code: "blind",

	AnglePref:       radsFromDegs(0.08),
	RingMod:         33,
	AverageDistance: 0.22,
	Weights:         [3]int{20, 100, 0},
	ClusterWeight:   150,
}

var OneEyeSet = LayerSet{
	Code: "educated",

	AnglePref:       radsFromDegs(0.005),
	RingMod:         100,
	AverageDistance: 0.61,
	MathFactor:      440,
	Weights:         [3]int{40, 100, 0},
	ClusterWeight:   180,
}

var TwoEyeSet = LayerSet{
	Code: "triangulation",

	AnglePref:       radsFromDegs(0.06),
	RingMod:         150,
	AverageDistance: 0.5,
	MathFactor:      38,
	Weights:         [3]int{20, 10, 100},
	ClusterWeight:   180,
}

var HyperSet = LayerSet{
	Code: "hyper",

	AnglePref:       radsFromDegs(0.01),
	RingMod:         0,
	AverageDistance: 0.5,
	MathFactor:      4,
	Weights:         [3]int{100, 5, 100},
	ClusterWeight:   150,
}

func (ls LayerSet) SumScores(throws []Throw) (map[Chunk]int, int) {
	scores := make(map[Chunk]int)
	reject := make(map[Chunk]bool)
	count := make(map[Chunk]int)

	layers := ls.Layers()
	for _, t := range throws {
		chunks := ChunksInThrow(t)
		for _, c := range chunks {
			count[c]++
		}
	}
	for c := range count {
		out := c == DEBUG_CHUNK
		if reject[c] {
			if out {
				log.Println("-> sumscore: goal already rejected")
			}
			continue
		}
		score := 0
		for n, l := range layers {
			s := l(throws, c)
			if s < 0 {
				log.Println("sumscore: goal score", s, "for layer", n, "weight", ls.Weights[n])
				panic("negative score")
			}
			if out {
				log.Println("-> sumscore: goal score", s, "for layer", n)
			}
			if s == 0 {
				score = 0
				break
			}
			score += s * ls.Weights[n]
		}
		if score == 0 {
			reject[c] = true
			if _, f := scores[c]; f {
				delete(scores, c)
			}
			if out {
				log.Println("sumscore: goal sum rejected")
			}
			continue
		}
		scores[c] += score
		if out {
			log.Println("sumscore: goal earned", score)
		}
	}
	highest := 0
	total := 0
	for c := range count {
		total += scores[c]
		if scores[c] > highest {
			highest = scores[c]
		}
	}

	if DEBUG {
		log.Println("summed scores, total", len(count), "matched", len(scores), "rejected", len(reject), "highscore", highest)
	}
	return scores, total
}

func (ls LayerSet) Mutate() LayerSet {
	factor := 0.50
	eff := (rand.Float64() - .5) * 2 * factor
	switch rand.Intn(5) {
	case 0:
		ls.AnglePref *= 1 + eff
	case 1:
		ls.AverageDistance *= 1 + eff
	case 2:
		ls.RingMod *= 1 + eff
	case 3:
		ls.MathFactor *= 1 + eff
	case 4:
		ls.ClusterWeight *= 1 + eff
	}
	return ls
}

func (ls LayerSet) Layers() []Layer {
	return []Layer{ls.Angle, ls.Ring, ls.CrossAngle}
}

func (ls LayerSet) Ring(t []Throw, c Chunk) int {
	ringID := RingID(c)
	if ringID == -1 {
		return 0
	}

	total := 1
	for _, t := range t {
		sel := c.Selectable(t.X, t.Y)
		if c == DEBUG_CHUNK {
			log.Println("-> ls.sel:", sel)
		}
		if sel == 0 {
			if c == DEBUG_CHUNK {
				DEBUG = true
				c.Selectable(t.X, t.Y)
				panic("discarded debug chunk")
			}
			if SELECTION_EFFECT {
				return 0
			}
		}
		if SELECTION_EFFECT {
			total += sel
		}
	}

	cDist := c.Dist(0, 0)
	minDist, maxDist := float64(rings[ringID][0]), float64(rings[ringID][1])
	preferred := minDist + (maxDist-minDist)*ls.AverageDistance
	ring := cDist - preferred
	if ring < ls.RingMod {
		total++
	}
	if ring < ls.RingMod*2 {
		total++
	}
	if ring < ls.RingMod*3 {
		total++
	}
	return total
}

func (ls LayerSet) Angle(ts []Throw, c Chunk) int {
	total := 1
	for _, t := range ts {
		delta := math.Abs(c.Angle(t.A, t.X, t.Y))
		if delta > radsFromDegs(MAX_EYE_ANGLE) {
			if c == DEBUG_CHUNK {
				log.Println("-> ls.angle: discarded", delta)
			}
			return 0
		}
		if delta < ls.AnglePref {
			total++
		}
		if delta < ls.AnglePref*2 {
			total++
		}
		if delta < ls.AnglePref*4 {
			total++
		}
		if delta < ls.AnglePref*6 {
			total++
		}
		if delta < ls.AnglePref*9 {
			total++
		}
	}
	if c == DEBUG_CHUNK {
		log.Println("-> ls.angle:"+ls.Code+" goal scored", total)
	}
	return total / len(ts)
}

const CROSSANGLE_EXPERIMENT = true

func (ls LayerSet) CrossAngle(ts []Throw, c Chunk) int {
	if len(ts) <= 1 {
		return 1
	}
	printout := rand.Intn(10000) == 0
	if c == DEBUG_CHUNK {
		printout = true
	}
	if !DEBUG {
		printout = false
	}
	score := 1

	tx, ty := 0.0, 0.0
	count := 0
	for n, t := range ts[:len(ts)-1] {
		for _, ot := range ts[n+1:] {
			k := ((ot.Y-t.Y)*math.Sin(ot.A) + (ot.X-t.X)*math.Cos(ot.A)) / math.Sin(ot.A-t.A)
			ny := t.Y + k*math.Cos(t.A)
			nx := t.X - k*math.Sin(t.A)

			tx += nx
			ty += ny
			count++

			distFromPerfect := c.Dist(nx, ny)

			if distFromPerfect < ls.MathFactor {
				score++
			}
			if distFromPerfect < ls.MathFactor*5 {
				score++
			}
			if distFromPerfect < ls.MathFactor*12 {
				score++
			}
			if distFromPerfect < ls.MathFactor*25 {
				score++
			}

			if !printout {
				continue
			}

			log.Printf("crossangle: %s crossangle %.1f %.1f dist %.1f", c, nx, ny, distFromPerfect)
		}
	}
	tx /= float64(count)
	ty /= float64(count)

	distFromPerfect := c.Dist(tx, ty)

	if printout {
		log.Printf("crossangle: goal %s crossangle %.1f %.1f dist %.1f", c, tx, ty, distFromPerfect)
	}

	if CROSSANGLE_EXPERIMENT {
		if distFromPerfect < ls.MathFactor {
			return 7
		}
		if distFromPerfect < ls.MathFactor*2 {
			return 6
		}
		if distFromPerfect < ls.MathFactor*4 {
			return 5
		}
		if distFromPerfect < ls.MathFactor*8 {
			return 4
		}
		if distFromPerfect < ls.MathFactor*16 {
			return 3
		}
		if distFromPerfect < ls.MathFactor*32 {
			return 2
		}
		if distFromPerfect < ls.MathFactor*64 {
			return 1
		}
		return 0
	}

	return score / (len(ts) + 1)
}

func dist(x, y, x2, y2 float64) float64 {
	dx := x - x2
	dy := y - y2
	return math.Sqrt(dx*dx + dy*dy)
}

func radsFromDegs(degs float64) float64 {
	return wrapRads(degs * (math.Pi / 180))
}

func wrapRads(rads float64) float64 {
	for rads < math.Pi {
		rads += math.Pi * 2
	}
	for rads > math.Pi {
		rads -= math.Pi * 2
	}
	return rads
}

func ChunksInThrow(t Throw) ChunkList {
	angle := t.A
	cx, cy := t.X, t.Y
	dx, dy := -math.Sin(angle), math.Cos(angle)
	lastDist := dist(0, 0, cx, cy)

	chunks := make(ChunkList, 0)
	chunksFound := map[Chunk]bool{}

	pRing := t.RingID()

	scanIters := 0
	for {
		blockX := int(math.Floor(cx))
		blockY := int(math.Floor(cy))

		centerX := modLikePython(blockX, 16)
		centerY := modLikePython(blockY, 16)

		for xo := -1; xo <= 1; xo++ {
			for yo := -1; yo <= 1; yo++ {
				chunk := Chunk{(blockX-centerX)/16 + xo, (blockY-centerY)/16 + yo}
				if _, found := chunksFound[chunk]; found {
					continue
				}
				chunksFound[chunk] = true
				ringID := RingID(chunk)
				if ringID == -1 {
					if chunk == DEBUG_CHUNK {
						log.Println("-> goal chunk out of ring")
					}
					continue
				}
				if ringID < pRing-1 || ringID > pRing+1 {
					if chunk == DEBUG_CHUNK {
						log.Println("-> goal chunk out of player ring", ringID, pRing)
						log.Println(t)
					}
					continue
				}

				chunks = append(chunks, chunk)
			}
		}

		nextX := (blockX/16)*16 - 16
		if dx > 0 {
			nextX += 32
		}
		nextY := (blockY/16)*16 - 16
		if dy > 0 {
			nextY += 32
		}
		distX, distY := math.Inf(1), math.Inf(1)
		if dx != 0 {
			distX = (float64(nextX) - cx) / dx
		}
		if dy != 0 {
			distY = (float64(nextY) - cy) / dy
		}
		useX := math.Abs(distX) < math.Abs(distY)
		if distX == 0 {
			useX = false
		}
		if distY == 0 {
			useX = true
		}
		if useX {
			cx += dx * distX
			cy += dy * distX
		} else {
			cx += dx * distY
			cy += dy * distY
		}

		// break

		newDist := dist(0, 0, cx, cy)
		if newDist > lastDist && newDist > float64(rings[len(rings)-1][1]+240) {
			break
		}
		scanIters++
		if scanIters > 10000 {
			log.Println(blockX, blockY, nextX, nextY, distX, distY)
		}
		if scanIters > 10050 {
			panic("overscanning")
		}
	}
	// log.Println("scan iterations:", scanIters)
	return chunks
}

func modLikePython(d, m int) int {
	var res int = d % m
	if (res < 0 && m > 0) || (res > 0 && m < 0) {
		return res + m
	}
	return res
}
