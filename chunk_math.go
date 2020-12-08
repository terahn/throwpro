package throwpro

import (
	"fmt"
	"math"
)

const minDist = 1408
const maxDist = 2688
const ringMod = 300

var angleDiff = radsFromDegs(0.15)

func ChunkFromCenter(x, y int) Chunk {
	return Chunk{(x - modLikePython(x, 16)) / 16, (y - modLikePython(y, 16)) / 16}
}

func ChunkFromPosition(x, y float64) Chunk {
	return Chunk{(int(x) - modLikePython(int(x), 16)) / 16, (int(y) - modLikePython(int(y), 16)) / 16}
}

func (c Chunk) Staircase() (int, int) {
	x, y := c.Center()
	return x - 4, y - 4
}

func (c Chunk) String() string {
	x, y := c.Center()
	return fmt.Sprintf("chunk %d,%d (center %d, %d)", c[0], c[1], x, y)
}

func (c Chunk) Score(a, sx, sy float64) int {
	score := 7

	delta := math.Abs(c.Angle(a, sx, sy))
	if delta > angleDiff*7 {
		return 0
	}

	if delta > angleDiff {
		score--
	}
	if delta > angleDiff*2 {
		score--
	}
	if delta > angleDiff*3 {
		score--
	}
	if delta > angleDiff*5 {
		score--
	}

	cDist := c.Dist(0, 0)
	if cDist < minDist {
		return 0
	}
	if cDist > maxDist {
		return 0
	}

	spawn := math.Max(0, math.Min(.06, .06*dist(0, 0, sx, sy)/300))
	preferred := minDist + (maxDist-minDist)*.165 + spawn
	ring := cDist - preferred
	if ring > ringMod {
		score--
	}
	if ring > ringMod*2 {
		score--
	}
	if ring > ringMod*3 {
		score--
	}
	return score
}

func dist(x, y, x2, y2 float64) float64 {
	dx := x - x2
	dy := y - y2
	return math.Sqrt(dx*dx + dy*dy)
}

func (c Chunk) Dist(x, y float64) float64 {
	cx, cy := c.Center()
	return dist(float64(cx), float64(cy), x, y)
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

func (c Chunk) Ring() int {
	preferred := minDist + (maxDist-minDist)*.2
	dist := c.Dist(0, 0) - preferred
	return int(math.Abs(dist))
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

	chunks := make(ChunkList, 0)
	chunksFound := map[Chunk]bool{}
	for {
		blockX := int(math.Floor(cx))
		blockY := int(math.Floor(cy))

		centerX := modLikePython(blockX, 16)
		centerY := modLikePython(blockY, 16)

		for xo := -1; xo < 1; xo++ {
			for yo := -1; yo < 1; yo++ {
				chunk := Chunk{(blockX-centerX)/16 + xo, (blockY-centerY)/16 + yo}
				if _, found := chunksFound[chunk]; !found {
					chunksFound[chunk] = true
					chunks = append(chunks, chunk)
				}
			}
		}

		lastDist := Chunk{0, 0}.Dist(cx, cy)
		cx += dx * 2
		cy += dy * 2
		newDist := Chunk{0, 0}.Dist(cx, cy)
		if newDist > lastDist && newDist > maxDist {
			break
		}
	}
	return chunks
}

func modLikePython(d, m int) int {
	var res int = d % m
	if (res < 0 && m > 0) || (res > 0 && m < 0) {
		return res + m
	}
	return res
}
