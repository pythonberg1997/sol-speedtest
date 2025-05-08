package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type NextblockClient struct {
	Url        string
	AuthHeader string
}

func NewNextblockClient(url string, authHeader string) *NextblockClient {
	return &NextblockClient{
		Url:        url,
		AuthHeader: authHeader,
	}
}

type nextblockRequest struct {
	Transaction struct {
		Content string `json:"content"`
	} `json:"transaction"`
	FrontRunningProtection bool `json:"frontRunningProtection"`
}

type nextblockResponse struct {
	Signature string `json:"signature"`
	Uuid      string `json:"uuid"`
}

func (c *NextblockClient) SendTransaction(ctx context.Context, txBase64 string, antiMev bool) (string, error) {
	reqData := nextblockRequest{}
	reqData.Transaction.Content = txBase64
	reqData.FrontRunningProtection = false

	if antiMev {
		reqData.FrontRunningProtection = true
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.Url+"/api/v2/submit", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthHeader != "" {
		req.Header.Set("Authorization", c.AuthHeader)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("nextblock request failed with status: " + resp.Status + ", body: " + string(body))
	}

	var response nextblockResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.Signature, nil
}
