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
