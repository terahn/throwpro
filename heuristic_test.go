package throwpro

import "testing"

var heuristicTests = [][5]float64{
	{-214.79, 386.16, 76.50, -1608, 728},
	{320.18, 255.34, -53.40, 1240, 936},
	{454.38, -319.63, -188.55, 248, -1688},
	{-87.85, -434.11, 575.85, 504, -1256},
	{-1003.81, 170.63, 448.94, -2600, 200},
	{-146.06, 457.92, 668.39, 1192, 1528},
}

func TestHeuristics(t *testing.T) {
testing:
	for n, heuristic := range heuristicTests {
		sess := NewSession(NewThrow(heuristic[0], heuristic[1], heuristic[2]))
		goal := ChunkFromCenter(int(heuristic[3]), int(heuristic[4]))
		found := sess.Sorted()
		for _, f := range found {
			if f.Chunk == goal {
				continue testing
			}
		}
		t.Errorf("test %d failed, stronghold %s not found", n, goal)
		t.Logf("chunk had dist of %f", goal.Dist(0, 0))
		for _, f := range found {
			x, y := f.Center()
			if goal.Dist(float64(x), float64(y)) < 20 {
				t.Logf("did find %s nearby", f)
			}
		}
	}
}

type progressionTest struct {
	a, b, c Throw
	goal    Chunk
}

var progressionTests = []progressionTest{
	{
		a:    NewThrowFromArray([3]float64{294.96, -486.85, -499.05}),
		b:    NewThrowFromArray([3]float64{362.90, -669.03, -493.95}),
		c:    NewThrowFromArray([3]float64{467.60, -843.82, -488.70}),
		goal: ChunkFromCenter(936, -1224),
	},
	{
		a:    NewThrowFromArray([3]float64{-456.90, 120.37, -752.41}),
		b:    NewThrowFromArray([3]float64{-237.07, 508.18, -753.61}),
		c:    NewThrowFromArray([3]float64{-109.32, 640.59, -751.96}),
		goal: ChunkFromCenter(536, 1672),
	},
	{
		a:    NewThrowFromArray([3]float64{-241.27, 283.87, -125.85}),
		b:    NewThrowFromArray([3]float64{-43.73, 252.43, -128.85}),
		c:    NewThrowFromArray([3]float64{63.99, 198.62, -129.60}),
		goal: ChunkFromCenter(1352, -872),
	},
}

func TestProgression(t *testing.T) {
	test := progressionTests[2]
	sess := NewSession(test.a)
	runnerUp := Chunk{56, -75}
	t.Logf("throw 1 had %d chunks, stronghold at %s", len(sess.Scores), test.goal)

	for _, throw := range []Throw{test.a, test.b, test.c} {
		if throw != test.a {
			matches, discards := sess.AddThrow(throw)
			t.Logf("throw matched %d chunked, discarded %d chunks", matches, discards)
		}
		guesses := sess.Sorted()
		highScore := sess.Scores[guesses[0].Chunk]
		for _, c := range guesses {
			if sess.Scores[c.Chunk] < highScore-1 {
				break
			}
			t.Logf("%s score %d confidence %d", c, sess.Scores[c.Chunk], c.Confidence)
			t.Logf("current angle: %f", c.Angle(throw.A, throw.X, throw.Y))
			t.Logf("current score: %d", c.Score(throw.A, throw.X, throw.Y))
			t.Logf("current ring: %d", c.Ring())
		}
		for _, c := range guesses {
			if c.Chunk == test.goal {
				t.Logf("%s score %d confidence %d", c, sess.Scores[c.Chunk], c.Confidence)
				t.Logf("goal angle: %f", c.Angle(throw.A, throw.X, throw.Y))
				t.Logf("goal score: %d", c.Score(throw.A, throw.X, throw.Y))
				t.Logf("goal ring: %d", c.Ring())
			}
			if c.Chunk == runnerUp {
				t.Logf("%s score %d confidence %d", c, sess.Scores[c.Chunk], c.Confidence)
				t.Logf("runnerUp angle: %f", c.Angle(throw.A, throw.X, throw.Y))
				t.Logf("runnerUp score: %d", c.Score(throw.A, throw.X, throw.Y))
				t.Logf("runnerUp ring: %d", c.Ring())
			}
		}
		t.Logf("educated guess: %s", guesses.String())
	}
}
