package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

func TestWs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ws.Connect(ctx, rpc.MainNetBeta_WS)
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

	received := 0
	for {
		select {
		case <-ctx.Done():
			if received == 0 {
				t.Fatalf("Test timed out after 30 seconds without receiving any updates")
			} else {
				fmt.Printf("Test completed: received %d slot updates\n", received)
				return
			}
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

			received++
			fmt.Printf("timestamp: %d\n", time.Now().Unix())
			fmt.Printf("slot received: %d\n", slot.Slot)

			if received >= 1 {
				fmt.Println("Successfully received slot update!")
				return
			}
		}
	}
}
