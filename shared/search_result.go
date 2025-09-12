package shared

// SearchItem represents a search result item for programs or sequences
type SearchItem struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

// SearchResult represents a paginated list of search items
type SearchResult struct {
	Total   int          `json:"total"`
	Results []SearchItem `json:"results"`
}
