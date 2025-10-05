package shared

type Submitter struct {
	Name        string `json:"name"`
	RefId       int    `json:"-"`
	NumPrograms int    `json:"numPrograms"`
}

type Author struct {
	Name         string `json:"name"`
	NumSequences int    `json:"numSequences"`
}
