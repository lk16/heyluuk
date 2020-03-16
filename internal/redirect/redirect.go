package redirect

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
)

const maxPathDepth = 5

// Migrate does automatic DB model migrations
func Migrate(db *gorm.DB) error {

	if err := db.AutoMigrate(&Node{}).Error; err != nil {
		return err
	}

	err := db.Model(&Node{}).AddForeignKey(
		"parent_id",               // field
		Node{}.TableName()+"(id)", // dest
		"CASCADE",                 // onDelete
		"RESTRICT",                // onUpdate
	).Error

	return err
}

// Controller supplies some additional context for all request handlers
type Controller struct {
	DB *gorm.DB
}

func (cont *Controller) splitPath(path string) []string {
	segments := strings.Split(path, "/")

	var splitPath []string
	for _, segment := range segments {
		if segment != "" {
			splitPath = append(splitPath, segment)
		}
	}

	return splitPath
}

func (cont *Controller) getLink(segments []string) (string, error) {

	if len(segments) == 0 {
		return "", errors.New("Cannot get link for empty URL")
	}

	var node Node
	cont.DB.Where(&Node{PathSegment: segments[0]}).First(&node)

	for i := 1; i < len(segments); i++ {
		parentID := node.ID
		filter := &Node{PathSegment: segments[i], ParentID: &parentID}
		node = Node{}

		cont.DB.Where(filter).First(&node)
	}

	if node.URL == "" {
		return "", errors.New("No URL found")
	}

	return node.URL, nil
}

// Redirect redirects any url in the db
func (cont *Controller) Redirect(c echo.Context) error {

	path := c.Request().URL.Path
	splitPath := cont.splitPath(path)

	if len(splitPath) == 0 {
		return c.String(http.StatusOK, "Home page")
	}

	if len(splitPath) > maxPathDepth {
		return c.String(http.StatusBadRequest, "Path has too many segments")
	}

	url, err := cont.getLink(splitPath)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.Redirect(http.StatusFound, url)
}
