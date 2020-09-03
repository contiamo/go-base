package tokens

type CreatorMock struct {
	Err       error
	ProjectID string
	Token     string
}

func (m *CreatorMock) CreateProjectAdmin(projectID string, reference string) (string, error) {
	m.ProjectID = projectID
	return m.Token, m.Err
}

func (m *CreatorMock) Create(reference string) (string, error) {
	return m.Token, m.Err
}
