package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

func main() {
	privKey := os.Getenv("SIGNER")
	signer := solana.MustPrivateKeyFromBase58(privKey)
	client := rpc.New("https://solana-rpc.publicnode.com")

	nonceAccount, err := solana.NewRandomPrivateKey()
	if err != nil {
		fmt.Printf("failed to create nonce account, err: %v", err)
		return
	}

	file, err := os.Create("nonce_account_info.txt")
	if err != nil {
		fmt.Printf("failed to create file, err: %v", err)
		return
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "private key: %s\n", nonceAccount.String())
	if err != nil {
		fmt.Printf("failed to write to file, err: %v", err)
		return
	}
	_, err = fmt.Fprintf(file, "public key: %s\n", nonceAccount.PublicKey().String())
	if err != nil {
		fmt.Printf("failed to write to file, err: %v", err)
		return
	}
	fmt.Println("Nonce account information has been saved to nonce_account_info.txt")

	ctx := context.Background()
	nonceAccountMinimumBalance, err := client.GetMinimumBalanceForRentExemption(ctx, 80, rpc.CommitmentFinalized)
	if err != nil {
		fmt.Printf("failed to get minimum balance for nonce account, err: %v", err)
		return
	}

	recentBlockhashResponse, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		fmt.Printf("failed to get recent blockhash, err: %v", err)
		return
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewCreateAccountInstruction(
				nonceAccountMinimumBalance,
				80,
				solana.SystemProgramID,
				signer.PublicKey(),
				nonceAccount.PublicKey(),
			).Build(),
			system.NewInitializeNonceAccountInstruction(
				signer.PublicKey(),
				nonceAccount.PublicKey(),
				solana.SysVarRecentBlockHashesPubkey,
				solana.SysVarRentPubkey,
			).Build(),
		},
		recentBlockhashResponse.Value.Blockhash,
		solana.TransactionPayer(signer.PublicKey()),
	)
	if err != nil {
		fmt.Printf("failed to new a transaction, err: %v", err)
		return
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(signer.PublicKey()) {
			return &signer
		} else if key.Equals(nonceAccount.PublicKey()) {
			return &nonceAccount
		}
		return nil
	})

	_, err = client.SimulateTransaction(ctx, tx)
	if err != nil {
		fmt.Printf("failed to simulate transaction, err: %v", err)
		return
	}

	txHash, err := client.SendTransaction(ctx, tx)
	if err != nil {
		fmt.Printf("failed to send tx, err: %v", err)
		return
	}

	fmt.Println("txhash", txHash)

	// Also save transaction hash to the file
	_, err = fmt.Fprintf(file, "transaction hash: %s\n", txHash)
	if err != nil {
		fmt.Printf("failed to write txhash to file, err: %v", err)
		return
	}
}
