package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"

	"sol-speedtest/models"
	"sol-speedtest/pkg/client"
	"sol-speedtest/pkg/config"
	"sol-speedtest/pkg/logger"
	"sol-speedtest/pkg/metrics"
)

type TestRunner struct {
	config         *config.Config
	rpcClient      *rpc.Client
	signer         solana.PrivateKey
	nonceAccount   solana.PublicKey
	collector      *metrics.Collector
	providerConfig map[string]config.ProviderConfig
	testInterval   time.Duration
	logger         *logger.Logger
}

type EndpointInfo struct {
	ProviderName string
	URL          string
	TipAmount    uint64
	TipAccount   string
}

type NonceRequest struct {
	Nonce         solana.Hash
	TestIndex     int
	ResultChan    chan<- *models.TransactionTest
	SuccessChan   chan bool
	SuccessMu     *sync.Mutex
	SuccessSignal *bool
}

func NewTestRunner(cfg *config.Config, privKey string, logLevel logger.LogLevel) *TestRunner {
	providerConfig := make(map[string]config.ProviderConfig)
	for _, p := range cfg.Providers {
		providerConfig[p.Name] = p
	}

	rpcClient := rpc.New(cfg.RPCURL)
	signer := solana.MustPrivateKeyFromBase58(privKey)
	nonceAccount := solana.MustPublicKeyFromBase58(cfg.NonceAccount)

	log := logger.GetDefaultLogger()
	logDir := "logs"
	if cfg.LogFilePath != "" {
		logDir = cfg.LogFilePath
	}

	rotationConfig := logger.DefaultRotationConfig()
	if cfg.LogRotation.MaxSizeMB > 0 {
		rotationConfig.MaxSize = int64(cfg.LogRotation.MaxSizeMB) * 1024 * 1024
	}
	if cfg.LogRotation.MaxAgeHours > 0 {
		rotationConfig.MaxAge = time.Duration(cfg.LogRotation.MaxAgeHours) * time.Hour
	}
	if cfg.LogRotation.MaxBackups > 0 {
		rotationConfig.MaxBackups = cfg.LogRotation.MaxBackups
	}
	rotationConfig.RotateOnTime = cfg.LogRotation.RotateOnTime

	err := logger.InitGlobalLoggerWithRotation(logDir, logLevel, rotationConfig)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize logger: %v. Logs will be sent to stderr.\n", err)
	}

	return &TestRunner{
		config:         cfg,
		rpcClient:      rpcClient,
		signer:         signer,
		nonceAccount:   nonceAccount,
		collector:      metrics.NewCollector(cfg.OutputPath),
		providerConfig: providerConfig,
		testInterval:   400 * time.Millisecond,
		logger:         log,
	}
}

func (r *TestRunner) SetTestInterval(interval time.Duration) {
	r.testInterval = interval
}

func (r *TestRunner) RunTests() error {
	var endpoints []EndpointInfo
	endpointChannels := make(map[string]chan NonceRequest)

	r.logger.Info("Initializing test run with %d providers", len(r.providerConfig))

	var wg sync.WaitGroup
	for name, cfg := range r.providerConfig {
		providerLogger := r.logger.WithProvider(name)
		providerLogger.Info("Setting up provider with %d endpoints", len(cfg.Endpoints))

		for _, endpoint := range cfg.Endpoints {
			var cli client.Client
			switch cfg.Name {
			case "0xslot":
				apiKey := os.Getenv("SLOT0")
				if apiKey == "" {
					providerLogger.Error("0xslot API key not set in environment variables")
					continue
				}
				cli = client.NewSlot0Client(endpoint.URL, apiKey)
			// case "blockRazor":
			// 	cli = client.NewBlockrazorClient(endpoint.URL, cfg.ApiKey)
			case "bloXroute":
				apiKey := os.Getenv("BLOX_ROUTE")
				if apiKey == "" {
					providerLogger.Error("bloXroute API key not set in environment variables")
					continue
				}
				cli = client.NewBloxrouteClient(endpoint.URL, apiKey)
			case "nextBlock":
				apiKey := os.Getenv("NEXT_BLOCK")
				if apiKey == "" {
					providerLogger.Error("nextBlock API key not set in environment variables")
					continue
				}
				cli = client.NewNextblockClient(endpoint.URL, apiKey)
			default:
				providerLogger.Error("Unsupported provider: %s", cfg.Name)
			}

			endpointInfo := EndpointInfo{
				ProviderName: name,
				URL:          endpoint.URL,
				TipAmount:    cfg.TipAmount,
				TipAccount:   endpoint.TipAccount,
			}

			endpoints = append(endpoints, endpointInfo)
			endpointKey := fmt.Sprintf("%s|%s", name, endpoint.URL)

			providerLogger.Info("Adding endpoint %s with tip account %s", endpoint.URL, endpoint.TipAccount)

			requestChan := make(chan NonceRequest, r.config.TestCount)
			endpointChannels[endpointKey] = requestChan

			wg.Add(1)
			go r.endpointWorker(cli, endpointInfo, requestChan, &wg)
		}
	}

	resultsChan := make(chan *models.TransactionTest, r.config.TestCount*len(endpoints))

	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go r.collectResults(resultsChan, &resultsWg)

	r.logger.Info("Starting %d test rounds with %d ms interval across %d endpoints",
		r.config.TestCount, int64(r.testInterval/time.Millisecond), len(endpoints))

	for i := 0; i < r.config.TestCount; i++ {
		nonce := r.getNonce()
		r.logger.Debug("Round %d: Got nonce %s", i, nonce.String())

		successSignal := false
		var successMu sync.Mutex
		successChan := make(chan bool, len(endpoints))

		for endpointKey, ch := range endpointChannels {
			r.logger.Debug("Round %d: Sending nonce %s to endpoint %s", i, nonce.String(), endpointKey)
			ch <- NonceRequest{
				Nonce:         nonce,
				TestIndex:     i,
				ResultChan:    resultsChan,
				SuccessChan:   successChan,
				SuccessMu:     &successMu,
				SuccessSignal: &successSignal,
			}
		}

		time.Sleep(r.testInterval)
		close(successChan)

		if (i+1)%100 == 0 {
			r.logger.Info("Completed %d/%d test rounds", i+1, r.config.TestCount)
		}
	}

	r.logger.Info("All test rounds sent, waiting for endpoints to complete")

	for _, ch := range endpointChannels {
		close(ch)
	}

	wg.Wait()
	close(resultsChan)
	resultsWg.Wait()

	r.logger.Info("Test run completed, saving results")
	return r.collector.SaveResults()
}

