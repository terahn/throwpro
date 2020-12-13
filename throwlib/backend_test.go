package throwlib

import (
	"log"
	"math"
	"strconv"
	"strings"
	"testing"
)

type progressionTest struct {
	throws []Throw
	goal   Chunk
}

const TUNE_COUNT = 100

var progressionTests = append([]progressionTest{
	{
		throws: []Throw{NewThrowFromArray([3]float64{294.96, -486.85, -499.05}),
			NewThrowFromArray([3]float64{362.90, -669.03, -493.95}),
			NewThrowFromArray([3]float64{467.60, -843.82, -488.70})},
		goal: ChunkFromCenter(936, -1224),
	},
	{
		throws: []Throw{NewThrowFromArray([3]float64{-456.90, 120.37, -752.41}),
			NewThrowFromArray([3]float64{-237.07, 508.18, -753.61}),
			NewThrowFromArray([3]float64{-109.32, 640.59, -751.96})},
		goal: ChunkFromCenter(536, 1672),
	},
	{
		throws: []Throw{NewThrowFromArray([3]float64{-241.27, 283.87, -125.85}),
			NewThrowFromArray([3]float64{-43.73, 252.43, -128.85}),
			NewThrowFromArray([3]float64{63.99, 198.62, -129.60})},
		goal: ChunkFromCenter(1352, -872),
	},
}, loadTestsFromString(sample1)...)

func TestTriangulationAccuracy(t *testing.T) {
	distances := []int{0, 0, 0, 0}
	totals := []int{0, 0, 0, 0}
	for n, test := range progressionTests {
		DEBUG_CHUNK = test.goal
		sess := NewSession()
		throws := test.throws
		if len(throws) > 3 {
			throws = throws[:3]
		}
		for num := range throws {
			bestGuess := Chunk(sess.BestGuess(throws[:num+1]...).Chunk)
			chunkDist := int(bestGuess.ChunkDist(test.goal))

			if chunkDist > 10000 {
				t.Logf("bad test %d result %#v, guessed %s", n, test, bestGuess)
			}
			distances[num+1] += chunkDist
			totals[num+1]++
		}

		throw := NewBlindThrow(throws[0].X, throws[0].Y)
		bestGuess := Chunk(sess.BestGuess(throw).Chunk)
		chunkDist := int(bestGuess.ChunkDist(test.goal))

		if chunkDist > 10000 {
			t.Logf("bad test result %#v, guessed %s", test, bestGuess)
		}
		distances[0] += chunkDist
		totals[0]++
	}

	for throw, score := range distances {
		score = score / totals[throw]
		t.Logf("average throw %d accuracy for %d samples: %d blocks", throw, totals[throw], score)
	}
}

func TestTuneZeroConfigs(t *testing.T) {
	ls := ZeroEyeSet
	acc, _ := AverageAccuracy(ls, 0)
	log.Println("better params", ls, acc)
	for i := 0; i < TUNE_COUNT; i++ {
		test := ls.Mutate()
		newACC, _ := AverageAccuracy(test, 0)
		// log.Println("trying params", test, newACC)
		if newACC < acc {
			ls = test
			acc = newACC
			log.Println("better params", ls, acc)
		}
	}
}

func TestTuneOneConfigs(t *testing.T) {
	ls := OneEyeSet
	acc, _ := AverageAccuracy(ls, 1)
	log.Println("better params", ls, acc)
	for i := 0; i < TUNE_COUNT; i++ {
		test := ls.Mutate()
		newACC, _ := AverageAccuracy(test, 1)
		// log.Println("trying params", test, newACC)
		if newACC < acc {
			ls = test
			acc = newACC
			log.Println("better params", ls, acc)
		}
	}
}

func TestTuneTwoConfigs(t *testing.T) {
	ls := TwoEyeSet
	acc, _ := AverageAccuracy(ls, 2)
	log.Println("better params", ls, acc)
	for i := 0; i < TUNE_COUNT; i++ {
		test := ls.Mutate()
		newACC, _ := AverageAccuracy(test, 2)
		// log.Println("trying params", test, newACC)
		if newACC < acc {
			ls = test
			acc = newACC
			log.Println("better params", ls, acc)
		}
	}
}

