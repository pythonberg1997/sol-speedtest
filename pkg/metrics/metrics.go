package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sol-speedtest/models"
)

// NonceResult tracks the results for a specific nonce across all providers
type NonceResult struct {
	Nonce              string
	SuccessfulProvider string
	AllTests           []*models.TransactionTest
}

// Collector collects and processes test metrics
type Collector struct {
	mu           sync.Mutex
	startTime    time.Time
	tests        []models.TransactionTest
	nonceResults map[string]*NonceResult // Track results by nonce
	outputPath   string
}

// NewCollector creates a new metrics collector
func NewCollector(outputPath string) *Collector {
	return &Collector{
		startTime:    time.Now(),
		tests:        make([]models.TransactionTest, 0),
		nonceResults: make(map[string]*NonceResult),
		outputPath:   outputPath,
	}
}

// AddTest adds a test result to the collector
func (c *Collector) AddTest(test models.TransactionTest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tests = append(c.tests, test)

	if _, exists := c.nonceResults[test.Nonce]; !exists {
		c.nonceResults[test.Nonce] = &NonceResult{
			Nonce:    test.Nonce,
			AllTests: make([]*models.TransactionTest, 0),
		}
	}

	testCopy := test
	c.nonceResults[test.Nonce].AllTests = append(c.nonceResults[test.Nonce].AllTests, &testCopy)

	if test.Success && c.nonceResults[test.Nonce].SuccessfulProvider == "" {
		c.nonceResults[test.Nonce].SuccessfulProvider = test.ProviderName
	}
}

// ProcessResults processes the test results and generates a report
func (c *Collector) ProcessResults() (*models.TestReport, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	endTime := time.Now()
	totalDuration := endTime.Sub(c.startTime).Milliseconds()

	report := &models.TestReport{
		StartTime:          c.startTime,
		EndTime:            endTime,
		TotalDuration:      totalDuration,
		SuccessfulTxHashes: []string{},
	}

	providerWins := make(map[string]int)
	for _, nonceResult := range c.nonceResults {
		if nonceResult.SuccessfulProvider != "" {
			providerWins[nonceResult.SuccessfulProvider]++
		}
	}

	totalRounds := len(c.nonceResults)
	report.TotalRounds = totalRounds

	providerMap := make(map[string]map[string][]models.TransactionTest)
	for _, test := range c.tests {
		endpointKey := test.URL + "|" + test.TipAccount

		if _, ok := providerMap[test.ProviderName]; !ok {
			providerMap[test.ProviderName] = make(map[string][]models.TransactionTest)
		}

		providerMap[test.ProviderName][endpointKey] = append(providerMap[test.ProviderName][endpointKey], test)

		if test.Success && test.TxHash != "" {
			report.SuccessfulTxHashes = append(report.SuccessfulTxHashes, test.TxHash)
		}
	}

	for providerName, endpointTests := range providerMap {
		winCount := providerWins[providerName]
		winRate := 0.0
		if totalRounds > 0 {
			winRate = float64(winCount) / float64(totalRounds) * 100
		}

		providerResult := models.ProviderResult{
			ProviderName: providerName,
			URLResults:   make([]models.URLTestResult, 0),
			WinCount:     winCount,
			WinRate:      winRate,
		}

		var totalSuccessCount int
		var totalDurationSum, minDuration, maxDuration int64
		var totalSlotDeltaSum, minSlotDelta, maxSlotDelta uint64

		minDuration = math.MaxInt64
		minSlotDelta = math.MaxUint64

		for endpointKey, tests := range endpointTests {
			urlAndTip := splitEndpointKey(endpointKey)

			urlResult := models.URLTestResult{
				URL:         urlAndTip[0],
				TipAccount:  urlAndTip[1],
				TestDetails: tests,
			}

			urlMinDuration := int64(math.MaxInt64)
			urlMaxDuration := int64(0)
			urlMinSlotDelta := uint64(math.MaxUint64)
			urlMaxSlotDelta := uint64(0)
			var urlSuccessCount int
			var urlDurationSum int64
			var urlSlotDeltaSum uint64

			for _, test := range tests {
				if test.Success {
					urlSuccessCount++
					duration := test.Duration
					urlDurationSum += duration
					urlSlotDeltaSum += test.SlotDelta

					if duration < urlMinDuration {
						urlMinDuration = duration
					}

					if duration > urlMaxDuration {
						urlMaxDuration = duration
					}

					if test.SlotDelta < urlMinSlotDelta {
						urlMinSlotDelta = test.SlotDelta
					}

					if test.SlotDelta > urlMaxSlotDelta {
						urlMaxSlotDelta = test.SlotDelta
					}
				}
			}

			totalSuccessCount += urlSuccessCount
			totalDurationSum += urlDurationSum
			totalSlotDeltaSum += urlSlotDeltaSum

			urlResult.WinCount = urlSuccessCount

			if urlSuccessCount > 0 {
				urlResult.AverageDuration = urlDurationSum / int64(urlSuccessCount)
				urlResult.MinDuration = urlMinDuration
				urlResult.MaxDuration = urlMaxDuration

				urlResult.AverageSlotDelta = urlSlotDeltaSum / uint64(urlSuccessCount)
				urlResult.MinSlotDelta = urlMinSlotDelta
				urlResult.MaxSlotDelta = urlMaxSlotDelta

				if urlMinDuration < minDuration {
					minDuration = urlMinDuration
				}

				if urlMaxDuration > maxDuration {
					maxDuration = urlMaxDuration
				}

				if urlMinSlotDelta < minSlotDelta {
					minSlotDelta = urlMinSlotDelta
				}

				if urlMaxSlotDelta > maxSlotDelta {
					maxSlotDelta = urlMaxSlotDelta
				}
			}

			providerResult.URLResults = append(providerResult.URLResults, urlResult)
		}

		if totalSuccessCount > 0 {
			providerResult.AverageDuration = totalDurationSum / int64(totalSuccessCount)
			providerResult.MinDuration = minDuration
			providerResult.MaxDuration = maxDuration

			providerResult.AverageSlotDelta = totalSlotDeltaSum / uint64(totalSuccessCount)
			providerResult.MinSlotDelta = minSlotDelta
			providerResult.MaxSlotDelta = maxSlotDelta
		}

		report.ProviderResults = append(report.ProviderResults, providerResult)
	}

	return report, nil
}

// splitEndpointKey splits the combined key into URL and tipAccount
func splitEndpointKey(key string) []string {
	result := make([]string, 2)

	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			result[0] = key[:i]
			result[1] = key[i+1:]
			return result
		}
	}

	result[0] = key
	result[1] = ""
	return result
}

// SaveResults saves the test results to a file in the specified directory
func (c *Collector) SaveResults() error {
	report, err := c.ProcessResults()
	if err != nil {
		return err
	}

	outputDir := c.outputPath
	err = os.MkdirAll(outputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("speedtest-results-%s.json", timestamp)
	outputPath := filepath.Join(outputDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(outputPath, data, 0o644)
	if err != nil {
		return err
	}

	fmt.Printf("Results saved to %s\n", outputPath)
	return nil
}
