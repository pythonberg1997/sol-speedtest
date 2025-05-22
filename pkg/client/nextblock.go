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
	baseUrl    string
	authHeader string
	client     *http.Client
}

func NewNextblockClient(url string, authHeader string) *NextblockClient {
	return &NextblockClient{
		baseUrl:    url + "/api/v2/submit",
		authHeader: authHeader,
		client:     &http.Client{},
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

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}

	resp, err := c.client.Do(req)
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
