package redirect

import (
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/lk16/heyluuk/internal/captcha"
)

var (
	captchaSiteKey = os.Getenv("CAPTCHA_SITE_KEY")

	errEmptyPath           = errors.New("Cannot get link for empty path")
	errEmptyRedirectURL    = errors.New("No redirect URL found")
	errLinkNotFound        = errors.New("Link not found")
	errTooManyPathSegments = errors.New("Too many path segments")
	errInvalidPath         = errors.New("Path contains invalid characters")
	errTooLongSegment      = errors.New("Path has a segment that is too long")
	errLinkPointsElsewhere = errors.New("Link exists already and redirects elsewhere")
	errLinkExists          = errors.New("Link exists already")
	errPathTooLong         = errors.New("Path is too long")
	errInvalidLink         = errors.New("Link is invalid")

	pathRegex = regexp.MustCompile("[a-z0-9/-]*")
)

const (
	maxPathDepth     = 5
	maxSegmentLength = 20
	maxPathLength    = (maxPathDepth * maxSegmentLength) + maxPathDepth
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
	DB      *gorm.DB
	Captcha captcha.CaptchaVerifier
}

func (cont *Controller) getLink(pathSegments []string) (string, error) {

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
	splitPath, err := verifyAndSplitPath(path)

	if err != nil {
		log.Printf("Error for path %s: %s", path, err.Error())
		return c.Render(http.StatusNotFound, "not_found.html", nil)
	}

	url, err := cont.getLink(splitPath)

	if err != nil {
		log.Printf("Error for path %s: %s", path, err.Error())
		return c.Render(http.StatusNotFound, "not_found.html", nil)
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

func verifyAndSplitPath(path string) (segments []string, err error) {

	if len(path) == 0 {
		return nil, errEmptyPath
	}

	if len(path) > maxPathLength {
		return nil, errPathTooLong
	}

	if !pathRegex.MatchString(path) {
		return nil, errInvalidPath
	}

	splitPath := strings.Split(path, "/")

	for _, segment := range splitPath {
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	if len(segments) == 0 {
		return nil, errEmptyPath
	}

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
	// TODO test we get HTTP 2xx and test for timeout

	if strings.Contains(URL, "heylu.uk") {
		return "", errInvalidLink
	}

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

	for i, segment := range segments {

		parentID := node.ID
		var where []interface{}

		// GORM ignores NULL values in Find(...) and similar, so we work around it
		if i == 0 {
			where = []interface{}{
				"path_segment = ? AND parent_id IS NULL", segment}
		} else {
			where = []interface{}{
				&Node{PathSegment: segment, ParentID: &parentID}}
		}

		node = Node{} // reset node to not confuse GORM
		err = cont.DB.Find(&node, where...).Error

		if gorm.IsRecordNotFoundError(err) {
			// Link not found, create it

			node = Node{PathSegment: segment}

			if i != 0 {
				node.ParentID = &parentID
			}

			if err = cont.DB.Create(&node).Error; err != nil {
				return err
			}

		} else if err != nil {
			// DB error
			return err
		}
	}

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

// PostLink handles POST requests for creating new links
func (cont *Controller) PostLink(c echo.Context) error {

	body := PostLinkBody{}

	if err := c.Bind(&body); err != nil {
		response := ErrorResponse{err.Error()}
		return c.JSON(http.StatusBadRequest, response)
	}

	res, err := cont.Captcha.Verify(body.Recaptcha)

	linkResponse := CreateLinkResponse{
		Shortcut: body.Path,
		Redirect: body.URL}

	if err != nil {
		log.Printf("Recaptcha error: %s", err.Error())
	}

	if err != nil || !res.Success {
		response := ErrorResponse{"Recaptcha verification failed"}
		return c.JSON(http.StatusBadRequest, response)
	}

	var segments []string
	if segments, err = verifyAndSplitPath(body.Path); err != nil {
		response := ErrorResponse{"Invalid shortcut"}
		return c.JSON(http.StatusBadRequest, response)
	}

	linkResponse.Shortcut = "/" + strings.Join(segments, "/")

	var URL = body.URL
	if URL, err = verifyURL(URL); err != nil {
		response := ErrorResponse{"Invalid redirect link"}
		return c.JSON(http.StatusBadRequest, response)
	}

	linkResponse.Redirect = URL

	if err = cont.insertNewLink(URL, segments); err != nil {
		response := ErrorResponse{"Saving new link failed: " + err.Error()}
		return c.JSON(http.StatusInternalServerError, response)
	}

	return c.JSON(http.StatusCreated, linkResponse)
}

// GetNode returns a node by ID
func (cont *Controller) GetNode(c echo.Context) error {

	IDString := c.Param("id")

	ID, err := strconv.Atoi(IDString)
	if err != nil {
		response := ErrorResponse{"Invalid id parameter"}
		return c.JSON(http.StatusBadRequest, response)
	}

	var node Node
	err = cont.DB.Find(&node, &Node{ID: uint(ID)}).Error

	if gorm.IsRecordNotFoundError(err) {
		return c.JSON(http.StatusNotFound, nil)
	}

	if err != nil {
		log.Printf("GetNode error: %s", err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, node)
}

// GetNodeRoot returns all root nodes
func (cont *Controller) GetNodeRoot(c echo.Context) error {

	var nodes []Node
	err := cont.DB.Find(&nodes, "parent_id IS NULL").Error

	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nodes)

}

// GetNodeChildren returns nodes whose parent is the specified ID
func (cont *Controller) GetNodeChildren(c echo.Context) error {

	IDString := c.Param("id")

	ID, err := strconv.Atoi(IDString)
	if err != nil {
		response := ErrorResponse{"Invalid id parameter"}
		return c.JSON(http.StatusBadRequest, response)
	}

	var nodes []Node
	parentID := uint(ID)
	err = cont.DB.Find(&nodes, &Node{ParentID: &parentID}).Error

	if err != nil && !gorm.IsRecordNotFoundError(err) {
		log.Printf("GetNodeChildren error: %s", err.Error())
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nodes)
}
