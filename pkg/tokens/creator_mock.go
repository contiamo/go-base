package tokens

// CreatorMock is a utility function to simplify writing tests that use the Creator
type CreatorMock struct {
	Err   error
	Token string
	Opts  Options
}

// Create implements tokens.Creator
func (m *CreatorMock) Create(reference string, opts Options) (string, error) {
	return m.Token, m.Err
}
