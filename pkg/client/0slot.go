package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Slot0Client struct {
	Url    string
	ApiKey string
}

func NewSlot0Client(url string, apiKey string) *Slot0Client {
	return &Slot0Client{
		Url:    url,
		ApiKey: apiKey,
	}
}

func (c *Slot0Client) SendTransaction(ctx context.Context, txBase64 string) (string, error) {
	type requestParams struct {
		Jsonrpc string        `json:"jsonrpc"`
		Id      string        `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	type responseBody struct {
		Jsonrpc string          `json:"jsonrpc"`
		Id      string          `json:"id"`
		Result  string          `json:"result,omitempty"`
		Error   json.RawMessage `json:"error,omitempty"`
	}

	requestID := strconv.FormatInt(time.Now().Unix(), 10)
	req := requestParams{
		Jsonrpc: "2.0",
		Id:      requestID,
		Method:  "sendTransaction",
		Params: []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?api-key=%s", c.Url, c.ApiKey)
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		url,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("unexpected status code: " + resp.Status)
	}

	var response responseBody
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("RPC error: " + string(response.Error))
	}

	return response.Result, nil
}
