package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type BloxrouteClient struct {
	Url        string
	AuthHeader string
}

func NewBloxrouteClient(url string, authHeader string) *BloxrouteClient {
	return &BloxrouteClient{
		Url:        url,
		AuthHeader: authHeader,
	}
}

type bloxrouteRequest struct {
	Transaction struct {
		Content string `json:"content"`
	} `json:"transaction"`
	SkipPreFlight          bool `json:"skipPreFlight"`
	FrontRunningProtection bool `json:"frontRunningProtection"`
	FastBestEffort         bool `json:"fastBestEffort"`
	UseStakedRPCs          bool `json:"useStakedRPCs"`
	Sniping                bool `json:"sniping"`
	AllowBackRun           bool `json:"allowBackRun"`
}

type bloxrouteResponse struct {
	Signature string `json:"signature"`
}

func (c *BloxrouteClient) SendTransaction(ctx context.Context, txBase64 string) (string, error) {
	reqData := bloxrouteRequest{}
	reqData.Transaction.Content = txBase64
	reqData.SkipPreFlight = true
	reqData.FrontRunningProtection = false
	reqData.FastBestEffort = false
	reqData.UseStakedRPCs = true

	// what is this?
	reqData.AllowBackRun = true
	reqData.Sniping = true

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
		return "", errors.New("bloxroute request failed with status: " + resp.Status + ", body: " + string(body))
	}

	var response bloxrouteResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.Signature, nil
}
