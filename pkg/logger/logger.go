package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel defines the severity level of the log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var levelNames = map[LogLevel]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// LogRotationConfig defines how log rotation should be managed
type LogRotationConfig struct {
	MaxSize      int64         // Maximum size of a log file in bytes before rotation (default: 10MB)
	MaxAge       time.Duration // Maximum age of a log file before rotation (default: 24h)
	MaxBackups   int           // Maximum number of old log files to retain (default: 5)
	RotateOnTime bool          // Whether to rotate logs based on time (daily rotation)
}

// DefaultRotationConfig provides default values for log rotation
func DefaultRotationConfig() LogRotationConfig {
	return LogRotationConfig{
		MaxSize:      10 * 1024 * 1024, // 10MB
		MaxAge:       24 * time.Hour,   // 1 day
		MaxBackups:   5,                // Keep 5 backup files
		RotateOnTime: true,             // Rotate daily
	}
}

// Logger is a structured logger with provider and endpoint context
type Logger struct {
	providerName   string
	endpointName   string
	endpointURL    string
	level          LogLevel
	logger         *log.Logger
	mu             sync.Mutex
	rotationConfig LogRotationConfig
	logDir         string
	currentFile    *os.File
	currentSize    int64
	lastRotation   time.Time
}

var (
	defaultLogger *Logger
	logMu         sync.Mutex
	logLevel      = InfoLevel
)

// InitGlobalLogger initializes the global logger with the specified log directory
func InitGlobalLogger(logDir string, level LogLevel) error {
	return InitGlobalLoggerWithRotation(logDir, level, DefaultRotationConfig())
}

// InitGlobalLoggerWithRotation initializes the global logger with specified rotation settings
func InitGlobalLoggerWithRotation(logDir string, level LogLevel, config LogRotationConfig) error {
	logMu.Lock()
	defer logMu.Unlock()

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logLevel = level

	// Create a new logger
	logger := &Logger{
		level:          level,
		rotationConfig: config,
		logDir:         logDir,
		lastRotation:   time.Now(),
	}

	// Open the initial log file
	if err := logger.rotateLog(true); err != nil {
		return fmt.Errorf("failed to create initial log file: %w", err)
	}

	defaultLogger = logger
	defaultLogger.Info("Logging initialized at level %s with rotation (maxSize=%d bytes, maxAge=%v, maxBackups=%d)",
		levelNames[level], config.MaxSize, config.MaxAge, config.MaxBackups)
	return nil
}

// cleanOldLogs removes old log files beyond the maximum number of backups
func (l *Logger) cleanOldLogs() error {
	if l.rotationConfig.MaxBackups <= 0 {
		return nil // No cleanup needed
	}

	files, err := filepath.Glob(filepath.Join(l.logDir, "speedtest-*.log"))
	if err != nil {
		return fmt.Errorf("error finding log files: %w", err)
	}

	// Sort files by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	fileInfos := make([]fileInfo, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{path: file, modTime: info.ModTime()})
	}

	// Sort by modification time, oldest first
	for i := 0; i < len(fileInfos); i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.After(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove excess files, leaving the most recent ones
	if len(fileInfos) > l.rotationConfig.MaxBackups {
		for i := 0; i < len(fileInfos)-l.rotationConfig.MaxBackups; i++ {
			err := os.Remove(fileInfos[i].path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error removing old log file %s: %v\n", fileInfos[i].path, err)
			} else {
				fmt.Printf("Removed old log file: %s\n", fileInfos[i].path)
			}
		}
	}

	return nil
}

// rotateLog closes the current log file and opens a new one
func (l *Logger) rotateLog(initial bool) error {
	// Close existing file if open
	if l.currentFile != nil {
		l.currentFile.Close()
		l.currentFile = nil
	}

	// Generate new log file name with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	logFileName := filepath.Join(l.logDir, fmt.Sprintf("speedtest-%s.log", timestamp))

	// Open new log file
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	// Update logger state
	l.currentFile = file
	l.currentSize = 0
	l.lastRotation = time.Now()
	l.logger = log.New(file, "", 0)

	if !initial {
		l.Info("Log rotated to new file: %s", logFileName)
	}

	// Clean up old log files
	if err := l.cleanOldLogs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning old log files: %v\n", err)
	}

	return nil
}

