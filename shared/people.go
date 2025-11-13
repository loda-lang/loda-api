package shared

type Submitter struct {
	Name        string `json:"name"`
	RefId       int    `json:"-"`
	NumPrograms int    `json:"numPrograms"`
}

// SubmittersResult represents a paginated list of submitters
type SubmittersResult struct {
	Total   int         `json:"total"`
	Results []Submitter `json:"results"`
}

type Author struct {
	Name         string `json:"name"`
	NumSequences int    `json:"numSequences"`
}
