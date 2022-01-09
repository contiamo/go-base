package component

// Info is the component metadata used to configure the OTEL sdks.
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}
