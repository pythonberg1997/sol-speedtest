package models

import "time"

// TransactionTest represents a single transaction test
type TransactionTest struct {
	ProviderName string    `json:"providerName"`
	URL          string    `json:"url"`
	TipAccount   string    `json:"tipAccount"`
	Nonce        string    `json:"nonce"`
	TxHash       string    `json:"txHash"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime,omitempty"`
	Duration     int64     `json:"duration,omitempty"` // Duration in milliseconds
	StartSlot    uint64    `json:"startSlot,omitempty"`
	SlotDelta    uint64    `json:"slotDelta,omitempty"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// TestReport represents the final report of all tests
type TestReport struct {
	StartTime          time.Time        `json:"startTime"`
	EndTime            time.Time        `json:"endTime"`
	TotalDuration      int64            `json:"totalDuration"` // Duration in milliseconds
	TotalRounds        int              `json:"totalRounds"`
	ProviderResults    []ProviderResult `json:"providerResults"`
	SuccessfulTxHashes []string         `json:"successfulTxHashes"` // All successful transaction hashes
}

// ProviderResult represents aggregated results for a provider
type ProviderResult struct {
	ProviderName     string          `json:"providerName"`
	AverageDuration  int64           `json:"averageDuration"` // Milliseconds
	MinDuration      int64           `json:"minDuration"`     // Milliseconds
	MaxDuration      int64           `json:"maxDuration"`     // Milliseconds
	AverageSlotDelta uint64          `json:"averageSlotDelta"`
	MinSlotDelta     uint64          `json:"minSlotDelta"`
	MaxSlotDelta     uint64          `json:"maxSlotDelta"`
	WinCount         int             `json:"winCount"`
	WinRate          float64         `json:"winRate"`
	URLResults       []URLTestResult `json:"urlResults"`
}

// URLTestResult represents the test results for a specific URL
type URLTestResult struct {
	URL              string            `json:"url"`
	TipAccount       string            `json:"tipAccount"`
	AverageDuration  int64             `json:"averageDuration"` // Milliseconds
	MinDuration      int64             `json:"minDuration"`     // Milliseconds
	MaxDuration      int64             `json:"maxDuration"`     // Milliseconds
	AverageSlotDelta uint64            `json:"averageSlotDelta"`
	MinSlotDelta     uint64            `json:"minSlotDelta"`
	MaxSlotDelta     uint64            `json:"maxSlotDelta"`
	WinCount         int               `json:"winCount"`
	TestDetails      []TransactionTest `json:"testDetails,omitempty"` // Detailed test results for this URL
}
