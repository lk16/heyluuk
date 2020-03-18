package redirect

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
)

var (
	errNoPathSegments      = errors.New("Cannot get link for empty URL")
	errEmptyRedirectURL    = errors.New("No redirect URL found")
	errLinkNotFound        = errors.New("Link not found")
	errTooManyPathSegments = errors.New("Too many path segments")
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

func (cont *Controller) getLink(pathSegments []string) (string, error) {

	if len(pathSegments) == 0 {
		return "", errNoPathSegments
	}

	if len(pathSegments) > maxPathDepth {
		return "", errTooManyPathSegments
	}

	var node Node
	var err error

	for i := 0; i < len(pathSegments); i++ {

		if i == 0 {
			// GORM does not deal with NULL very well, this is a work-around
			err = cont.DB.Find(&node, "parent_id IS NULL AND path_segment = ?",
				pathSegments[0]).Limit(1).Error
		} else {
			parentID := node.ID
			node = Node{} // reset node to not confuse GORM
			filter := &Node{PathSegment: pathSegments[i], ParentID: &parentID}
			err = cont.DB.Find(&node, filter).Error
		}

		if gorm.IsRecordNotFoundError(err) {
			return "", errLinkNotFound
		}
	}

	if node.URL == "" {
		return "", errEmptyRedirectURL
	}

	return node.URL, nil
}

// Redirect redirects any url in the db
func (cont *Controller) Redirect(c echo.Context) error {

	if c.Request().Method != "GET" {
		return c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	}

	path := c.Request().URL.Path
	splitPath := cont.splitPath(path)

	url, err := cont.getLink(splitPath)

	if err != nil {
		return c.String(http.StatusNotFound, "Not Found\n")
	}

	return c.Redirect(http.StatusFound, url)
}
