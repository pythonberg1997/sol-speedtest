package client

import (
	"context"
)

type Client interface {
	SendTransaction(ctx context.Context, txBase64 string, antiMev bool) (string, error)
}
