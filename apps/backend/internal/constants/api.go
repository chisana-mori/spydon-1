package constants

// API Base Paths
const (
	APIVersionV1 = "/api/v1"
)

// API Endpoint Paths
const (
	// Core API endpoints
	APIPathAnalyze           = APIVersionV1 + "/analyze"
	APIPathIngestAlert       = APIVersionV1 + "/ingest/alert"
	APIPathStreamInvestigate = "/api/stream/investigate"

	// Authentication paths
	AuthPathBase  = "/auth"
	CASLoginPath  = AuthPathBase + "/cas/login"
	CASLogoutPath = AuthPathBase + "/cas/logout"

	// Health check paths
	HealthPath = "/health"
	ReadyPath  = "/ready"
)
