package redirect

// Node is a database model
type Node struct {
	ID          uint   `gorm:"primary_key" json:"id"`
	ParentID    *uint  `gorm:"unique_index:path_segment_parent_id" json:"parent"`
	PathSegment string `gorm:"not null;unique_index:path_segment_parent_id" json:"path_segment"`
	URL         string `gorm:"not null" json:"url"`
}

// TableName returns the name of the table associated with this model
func (Node) TableName() string {
	return "redirect_node"
}

// ErrorResponse is a JSON response model
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateLinkResponse is a JSON response model
type CreateLinkResponse struct {
	Shortcut string `json:"shortcut"`
	Redirect string `json:"redirect"`
}

// PostLinkBody is used by a JSON request model
type PostLinkBody struct {
	Recaptcha string `json:"g-recaptcha-response"`
	URL       string `json:"url"`
	Path      string `json:"path"`
}
