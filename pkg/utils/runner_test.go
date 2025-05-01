package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go/rpc/ws"
)

func TestWs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ws.Connect(ctx, "wss://mainnet.helius-rpc.com/?api-key=4dd6c987-f5b9-4302-ac47-4cf509b2e534")
	if err != nil {
		t.Fatalf("failed to connect to node: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected to websocket endpoint, subscribing to slot updates...")

	slotSub, err := client.SlotsUpdatesSubscribe()
	if err != nil {
		t.Fatalf("failed to subscribe to slot updates: %v", err)
	}
	defer slotSub.Unsubscribe()
	fmt.Println("Subscription created, waiting for updates...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			slot, err := slotSub.Recv(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				fmt.Printf("failed to receive slot: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			fmt.Printf("timestamp: %d\n", time.Now().Unix())
			fmt.Printf("slot received: %d\n", slot.Slot)
		}
	}
}