// checkRotation checks if log rotation is needed and performs rotation if necessary
func (l *Logger) checkRotation() {
	// Check size-based rotation
	if l.currentSize >= l.rotationConfig.MaxSize && l.rotationConfig.MaxSize > 0 {
		if err := l.rotateLog(false); err != nil {
			fmt.Fprintf(os.Stderr, "Error rotating log file by size: %v\n", err)
		}
		return
	}

	// Check time-based rotation
	if l.rotationConfig.RotateOnTime {
		now := time.Now()
		nextRotationTime := l.lastRotation.Add(l.rotationConfig.MaxAge)

		// Also check if day changed for daily rotation
		lastDay := l.lastRotation.YearDay()
		currentDay := now.YearDay()

		if now.After(nextRotationTime) || (currentDay != lastDay) {
			if err := l.rotateLog(false); err != nil {
				fmt.Fprintf(os.Stderr, "Error rotating log file by time: %v\n", err)
			}
		}
	}
}

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	logMu.Lock()
	defer logMu.Unlock()
	logLevel = level
	if defaultLogger != nil {
		defaultLogger.level = level
		defaultLogger.Info("Log level changed to %s", levelNames[level])
	}
}

// GetRotationConfig gets a copy of the current rotation configuration
func GetRotationConfig() LogRotationConfig {
	logMu.Lock()
	defer logMu.Unlock()
	if defaultLogger != nil {
		return defaultLogger.rotationConfig
	}
	return DefaultRotationConfig()
}

// SetRotationConfig updates the rotation configuration
func SetRotationConfig(config LogRotationConfig) {
	logMu.Lock()
	defer logMu.Unlock()
	if defaultLogger != nil {
		defaultLogger.rotationConfig = config
		defaultLogger.Info("Log rotation configuration updated (maxSize=%d bytes, maxAge=%v, maxBackups=%d)",
			config.MaxSize, config.MaxAge, config.MaxBackups)
	}
}

// GetLogger returns a new logger with the specified provider and endpoint context
func GetLogger(providerName, endpointName, endpointURL string) *Logger {
	logMu.Lock()
	defer logMu.Unlock()

	if defaultLogger == nil {
		// Create a default logger that logs to stderr if not initialized
		return &Logger{
			providerName: providerName,
			endpointName: endpointName,
			endpointURL:  endpointURL,
			level:        logLevel,
			logger:       log.New(os.Stderr, "", 0),
		}
	}

	return &Logger{
		providerName:   providerName,
		endpointName:   endpointName,
		endpointURL:    endpointURL,
		level:          logLevel,
		logger:         defaultLogger.logger,
		rotationConfig: defaultLogger.rotationConfig,
		logDir:         defaultLogger.logDir,
		currentFile:    defaultLogger.currentFile,
		currentSize:    defaultLogger.currentSize,
		lastRotation:   defaultLogger.lastRotation,
	}
}

// GetDefaultLogger returns the default global logger
func GetDefaultLogger() *Logger {
	logMu.Lock()
	defer logMu.Unlock()

	if defaultLogger == nil {
		// Create a default logger that logs to stderr if not initialized
		return &Logger{
			level:  logLevel,
			logger: log.New(os.Stderr, "", 0),
		}
	}

	return defaultLogger
}

// formatMessage formats a log message with context information
func (l *Logger) formatMessage(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := levelNames[level]

	message := fmt.Sprintf(format, args...)

	var contextStr string
	if l.providerName != "" {
		if l.endpointName != "" {
			contextStr = fmt.Sprintf("[%s][%s][%s]", l.providerName, l.endpointName, l.endpointURL)
		} else {
			contextStr = fmt.Sprintf("[%s]", l.providerName)
		}
	}

	return fmt.Sprintf("%s %s %s %s", timestamp, levelStr, contextStr, message)
}

// log outputs a log message with the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if log rotation is needed
	if l.currentFile != nil {
		l.checkRotation()
	}

	msg := l.formatMessage(level, format, args...)

	if l.logger != nil {
		l.logger.Println(msg)

		// Update current log size
		l.currentSize += int64(len(msg)) + 1 // +1 for newline
	} else {
		// Fallback to stderr if logger is not initialized
		fmt.Fprintln(os.Stderr, msg)
	}

	// Always write errors and fatal messages to stderr
	if level >= ErrorLevel {
		fmt.Fprintln(os.Stderr, msg)
	}

	if level == FatalLevel {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a fatal error message and terminates the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
}

// WithProvider returns a new logger with the specified provider context
func (l *Logger) WithProvider(providerName string) *Logger {
	return &Logger{
		providerName:   providerName,
		endpointName:   l.endpointName,
		endpointURL:    l.endpointURL,
		level:          l.level,
		logger:         l.logger,
		rotationConfig: l.rotationConfig,
		logDir:         l.logDir,
		currentFile:    l.currentFile,
		currentSize:    l.currentSize,
		lastRotation:   l.lastRotation,
	}
}

// WithEndpoint returns a new logger with the specified endpoint context
func (l *Logger) WithEndpoint(endpointURL string) *Logger {
	return &Logger{
		providerName:   l.providerName,
		endpointURL:    endpointURL,
		level:          l.level,
		logger:         l.logger,
		rotationConfig: l.rotationConfig,
		logDir:         l.logDir,
		currentFile:    l.currentFile,
		currentSize:    l.currentSize,
		lastRotation:   l.lastRotation,
	}
}
