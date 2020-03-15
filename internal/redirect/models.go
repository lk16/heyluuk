package redirect

import (
	"github.com/jinzhu/gorm"
)

// Node is a database model
type Node struct {
	gorm.Model
	ParentID    *uint
	PathSegment string `gorm:"not null"`
	URL         string `gorm:"not null"`
}

// TableName returns the name of the table associated with this model
func (Node) TableName() string {
	return "redirect_node"
}
