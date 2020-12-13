package throwlib

import "testing"

var xTests = map[string]Throw{
	"/execute in minecraft:overworld run tp @s -214.79 104.61 386.16 76.50 -32.40":   NewThrow(-214.79, 386.16, 76.50),
	"/execute in minecraft:overworld run tp @s 320.18 104.61 255.34 -53.40 -32.55":   NewThrow(320.18, 255.34, -53.40),
	"/execute in minecraft:overworld run tp @s 454.38 104.61 -319.63 -188.55 -32.40": NewThrow(454.38, -319.63, -188.55),
	"/execute in minecraft:overworld run tp @s -87.85 107.54 -434.11 575.85 -31.80":  NewThrow(-87.85, -434.11, 575.85),
	"/execute in minecraft:overworld run tp @s -1003.81 131.53 170.63 448.94 -32.25": NewThrow(-1003.81, 170.63, 448.94),
	"/execute in minecraft:overworld run tp @s -146.06 131.53 457.92 668.39 -31.35":  NewThrow(-146.06, 457.92, 668.39),
	"/execute in minecraft:overworld run tp @s -146.06 131.53 457.92 668.39 -10.35":  {X: -146.06, Y: 457.92},
}

func TestParsing(t *testing.T) {
	for input, output := range xTests {
		res, err := NewThrowFromString(input)
		if err != nil {
			t.Errorf("failed test with error %s", err.Error())
			continue
		}
		if res != output {
			t.Errorf("failed test %#v != %#v for string '%s'", res, output, input)
		}
	}
}

func TestBlind(t *testing.T) {
	throw, _ := NewThrowFromString(`/execute in minecraft:overworld run tp @s -146.06 131.53 457.92 668.39 -10.35`)
	DEBUG = true
	guess, _ := NewSession().NewThrow(throw).BestGuess()
	x, y := guess.Center()
	t.Logf("%#v blind to %d %d", throw, x, y)
}
