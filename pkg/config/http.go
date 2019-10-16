package config

// HTTP contains all configuration parameters for HTTP
type HTTP struct {
	// Address to listen for the HTTP server
	Address string `json:"address"`
}
