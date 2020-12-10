package throwpro

import (
	"fmt"
	"math"
)

var rings = [][2]int{{1408, 2688}, {4480, 5760}, {7552, 8832}, {10624, 11904}, {13696, 14976}, {16768, 18048}, {19840, 21120}, {22912, 24192}}

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

func RingID(c Chunk) int {
	cDist := c.Dist(0, 0)
	for n, ring := range rings {
		minDist, maxDist := float64(ring[0]), float64(ring[1])
		if cDist < minDist-240 {
			continue
		}
		if cDist > maxDist+240 {
			continue
		}
		return n
	}
	return -1
}

func OneEyeSet() LayerSet {
	return LayerSet{
		// AnglePref: radsFromDegs(.2),
		// RingMod:   300,
		AnglePref: radsFromDegs(2),
		RingMod:   150,
	}
}

func TwoEyeSet() LayerSet {
	return LayerSet{
		// AnglePref: radsFromDegs(.2),
		// RingMod:   300,
		AnglePref: radsFromDegs(1.45),
		RingMod:   282,
	}
}

type LayerSet struct {
	AnglePref float64
	RingMod   float64
}

func (ls LayerSet) Layers() []func(Throw, Chunk) int {
	return []func(Throw, Chunk) int{ls.Angle, ls.Ring, ls.Preference}
}

func (ls LayerSet) Ring(t Throw, c Chunk) int {
	ringID := RingID(c)
	if ringID == -1 {
		return 0
	}
	cDist := c.Dist(0, 0)
	minDist, maxDist := float64(rings[ringID][0]), float64(rings[ringID][1])
	preferred := minDist + (maxDist-minDist)*.2
	ring := cDist - preferred
	if ring < ls.RingMod {
		return 3
	}
	if ring < ls.RingMod*2 {
		return 2
	}
	return 1
}

func (ls LayerSet) Preference(t Throw, c Chunk) int {
	dist := c.Dist(t.X, t.Y)
	if dist < ls.RingMod*9 {
		return 4
	}
	if dist < ls.RingMod*15 {
		return 3
	}
	if dist < ls.RingMod*21 {
		return 2
	}
	if dist < ls.RingMod*27 {
		return 1
	}
	return 0
}

func (ls LayerSet) Angle(t Throw, c Chunk) int {
	delta := math.Abs(c.Angle(t.A, t.X, t.Y))
	if delta < ls.AnglePref {
		return 4
	}
	if delta < ls.AnglePref*2 {
		return 3
	}
	if delta > ls.AnglePref*3 {
		return 2
	}
	if delta > ls.AnglePref*5 {
		return 1
	}
	return 0
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
				if RingID(chunk) == -1 {
					continue
				}
				if _, found := chunksFound[chunk]; !found {
					chunksFound[chunk] = true
					chunks = append(chunks, chunk)
				}
			}
		}

		lastDist := dist(0, 0, cx, cy)
		cx += dx * 4
		cy += dy * 4
		newDist := dist(0, 0, cx, cy)
		if newDist > lastDist && newDist > float64(rings[len(rings)-1][1]+240) {
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
