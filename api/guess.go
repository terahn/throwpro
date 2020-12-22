package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dantoye/throwpro/throwlib"
)

func guess(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	res := events.APIGatewayProxyResponse{}

	throws := throwlib.Request{}
	if err := json.Unmarshal([]byte(req.Body), &throws); err != nil {
		return res, err
	}

	if len(throws.Clips) == 0 {
		res.Body = `{"method":"reset"}`
		res.StatusCode = 400
		return res, nil
	}

	if len(throws.Clips) > 5 {
		res.Body = `{"method":"reset"}`
		res.StatusCode = 400
		return res, nil
	}

	response := throwlib.NewResponse(throws)
	enc, _ := json.MarshalIndent(response, "", "\t")
	res.Body = string(enc)
	res.StatusCode = 200
	return res, nil
}

func main() {
	lambda.Start(guess)
}