func (r *TestRunner) getNonce() solana.Hash {
	nonceAccount, err := r.rpcClient.GetAccountInfo(context.Background(), r.nonceAccount)
	if err != nil {
		r.logger.Error("Error getting nonce account info: %v", err)
		return solana.Hash{}
	}
	if nonceAccount == nil {
		r.logger.Error("Nonce account not found")
		return solana.Hash{}
	}
	nonceData := nonceAccount.Value.Data.GetBinary()[40:72]

	nonce := solana.Hash(nonceData)
	r.logger.Debug("Retrieved nonce: %s", nonce.String())
	return nonce
}

func (r *TestRunner) endpointWorker(
	cli client.Client,
	endpoint EndpointInfo,
	requestChan <-chan NonceRequest,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	endpointLogger := r.logger.WithProvider(endpoint.ProviderName).WithEndpoint(endpoint.URL)
	endpointLogger.Info("Starting endpoint worker for %s", endpoint.URL)

	for req := range requestChan {
		req.SuccessMu.Lock()
		alreadySucceeded := *req.SuccessSignal
		req.SuccessMu.Unlock()

		if alreadySucceeded {
			endpointLogger.Debug("Skipping nonce %s due to existing success", req.Nonce.String())
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.Timeout)*time.Second)
		ctxWithCancel, cancelOp := context.WithCancel(ctx)

		go func() {
			select {
			case <-req.SuccessChan:
				endpointLogger.Debug("Transaction succeeded on another endpoint, canceling operation")
				cancelOp()
			case <-ctxWithCancel.Done():
				return
			}
		}()

		startTime := time.Now()
		test := models.TransactionTest{
			ProviderName: endpoint.ProviderName,
			URL:          endpoint.URL,
			TipAccount:   endpoint.TipAccount,
			Nonce:        req.Nonce.String(),
			StartTime:    startTime,
			Success:      false,
		}

		endpointLogger.Debug("Sending transaction with nonce %s", req.Nonce.String())
		txBase64, err := r.buildTransaction(endpoint.TipAmount, endpoint.TipAccount, req.Nonce)
		if err != nil {
			endpointLogger.Debug("Build tx failed: %s", test.Error)
			test.Error = "failed to build transaction"
			test.EndTime = time.Now()
			test.Duration = test.EndTime.Sub(startTime).Milliseconds()
			req.ResultChan <- &test
			cancel()
			cancelOp()
			continue
		}

		txHash, err := cli.SendTransaction(ctxWithCancel, txBase64)
		if errors.Is(ctxWithCancel.Err(), context.Canceled) {
			endpointLogger.Debug("Operation canceled: %s", test.Error)
			test.Error = "canceled due to successful transaction by another endpoint"
			test.EndTime = time.Now()
			test.Duration = test.EndTime.Sub(startTime).Milliseconds()
			req.ResultChan <- &test
			cancel()
			cancelOp()
			continue
		}
		if err != nil {
			endpointLogger.Error("Transaction failed: %v", err)
			test.Error = err.Error()
			test.EndTime = time.Now()
			test.Duration = test.EndTime.Sub(startTime).Milliseconds()
			req.ResultChan <- &test
			cancel()
			cancelOp()
			continue
		}

		test.TxHash = txHash
		endpointLogger.Debug("Transaction sent, hash: %s", txHash)
		confirmed, err := checkTransactionWithTimeout(ctxWithCancel, r.rpcClient, txHash, endpointLogger)
		if errors.Is(ctxWithCancel.Err(), context.Canceled) {
			endpointLogger.Debug("Confirmation check canceled: %s", test.Error)
			test.Error = "canceled due to successful transaction by another endpoint"
			test.EndTime = time.Now()
			test.Duration = test.EndTime.Sub(startTime).Milliseconds()
			req.ResultChan <- &test
			cancel()
			cancelOp()
			continue
		}

		test.EndTime = time.Now()
		test.Duration = test.EndTime.Sub(startTime).Milliseconds()

		if err != nil {
			test.Error = err.Error()
			endpointLogger.Error("Transaction confirmation failed: %v", err)
		} else {
			test.Success = confirmed
			if confirmed {
				endpointLogger.Info("Transaction confirmed in %d ms", test.Duration)
				req.SuccessMu.Lock()
				if !(*req.SuccessSignal) {
					*req.SuccessSignal = true
					for i := 0; i < cap(req.SuccessChan); i++ {
						select {
						case req.SuccessChan <- true:
						default:
						}
					}
					endpointLogger.Info("Signaled success for nonce %s", req.Nonce.String())
				}
				req.SuccessMu.Unlock()
			} else {
				endpointLogger.Warn("Transaction not confirmed after %d ms", test.Duration)
			}
		}

		req.ResultChan <- &test
		cancel()
		cancelOp()
	}

	endpointLogger.Info("Endpoint worker for %s completed", endpoint.URL)
}

