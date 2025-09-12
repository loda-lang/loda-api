package shared

type Submitter struct {
	Name        string `json:"name"`
	RefId       int    `json:"-"`
	NumPrograms int    `json:"numPrograms"`
}
