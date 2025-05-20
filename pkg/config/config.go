package config

// Config represents the application configuration
type Config struct {
	RpcUrl       string            `yaml:"rpcUrl"`
	WssUrl       string            `yaml:"wssUrl"`
	NonceAccount string            `yaml:"nonceAccount"`
	TestCount    int               `yaml:"testCount"`
	Timeout      int               `yaml:"timeout"` // Timeout in seconds
	Providers    []ProviderConfig  `yaml:"providers"`
	OutputPath   string            `yaml:"outputPath"`
	LogFilePath  string            `yaml:"logFilePath"`
	LogLevel     string            `yaml:"logLevel"`
	LogRotation  LogRotationConfig `yaml:"logRotation"`
}

// LogRotationConfig represents the configuration for log rotation
type LogRotationConfig struct {
	MaxSizeMB    int  `yaml:"maxSizeMB"`
	MaxAgeHours  int  `yaml:"maxAgeHours"`
	MaxBackups   int  `yaml:"maxBackups"`
	RotateOnTime bool `yaml:"rotateOnTime"`
}

// ProviderConfig represents a service provider configuration
type ProviderConfig struct {
	Name        string           `yaml:"name"`
	AntiMev     bool             `yaml:"antiMev"`
	PriorityFee uint64           `yaml:"priorityFee"`
	TipAmount   uint64           `yaml:"tipAmount"`
	Endpoints   []EndpointConfig `yaml:"endpoints"`
}

// EndpointConfig represents an endpoint with its URL and tip account
type EndpointConfig struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url"`
	TipAccount string `yaml:"tipAccount"`
}
