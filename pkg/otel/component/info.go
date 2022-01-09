package component

// Info is the component metadata used to configure the OTEL sdks.
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (i Info) GetName() string {
	return i.Name
}

func (i Info) GetVersion() string {
	return i.Version
}

func (i Info) GetCommit() string {
	return i.Commit
}
