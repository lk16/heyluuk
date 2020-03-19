package redirect

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"gopkg.in/romanyx/recaptcha.v1"
)

var (
	captchaSecretKey = os.Getenv("CAPTCHA_SECRET_KEY")
	captchaSiteKey   = os.Getenv("CAPTCHA_SITE_KEY")

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

// NewLinkGet is a page that handles GET requests to create a new link
func (cont *Controller) NewLinkGet(c echo.Context) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
		<html lang="en">
		<head>
		<meta charset="UTF-8">
		<title>Golang reCAPTCHA Signup Form</title>
		<script src="https://www.google.com/recaptcha/api.js?render=%s"></script>
		<script>
		grecaptcha.ready(function() {
			grecaptcha.execute('%s', {action: 'homepage'}).then(function(token) {
				document.getElementById('g-recaptcha-response').value = token;
			});
		});
		</script>
		</head>
		<body>
		<h1>Luuk at this new link!</h1>
		<form method="POST" action="/at/this">
		<input type="hidden" id="g-recaptcha-response" name="g-recaptcha-response" />
		https://heylu.uk/<input type="text" name="shortcut">
		<br>
		URL: <input type="text" name="url">
		<br>
		<br>
		<input type="submit" value="Submit">
		</form>
		</body>
		</html>`, captchaSiteKey, captchaSiteKey)

	return c.HTML(http.StatusOK, html)
}

// NewLinkPost is a page that handles POST request to create a new link
func (cont *Controller) NewLinkPost(c echo.Context) error {

	r := recaptcha.New(captchaSecretKey)
	_, err := r.Verify(c.FormValue("g-recaptcha-response"))

	if err != nil {
		return c.String(http.StatusOK, "AWW")
	}

	// TODO actually handle POST data

	return c.String(http.StatusOK, "YAY")
}
