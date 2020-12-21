package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestGuess(t *testing.T) {
	req := events.APIGatewayProxyRequest{}
	req.Body = `{"clips":[
		"/execute in minecraft:overworld run tp @s 294.96 116.93 -486.85 -499.05 -25.35",
		"/execute in minecraft:overworld run tp @s -456.90 116.93 120.37 -752.41 -31.65"
	]}`
	res, err := guess(req)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(res.Body)
}
