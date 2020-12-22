package throwlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// public free use key
const API_KEY = "b4chnoWQeR1pBtmDxslcTaReIdThofpR8QiUkiQ6"

func postRequest(body []byte) (*Response, error) {
	url := `https://4f3fvniy4f.execute-api.us-east-1.amazonaws.com/dev/guess`
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))

	req.Header.Add("x-api-key", API_KEY)

	cli := http.Client{Timeout: 500 * time.Millisecond}
	res, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	iRes := Response{}
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	if err := json.Unmarshal(out, &iRes); err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error using online mode, status code %d", res.StatusCode)
	}
	log.Println("used online mode, response", string(out))
	return &iRes, nil
}

func PostRequest(req Request, offline bool) Response {
	j, _ := json.Marshal(req)

	if !offline {
		res, err := postRequest(j)
		if err != nil {
			log.Println("error using online mode:", err.Error())
		} else {
			return *res
		}
	}

	iReq := Request{}
	json.Unmarshal(j, &iReq)
	iRes := NewResponse(iReq)
	out, _ := json.Marshal(iRes)

	res := Response{}
	json.Unmarshal(out, &res)
	return res
}
