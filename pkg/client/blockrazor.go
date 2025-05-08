package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "sol-speedtest/pkg/client/serverpb"
)

type BlockrazorClient struct {
	Url     string
	AuthKey string
}

func NewBlockrazorClient(url string, authKey string) *BlockrazorClient {
	return &BlockrazorClient{
		Url:     url,
		AuthKey: authKey,
	}
}

func (c *BlockrazorClient) SendTransaction(ctx context.Context, txBase64 string, _ bool) (string, error) {
	conn, err := grpc.NewClient(c.Url,
		// grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), // 對接通用端點時開啟tls配置
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 對接區域端點時使用
		grpc.WithPerRPCCredentials(&Authentication{c.AuthKey}),
	)
	if err != nil {
		return "", fmt.Errorf("connect error: %v", err)
	}

	client := pb.NewServerClient(conn)
	client.GetHealth(context.Background(), &pb.HealthRequest{})

	sendRes, err := client.SendTransaction(context.TODO(), &pb.SendRequest{
		Transaction:   txBase64,
		Mode:          "fast",
		SkipPreflight: true,
	})
	if err != nil {
		return "", fmt.Errorf("send tx error: %v", err)
	}

	return sendRes.Signature, nil
}

type Authentication struct {
	apiKey string
}

func (a *Authentication) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"apiKey": a.apiKey}, nil
}

func (a *Authentication) RequireTransportSecurity() bool {
	return false
}
