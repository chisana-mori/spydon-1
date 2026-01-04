package constants

// CORS Allowed Origins
const (
	CORSPort3000HTTP     = "http://localhost:3000"
	CORSPort5173HTTP     = "http://localhost:5173"
	CORSPort3000HTTPS    = "https://localhost:3000"
	CORSPort5173HTTPS    = "https://localhost:5173"
	CORSPort3000HTTP_127 = "http://127.0.0.1:3000"
	CORSPort5173HTTP_127 = "http://127.0.0.1:5173"
	CORSPort3000HTTP_V6  = "http://[::1]:3000"
	CORSPort5173HTTP_V6  = "http://[::1]:5173"
)

// CORS Configuration
const (
	CORSMaxAge = "86400" // 24 hours in seconds
)
