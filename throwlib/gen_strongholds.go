package throwlib

import (
	"math"
	"math/rand"
)

func ClosestStronghold(seed int64, t Throw) Chunk {
	DistanceChunks := 32
	TotalCount := 20 // only need like 20
	CountInRing := 3

	random := rand.New(rand.NewSource(seed))
	angle := random.Float64() * math.Pi * 2
	placedInRing := 0
	currentRing := 0

	var closest Chunk
	closestDist := 10000000.0
	for num := 0; num < TotalCount; num++ {
		dist := float64(4*DistanceChunks + DistanceChunks*currentRing*6)
		dist += (random.Float64() - 0.5) * float64(DistanceChunks) * 2.5
		chunkX := int(math.Round(math.Cos(angle) * dist))
		chunkY := int(math.Round(math.Sin(angle) * dist))
		c := Chunk{chunkX, chunkY}
		if c.Dist(t.X, t.Y) < closestDist {
			closestDist = c.Dist(0, 0)
			closest = c
		}
		angle += math.Pi * 2.0 / float64(CountInRing)
		placedInRing++
		if placedInRing == CountInRing {
			currentRing++
			placedInRing = 0
			CountInRing = CountInRing + 2*CountInRing/(currentRing+1)
			if CountInRing > TotalCount-num {
				CountInRing = TotalCount - num
			}
			angle += random.Float64() * math.Pi * 2.0
		}
	}
	return closest
}

func GenStrongholds(worldSeed int64) (positions []Chunk) {
	DistanceChunks := 32
	TotalCount := 128
	CountInRing := 3

	random := rand.New(rand.NewSource(worldSeed))
	angle := random.Float64() * math.Pi * 2
	placedInRing := 0
	currentRing := 0

	for num := 0; num < TotalCount; num++ {
		dist := float64(4*DistanceChunks + DistanceChunks*currentRing*6)
		dist += (random.Float64() - 0.5) * float64(DistanceChunks) * 2.5
		chunkX := int(math.Round(math.Cos(angle)*dist))*16 + 8
		chunkY := int(math.Round(math.Sin(angle)*dist))*16 + 8
		positions = append(positions, Chunk{chunkX, chunkY})
		angle += math.Pi * 2.0 / float64(CountInRing)
		placedInRing++
		if placedInRing == CountInRing {
			currentRing++
			placedInRing = 0
			CountInRing = CountInRing + 2*CountInRing/(currentRing+1)
			if CountInRing > TotalCount-num {
				CountInRing = TotalCount - num
			}
			angle += random.Float64() * math.Pi * 2.0
		}
	}
	return positions
}