func (r *TestRunner) collectResults(resultsChan <-chan *models.TransactionTest, wg *sync.WaitGroup) {
	defer wg.Done()

	r.logger.Info("Starting result collection")
	count := 0
	successCount := 0

	for test := range resultsChan {
		count++
		if test.Success {
			successCount++
			r.logger.Debug("Collected successful test result from %s: %s", test.ProviderName, test.TxHash)
		} else {
			r.logger.Debug("Collected failed test result from %s: %s", test.ProviderName, test.Error)
		}

		r.collector.AddTest(*test)

		if count%100 == 0 {
			r.logger.Info("Collected %d test results (%d successful)", count, successCount)
		}
	}

	r.logger.Info("Result collection complete: %d total results, %d successful", count, successCount)
}

func checkTransactionWithTimeout(
	ctx context.Context,
	rpcClient *rpc.Client,
	txHash string,
	log *logger.Logger,
) (bool, error) {
	signature, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		log.Error("Invalid transaction hash: %v", err)
		return false, fmt.Errorf("invalid transaction hash: %w", err)
	}

	pollInterval := 100 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Debug("Starting transaction status check for %s", txHash)
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			log.Debug("Transaction status check canceled: %v", ctx.Err())
			return false, ctx.Err()
		case <-ticker.C:
			attempts++
			log.Debug("Checking transaction status (attempt %d): %s", attempts, txHash)

			status, err := rpcClient.GetSignatureStatuses(
				ctx,
				true,
				signature,
			)
			if err != nil {
				log.Error("Error querying transaction status: %v", err)
				continue
			}

			if status == nil || len(status.Value) == 0 || status.Value[0] == nil {
				log.Debug("Transaction status not found for %s", txHash)
				continue
			}

			txStatus := status.Value[0]
			if txStatus.ConfirmationStatus != "" {
				if txStatus.ConfirmationStatus == rpc.ConfirmationStatusConfirmed {
					log.Debug("Transaction confirmed: %s", txHash)
					return true, nil
				}
			}

			if txStatus.Err != nil {
				log.Error("Transaction failed: %v", txStatus.Err)
				return false, fmt.Errorf("transaction failed: %v", txStatus.Err)
			}
		}
	}
}

func (r *TestRunner) buildTransaction(
	tipAmount uint64, tipAccount string, nonce solana.Hash,
) (string, error) {
	advanceNonceIx := system.NewAdvanceNonceAccountInstruction(
		r.nonceAccount,
		solana.SysVarRecentBlockHashesPubkey,
		r.signer.PublicKey(),
	).Build()

	tipIx := system.NewTransferInstruction(
		tipAmount,
		r.signer.PublicKey(),
		solana.MustPublicKeyFromBase58(tipAccount),
	).Build()

	tx, err := solana.NewTransaction(
		[]solana.Instruction{advanceNonceIx, tipIx},
		nonce,
		solana.TransactionPayer(r.signer.PublicKey()),
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		return &r.signer
	})
	if err != nil {
		return "", err
	}

	txBase64, err := tx.ToBase64()
	if err != nil {
		return "", fmt.Errorf("failed to convert transaction to base58: %w", err)
	}
	return txBase64, nil
}
