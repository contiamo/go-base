package config

// Authorization contains all the authorization-related parameters
type Authorization struct {
	// HeaderName is the name of the header where the authorization middleware is supposed
	// to be looking for a JWT token
	HeaderName string `json:"headerName"`
}
