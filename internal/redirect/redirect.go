package redirect

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
)

var (
	errNoPathSegments   = errors.New("Cannot get link for empty URL")
	errEmptyRedirectURL = errors.New("No redirect URL found")
	errLinkNotFound     = errors.New("Link not found")
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

	var node Node
	err := cont.DB.Find(&node, "parent_id IS NULL AND path_segment = ?", pathSegments[0]).Limit(1).Error

	if gorm.IsRecordNotFoundError(err) {
		return "", errLinkNotFound
	}

	for i := 1; i < len(pathSegments); i++ {
		parentID := node.ID
		filter := &Node{PathSegment: pathSegments[i], ParentID: &parentID}
		node = Node{}

		err = cont.DB.Find(&node, filter).Error

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
