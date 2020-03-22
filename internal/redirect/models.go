package redirect

// Node is a database model
type Node struct {
	ID          uint   `gorm:"primary_key"`
	ParentID    *uint  `gorm:"unique_index:path_segment_parent_id"`
	PathSegment string `gorm:"not null;unique_index:path_segment_parent_id"`
	URL         string `gorm:"not null"`
}

// TableName returns the name of the table associated with this model
func (Node) TableName() string {
	return "redirect_node"
}

// ErrorResponse is a JSON response model
type ErrorResponse struct {
	Error string
}

// CreateLinkResponse is a JSON response model
type CreateLinkResponse struct {
	Shortcut string
	Redirect string
}

// LinkTreeResponse is a JSON response model
type LinkTreeResponse struct {
	Nodes map[string]LinkTree
}

// LinkTree is used by a JSON response model
type LinkTree struct {
	Children map[string]LinkTree `json:",omitempty"`
	URL      string              `json:",omitempty"`
}

// PostLinkBody is used by a JSON request model
type PostLinkBody struct {
	Recaptcha string `json:"g-recaptcha-response"`
	URL       string `json:"url"`
	Path      string `json:"path"`
}
