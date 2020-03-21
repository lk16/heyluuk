package redirect

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"gopkg.in/romanyx/recaptcha.v1"
)

var (
	captchaSecretKey = os.Getenv("CAPTCHA_SECRET_KEY")
	captchaSiteKey   = os.Getenv("CAPTCHA_SITE_KEY")

	errEmptyPath           = errors.New("Cannot get link for empty path")
	errEmptyRedirectURL    = errors.New("No redirect URL found")
	errLinkNotFound        = errors.New("Link not found")
	errTooManyPathSegments = errors.New("Too many path segments")
	errInvalidPath         = errors.New("Path contains invalid characters")
	errTooLongSegment      = errors.New("Path has a segment that is too long")
	errLinkPointsElsewhere = errors.New("Link exists already and redirects elsewhere")
	errLinkExists          = errors.New("Link exists already")

	pathRegex = regexp.MustCompile("[a-z0-9/-]*")
)

const (
	maxPathDepth     = 5
	maxSegmentLength = 20
)

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

func splitPath(path string) []string {
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
		return "", errEmptyPath
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

		if err != nil {
			return "", err
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
	splitPath := splitPath(path)

	url, err := cont.getLink(splitPath)

	if err != nil {
		return c.String(http.StatusNotFound, "Not Found\n")
	}

	return c.Redirect(http.StatusFound, url)
}

// NewLinkGet is a page that handles GET requests to create a new link
func (cont *Controller) NewLinkGet(c echo.Context) error {

	type dataType struct {
		CaptchaSiteKey string
	}

	data := dataType{CaptchaSiteKey: captchaSiteKey}
	return c.Render(http.StatusOK, "new_link.html", data)
}

func verifyPath(path string) (segments []string, err error) {

	if len(path) == 0 {
		return nil, errEmptyPath
	}

	if !pathRegex.MatchString(path) {
		return nil, errInvalidPath
	}

	segments = splitPath(path)

	if len(segments) > maxPathDepth {
		return nil, errTooManyPathSegments
	}

	for _, segment := range segments {
		if len(segment) > maxSegmentLength {
			return nil, errTooLongSegment
		}
	}

	return segments, nil
}

func verifyURL(URL string) (string, error) {
	// TODO test we get http 200 and test for timeout

	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	return URL, nil
}

func (cont *Controller) insertNewLink(URL string, segments []string) error {

	if len(segments) == 0 {
		return errEmptyPath
	}

	var node Node
	var err error

	for i := 0; i < len(segments); i++ {

		parentID := node.ID

		if i == 0 {
			err = cont.DB.Find(&node, "parent_id IS NULL AND path_segment = ?",
				segments[0]).Limit(1).Error
		} else {
			node = Node{} // reset node to not confuse GORM
			filter := &Node{PathSegment: segments[i], ParentID: &parentID}
			err = cont.DB.Find(&node, filter).Error
		}

		if gorm.IsRecordNotFoundError(err) {
			// Link not found, create it

			node = Node{PathSegment: segments[i]}

			if parentID != 0 {
				node.ParentID = &parentID
			}

			if err = cont.DB.Create(&node).Error; err != nil {
				return err
			}

		} else if err != nil {
			// DB error
			return err
		}

		if i == len(segments)-1 {

			// Node has no link
			if node.URL == "" {
				node.URL = URL
				return cont.DB.Save(&node).Error
			}

			// Node has different link
			if node.URL != URL {
				return errLinkPointsElsewhere
			}

			// Node has same link
			return errLinkExists
		}
	}

	return nil
}

// NewLinkPost is a page that handles POST request to create a new link
func (cont *Controller) NewLinkPost(c echo.Context) error {

	r := recaptcha.New(captchaSecretKey)
	res, err := r.Verify(c.FormValue("g-recaptcha-response"))

	if err != nil {
		return err
	}

	if !res.Success {
		return c.String(http.StatusOK, "AWW")
	}

	URL := c.FormValue("url")
	path := c.FormValue("path")

	var segments []string
	if segments, err = verifyPath(path); err != nil {
		return err
	}

	if URL, err = verifyURL(URL); err != nil {
		return err
	}

	if err = cont.insertNewLink(URL, segments); err != nil {
		return err
	}

	return c.String(http.StatusOK, "YAY")
}
