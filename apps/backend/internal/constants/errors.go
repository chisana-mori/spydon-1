package constants

// Error Codes
const (
	ErrorCodeInternal         = "INTERNAL_ERROR"
	ErrorCodeBadRequest       = "BAD_REQUEST"
	ErrorCodeValidationFailed = "VALIDATION_FAILED"
	ErrorCodeUnauthorized     = "UNAUTHORIZED"
	ErrorCodeForbidden        = "FORBIDDEN"
	ErrorCodeNotFound         = "NOT_FOUND"
	ErrorCodeConflict         = "CONFLICT"
	// CRUD operation errors
	ErrorCodeCreateFailed = "CREATE_FAILED"
	ErrorCodeUpdateFailed = "UPDATE_FAILED"
	ErrorCodeDeleteFailed = "DELETE_FAILED"
	ErrorCodeListFailed   = "LIST_FAILED"
	ErrorCodeGetFailed    = "GET_FAILED"

	// Authentication errors
	ErrorCodeGenerateStateFailed = "GENERATE_STATE_FAILED"
	ErrorCodeAuthFailed          = "AUTH_FAILED"

	// Security errors
	ErrorCodeHMACVerificationFailed = "HMAC_VERIFICATION_FAILED"
)
