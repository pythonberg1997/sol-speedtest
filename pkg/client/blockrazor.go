package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "sol-speedtest/pkg/client/serverpb"
)

type BlockrazorClient struct {
	url     string
	authKey string
	conn    *grpc.ClientConn
	client  pb.ServerClient
}

func NewBlockrazorClient(url string, authKey string) (*BlockrazorClient, error) {
	conn, err := grpc.NewClient(url,
		// grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), // 對接通用端點時開啟tls配置
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 對接區域端點時使用
		grpc.WithPerRPCCredentials(&Authentication{authKey}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect error: %v", err)
	}

	client := pb.NewServerClient(conn)
	_, err = client.GetHealth(context.Background(), &pb.HealthRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("health check error: %v", err)
	}

	return &BlockrazorClient{
		url:     url,
		authKey: authKey,
		conn:    conn,
		client:  client,
	}, nil
}

func (c *BlockrazorClient) SendTransaction(ctx context.Context, txBase64 string, _ bool) (string, error) {
	sendRes, err := c.client.SendTransaction(ctx, &pb.SendRequest{
		Transaction:   txBase64,
		Mode:          "fast",
		SkipPreflight: true,
	})
	if err != nil {
		return "", fmt.Errorf("send tx error: %v", err)
	}

	return sendRes.Signature, nil
}

func (c *BlockrazorClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
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
