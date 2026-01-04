package constants

// HTTP Headers
const (
	HeaderAuthorization                 = "Authorization"
	HeaderRobustaSignature              = "X-Robusta-Signature"
	HeaderContentType                   = "Content-Type"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
)

// HTTP Content Types
const (
	MIMEApplicationJSON = "application/json"
)

// HTTP Authentication
const (
	AuthSchemeBearer = "Bearer "
)

// HTTP Status Codes
const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusNoContent           = 204
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusInternalServerError = 500
)

// HTTP Methods
const (
	AllowedHTTPMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
)
