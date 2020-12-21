package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dantoye/throwpro/throwlib"
)

func guess(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	type requestJSON struct {
		Clips []string
	}

	type responseJSON struct {
		Guess throwlib.Guess `json:"guess"`
		Notes []string       `json:"notes"`
	}

	res := events.APIGatewayProxyResponse{}

	clips := requestJSON{}
	if err := json.Unmarshal([]byte(req.Body), &clips); err != nil {
		return res, err
	}

	response := responseJSON{}
	throws := make([]throwlib.Throw, 0, len(clips.Clips))
	for _, clip := range clips.Clips {
		throw, err := throwlib.NewThrowFromString(clip)
		if err != nil {
			response.Notes = append(response.Notes, err.Error())
			continue
		}
		throws = append(throws, throw)
	}

	if len(throws) == 0 {
		res.Body = `{"method":"reset"}`
		res.StatusCode = 400
		return res, nil
	}

	sess := throwlib.NewSession()
	guess := sess.BestGuess(throws...)
	enc, _ := json.MarshalIndent(guess, "", "\t")
	res.Body = string(enc)
	res.StatusCode = 200
	return res, nil
}

func main() {
	lambda.Start(guess)
}
