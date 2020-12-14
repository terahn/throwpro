package throwlib

import (
	"math"
	"math/rand"
)

func GenStrongholds(worldSeed int64) (positions [][2]int) {
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
		positions = append(positions, [2]int{chunkX, chunkY})
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