func AverageAccuracy(ls LayerSet, total int) (float64, int) {
	distances := 0.0
	totals := 0.0
	for _, test := range progressionTests {
		// DEBUG_CHUNK = test.goal
		sess := NewSession(ls)
		throws := test.throws
		if len(throws) < total {
			continue
		}
		var bestGuess Chunk
		if total == 0 {
			bestGuess = Chunk(sess.BestGuess(NewBlindThrow(throws[0].X, throws[0].Y)).Chunk)
		} else {
			if len(throws) > total {
				throws = throws[:total]
			}
			bestGuess = Chunk(sess.BestGuess(throws...).Chunk)
		}
		distances += bestGuess.ChunkDist(test.goal)
		totals++
	}

	return distances / totals, int(totals)
}

func TestEducatedAccuracy(t *testing.T) {
	distance := 0
	for _, test := range progressionTests {
		DEBUG_CHUNK = test.goal
		throw := test.throws[0]
		goal := test.goal

		sess := NewSession()
		guess := Chunk(sess.BestGuess(throw).Chunk)
		chunkDist := int(guess.ChunkDist(goal))
		distance += chunkDist / len(progressionTests)
		t.Logf("goal %s, guess %s", goal, guess)
	}
	t.Logf("average educated accuracy: %d blocks", distance)
}

func TestDeterministic(t *testing.T) {
	for _, test := range progressionTests[:3] {
		DEBUG_CHUNK = test.goal
		throw := test.throws[0]

		guess1 := NewSession().BestGuess(throw)
		guess2 := NewSession().BestGuess(throw)

		if guess1 != guess2 {
			t.Errorf("mismatching guesses")
		}
	}
}

func TestProgression(t *testing.T) {
	test := progressionTests[1]
	DEBUG_CHUNK = test.goal
	DEBUG = true
	sess := NewSession()

	throw := NewBlindThrow(test.throws[0].X, test.throws[0].Y)
	guess := Chunk(sess.BestGuess(throw).Chunk)

	t.Logf("current angle: %f", guess.Angle(throw.A, throw.X, throw.Y))
	t.Logf("throw %d matched %d, educated guess: %s, goal: %s", 0, len(sess.Scores), guess, test.goal)

	for n, throw := range test.throws {
		g := sess.BestGuess(test.throws[:n+1]...)
		guess := Chunk(g.Chunk)

		t.Logf("current angle: %f", guess.Angle(throw.A, throw.X, throw.Y))
		t.Logf("throw %d matchd %d, config %v", n+1, len(sess.Scores), throw)
		t.Logf("throw %d educated guess: %s, goal: %s", n+1, g, test.goal)
	}
}

func loadTestsFromString(s string) []progressionTest {
	test := progressionTest{}
	tests := make([]progressionTest, 0)
	skips := 0
	for _, str := range strings.Split(s, "\n") {
		if strings.HasPrefix(str, "/execute") {
			t, err := NewThrowFromString(str)
			if err != nil {
				panic(err)
			}
			test.throws = append(test.throws, t)
		}
		if strings.HasPrefix(str, "/tp") {
			new := test
			test = progressionTest{}

			parts := strings.Split(str, " ")
			a, b := parts[2], parts[4]
			x, _ := strconv.Atoi(a)
			y, _ := strconv.Atoi(b)
			new.goal = ChunkFromCenter(x, y)

			// filter out extremely bad throws
			skip := false
			for _, t := range new.throws {
				if t.Type == Blind {
					skip = true
					log.Println("skipping blind throw", t, str)
					continue
				}
				delta := math.Abs(new.goal.Angle(t.A, t.X, t.Y))
				if delta > radsFromDegs(1) {
					skip = true
					log.Println("skipping bad throw", t, str)
				}
			}
			if len(new.throws) == 0 {
				log.Println("skipping no throws", str)
				skip = true
			}
			if !skip {
				tests = append(tests, new)
			} else {
				skips++
			}
		}
	}
	log.Println("skipped", skips, "bad samples")
	return tests
}

