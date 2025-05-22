package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type TritonClient struct {
	url    string
	client *http.Client
}

func NewTritonClient(baseUrl, apiKey string) *TritonClient {
	url := baseUrl
	if apiKey != "" {
		url = baseUrl + "/" + apiKey
	}

	return &TritonClient{
		url:    url,
		client: &http.Client{},
	}
}

func (c *TritonClient) SendTransaction(ctx context.Context, txBase64 string, _ bool) (string, error) {
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
			map[string]interface{}{"encoding": "base64", "skipPreflight": true},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.url,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
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
