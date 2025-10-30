package constants

import "time"

// Default Server Configuration
const (
	DefaultPort = 8080
)

// Default MinIO Configuration
const (
	DefaultMinIOEndpoint  = "localhost:9000"
	DefaultMinIOBucket    = "robusta-artifacts"
	DefaultMinIOAccessKey = "minioadmin"
)

// Default Holmes Configuration
const (
	DefaultHolmesTimeoutSeconds = 300
)

// Default Rate Limiting
const (
	DefaultRateLimitRPS = 100
)

// Timeout Values
const (
	TimeoutShort      = 10 * time.Second
	TimeoutMedium     = 30 * time.Second
	TimeoutLong       = 60 * time.Second
	TimeoutVeryLong   = 2 * time.Minute
	TimeoutHTTPClient = 120 * time.Second
	TimeoutOldShort   = 15 * time.Second // Legacy timeout value
)