const sample1 = `
personal collection:
/execute in minecraft:overworld run tp @s 171.83 84.51 131.77 306.60 -32.25
/execute in minecraft:overworld run tp @s 289.18 84.51 52.34 310.20 -27.00
/tp @s 1928 ~ 1432
/execute in minecraft:overworld run tp @s -999.50 84.51 500.50 70.65 -33.45
/execute in minecraft:overworld run tp @s -1147.76 84.51 440.20 64.35 -31.35
/tp @s -2008 ~ 856
/execute in minecraft:overworld run tp @s -264.36 87.01 -352.48 49.50 -31.65
/execute in minecraft:overworld run tp @s -579.65 93.01 -192.50 45.00 -31.05
/tp @s -1304 ~ 536
/execute in minecraft:overworld run tp @s 1000.50 86.54 -192.50 -25.65 -31.65
/execute in minecraft:overworld run tp @s 1000.19 100.03 84.02 -29.70 -32.40
/tp @s 1848 ~ 1560
/execute in minecraft:overworld run tp @s -535.86 126.53 -189.05 135.00 -31.50
/execute in minecraft:overworld run tp @s -635.67 111.54 -480.23 125.25 -30.60
/tp @s -1320 ~ -968
/execute in minecraft:overworld run tp @s 974.17 100.00 1033.01 48.30 -30.00
/execute in minecraft:overworld run tp @s 969.49 100.00 1165.66 52.65 -30.90
/tp @s -8 ~ 1912
/execute in minecraft:overworld run tp @s 326.38 99.97 -559.47 235.20 -31.05
/execute in minecraft:overworld run tp @s 697.72 99.97 -773.35 232.20 -31.20
/tp @s 1256 ~ -1208
/execute in minecraft:overworld run tp @s -864.93 99.97 21.17 130.20 -31.65
/execute in minecraft:overworld run tp @s -1041.26 86.47 -48.11 134.40 -27.15
/tp @s -1624 ~ -616
/execute in minecraft:overworld run tp @s -351.13 112.21 2246.84 109.35 -32.40
/execute in minecraft:overworld run tp @s -467.01 112.21 2137.93 90.45 -25.80
/execute in minecraft:overworld run tp @s -612.55 112.21 2048.49 30.15 -29.85
/tp @s -664 ~ 2136
/execute in minecraft:overworld run tp @s 70.27 96.91 771.62 358.95 -28.95
/execute in minecraft:overworld run tp @s -50.34 96.91 1277.49 345.15 -31.05
/execute in minecraft:overworld run tp @s -69.16 96.91 1440.67 336.45 -30.45
/execute in minecraft:overworld run tp @s 3.65 96.91 1646.68 331.20 -30.90
/execute in minecraft:overworld run tp @s 142.77 96.91 1824.68 474.45 -30.15
/execute in minecraft:overworld run tp @s 21.90 96.91 1868.91 583.65 -31.50
/tp @s 88 ~ 1800

Stronghold data from BadSap:
/execute in minecraft:overworld run tp @s 109.30 80.00 -152.32 -272.10 -28.50
/execute in minecraft:overworld run tp @s 34.65 94.00 -192.28 -273.30 -29.40
/tp @s -1592 ~ -104
/execute in minecraft:overworld run tp @s -1609.12 95.22 3905.94 10.05 -30.90
/execute in minecraft:overworld run tp @s -1773.80 72.00 3956.33 2.70 -32.55
/tp @s -1848 ~ 5400
/execute in minecraft:overworld run tp @s 2153.50 62.84 2405.10 95.10 -30.30
/execute in minecraft:overworld run tp @s 2091.59 63.60 2453.38 95.85 -36.15
/tp @s 1064 ~ 2296
/execute in minecraft:overworld run tp @s 2064.03 78.77 3301.19 289.65 -31.65
/execute in minecraft:overworld run tp @s 2157.59 77.27 3351.91 294.00 -39.75
/tp @s 2968 ~ 3624
/execute in minecraft:overworld run tp @s 8157.59 86.65 3351.91 528.00 -31.35
/execute in minecraft:overworld run tp @s 8138.07 80.27 3095.26 528.00 -35.70
/tp @s 7736 ~ 1352
/execute in minecraft:overworld run tp @s 8138.07 75.77 95.26 17.85 -31.80
/execute in minecraft:overworld run tp @s 8044.60 75.77 160.71 3.75 -37.35
/tp @s 7736 ~ 1352
/execute in minecraft:overworld run tp @s -9999.50 86.14 -9999.50 210.30 -31.80
/execute in minecraft:overworld run tp @s -9711.06 95.55 -10056.97 194.25 -31.80
/tp @s -9512 ~ -10824
/execute in minecraft:overworld run tp @s -14242.23 88.52 -15855.76 656.25 -30.75
/execute in minecraft:overworld run tp @s -14165.17 88.52 -15802.00 652.36 -43.65
/tp @s -12984 ~ -15240
/execute in minecraft:overworld run tp @s -9983.50 93.02 -15239.50 361.66 -32.10
/execute in minecraft:overworld run tp @s -10146.84 81.02 -15098.99 700.21 -31.95
/tp @s -10008 ~ -13528
/execute in minecraft:overworld run tp @s -9912.51 106.14 -10629.50 244.21 -31.05
/execute in minecraft:overworld run tp @s -9707.24 76.00 -10594.30 220.51 -31.50
/tp @s -9512 ~ -10824
/execute in minecraft:overworld run tp @s 10000.50 73.36 10000.50 244.36 -30.45
/execute in minecraft:overworld run tp @s 10119.60 73.36 10011.48 238.96 -31.95
/tp @s 11176 ~ 9432
/execute in minecraft:overworld run tp @s 3000.50 81.25 9430.75 479.11 -30.75
/execute in minecraft:overworld run tp @s 2945.21 81.25 9284.58 475.51 -30.15
/tp @s 1032 ~ 8344
/execute in minecraft:overworld run tp @s 900.78 104.99 4078.96 556.21 -33.45
/execute in minecraft:overworld run tp @s 789.73 110.61 4029.96 552.91 -43.05
/tp @s 1064 ~ 2296
/execute in minecraft:overworld run tp @s 117.85 91.51 -219.61 62.40 -30.30
/execute in minecraft:overworld run tp @s 10.05 99.00 -211.91 412.05 -39.30
/tp @s -1400 ~ 584
/execute in minecraft:overworld run tp @s -2395.74 79.89 -370.44 -46.95 -31.50
/execute in minecraft:overworld run tp @s -2361.24 63.40 -236.55 -48.75 -31.65
/tp @s -1400 ~ 584
/execute in minecraft:overworld run tp @s 3000.50 71.28 3000.50 146.40 -31.35
/execute in minecraft:overworld run tp @s 2862.53 70.00 2918.34 150.00 -31.35
/tp @s 2104 ~ 1640
/execute in minecraft:overworld run tp @s 7096.06 95.00 -3363.67 -41.85 -31.95
/execute in minecraft:overworld run tp @s 7220.42 123.00 -3235.26 -41.40 -31.95
/tp @s 7704 ~ -2680
/execute in minecraft:overworld run tp @s 4014.49 72.64 -2686.24 188.70 -34.35
/execute in minecraft:overworld run tp @s 4076.53 59.00 -2701.43 198.00 -37.35
/tp @s 4424 ~ -3640
/execute in minecraft:overworld run tp @s 10035.91 68.00 9971.27 359.40 -31.65
/execute in minecraft:overworld run tp @s 10126.74 96.00 10247.04 443.85 -39.30
/tp @s 10040 ~ 10264
/execute in minecraft:overworld run tp @s 9864.58 113.13 -19999.70 363.90 -47.70
/execute in minecraft:overworld run tp @s 10016.65 89.00 -19759.20 370.95 -30.90
/tp @s 9768 ~ -18488
/execute in minecraft:overworld run tp @s 9768.50 85.11 -8487.50 -125.70 -31.80
/execute in minecraft:overworld run tp @s 9820.35 68.00 -8623.89 -123.00 -31.95
/tp @s 11128 ~ -9464
/execute in minecraft:overworld run tp @s 7128.50 85.14 -9463.50 97.35 -31.05
/execute in minecraft:overworld run tp @s 6949.48 69.00 -9330.87 102.60 -31.50
/tp @s 5224 ~ -9704
/execute in minecraft:overworld run tp @s 4987.90 82.00 -5760.14 161.10 -31.50
/execute in minecraft:overworld run tp @s 4855.70 67.00 -5850.59 168.00 -31.80
/tp @s 4664 ~ -6712
/execute in minecraft:overworld run tp @s 4607.07 80.24 -2721.89 -191.25 -30.45
/execute in minecraft:overworld run tp @s 4389.86 68.00 -2928.40 -177.30 -30.75
/tp @s 4424 ~ -3640
/execute in minecraft:overworld run tp @s -9943.73 86.00 9993.32 -36.90 -31.20
/execute in minecraft:overworld run tp @s -9788.40 80.00 10043.49 -31.51 -31.50
/tp @s -9384 ~ 10728
/execute in minecraft:overworld run tp @s -5379.42 78.62 10734.23 -78.76 -31.50
/execute in minecraft:overworld run tp @s -5313.15 77.00 10486.68 -70.51 -31.65
/tp @s -3304 ~ 11128
/execute in minecraft:overworld run tp @s 701.14 67.82 11116.77 -60.01 -30.75
/execute in minecraft:overworld run tp @s 830.11 66.00 11052.38 -52.96 -31.65
/tp @s 1640 ~ 11656
/execute in minecraft:overworld run tp @s 5640.50 73.62 11656.50 -38.26 -31.35
/execute in minecraft:overworld run tp @s 5802.28 73.62 11690.22 -17.71 -36.75
/tp @s 6792 ~ 13128
/execute in minecraft:overworld run tp @s 25.48 84.00 329.95 252.15 -32.70
/execute in minecraft:overworld run tp @s 138.66 80.50 351.01 599.85 -49.50
/tp @s 1432 ~ -120
/execute in minecraft:overworld run tp @s 3432.50 70.00 1880.50 240.15 -30.60
/execute in minecraft:overworld run tp @s 3424.70 70.00 1811.74 258.60 -31.80
/tp @s 4536 ~ 1240
/execute in minecraft:overworld run tp @s 6594.75 68.75 3237.94 -39.00 -37.05
/execute in minecraft:overworld run tp @s 6732.49 87.49 3222.60 -25.95 -40.95
/tp @s 7656 ~ 4056
/execute in minecraft:overworld run tp @s 9656.50 87.49 6056.50 -173.40 -32.25
/execute in minecraft:overworld run tp @s 9696.93 87.49 6019.51 -179.25 -36.45
/tp @s 9752 ~ 5272
/execute in minecraft:overworld run tp @s 11713.20 86.79 7297.95 289.94 -32.40
/execute in minecraft:overworld run tp @s 11967.30 94.04 7321.98 294.74 -30.60
/tp @s 12760 ~ 7688
/execute in minecraft:overworld run tp @s 14760.50 91.80 9688.50 551.09 -31.95
/execute in minecraft:overworld run tp @s 14859.41 91.80 9669.92 518.54 -33.30
/tp @s 14984 ~ 8552
/execute in minecraft:overworld run tp @s 16905.62 79.80 10551.09 -78.31 -31.80
/execute in minecraft:overworld run tp @s 16937.72 79.80 10478.38 -53.86 -28.80
/tp @s 17864 ~ 10744
/execute in minecraft:overworld run tp @s 19764.54 70.62 12766.29 -223.35 -30.30
/execute in minecraft:overworld run tp @s 19708.98 66.49 12771.30 -239.40 -34.50
/tp @s 17864 ~ 10744
/execute in minecraft:overworld run tp @s -1999.50 67.63 -1999.50 -76.35 -30.60
/execute in minecraft:overworld run tp @s -1938.05 70.00 -1942.86 -81.45 -31.20
/tp @s -1400 ~ -1864
/execute in minecraft:overworld run tp @s -3411.52 72.98 -3826.83 -121.51 -24.45
/execute in minecraft:overworld run tp @s -3283.70 77.00 -3846.30 -120.16 -42.90
/tp @s -1384 ~ -5080`
