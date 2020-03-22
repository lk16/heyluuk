package redirect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo/v4"
	"github.com/lk16/heyluuk/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/romanyx/recaptcha.v1"
)

var (
	postgresDB       = os.Getenv("POSTGRES_TEST_DB")
	postgresUser     = os.Getenv("POSTGRES_TEST_USER")
	postgresPassword = os.Getenv("POSTGRES_TEST_PASSWORD")
	db               *gorm.DB
)

const postgresHost = "test_db"

func init() {
	dsn := fmt.Sprintf("host=%s sslmode=disable user=%s password=%s dbname=%s", postgresHost,
		postgresUser, postgresPassword, postgresDB)

	log.Printf("Connecting to dsn: %s", dsn)

	var err error
	if db, err = gorm.Open("postgres", dsn); err != nil {
		panic(err.Error())
	}

	log.Println("Running migrations")

	if err = Migrate(db); err != nil {
		panic(err.Error())
	}

	log.Println("Migrations done")

}

func TestControllerVerifyAndSplitPath(t *testing.T) {

	type testCase struct {
		path             string
		expectedSegments []string
		expectedError    error
	}

	longPathSegment := strings.Repeat("a", maxSegmentLength)
	longPath := strings.Repeat(longPathSegment+"/", maxPathDepth-1) + longPathSegment
	splitLongPath := strings.Split(longPath, "/")

	tooLongPath := strings.Repeat("a", maxPathLength+1)

	longSegment := strings.Repeat("a", maxSegmentLength)
	tooLongSegment := strings.Repeat("a", maxSegmentLength+1)

	testCases := []testCase{
		testCase{"", ([]string)(nil), errEmptyPath},
		testCase{"/", ([]string)(nil), errEmptyPath},
		testCase{"//", ([]string)(nil), errEmptyPath},
		testCase{"///", ([]string)(nil), errEmptyPath},
		testCase{"/1", []string{"1"}, nil},
		testCase{"/1/", []string{"1"}, nil},
		testCase{"1/2", []string{"1", "2"}, nil},
		testCase{"/1/2", []string{"1", "2"}, nil},
		testCase{"1/2/", []string{"1", "2"}, nil},
		testCase{"/1/2/", []string{"1", "2"}, nil},
		testCase{"/1/////2/", []string{"1", "2"}, nil},
		testCase{longPath, splitLongPath, nil},
		testCase{tooLongPath, ([]string)(nil), errPathTooLong},
		testCase{longSegment, []string{longSegment}, nil},
		testCase{tooLongSegment, ([]string)(nil), errTooLongSegment},
		testCase{"a/" + tooLongSegment, ([]string)(nil), errTooLongSegment},
	}

	for _, testCase := range testCases {
		segments, err := verifyAndSplitPath(testCase.path)
		assert.Equalf(t, segments, testCase.expectedSegments, "path=%s", testCase.path)
		assert.Equalf(t, err, testCase.expectedError, "path=%s", testCase.path)
	}
}

func TestControllerGetLink(t *testing.T) {

	// clean up after this test finishes
	defer func() {
		db.Delete(&Node{})
	}()

	fooNode := Node{PathSegment: "foo"}
	err := db.Create(&fooNode).Error
	assert.Nil(t, err)

	barNode := Node{PathSegment: "bar", ParentID: &fooNode.ID, URL: "https://example.com/"}
	err = db.Create(&barNode).Error
	assert.Nil(t, err)

	type testCase struct {
		segments      []string
		expectedLink  string
		expectedError error
	}

	testCases := []testCase{
		testCase{[]string{"a"}, "", errLinkNotFound},
		testCase{[]string{"foo"}, "", errEmptyRedirectURL},
		testCase{[]string{"foo", "a"}, "", errLinkNotFound},
		testCase{[]string{"foo", "bar"}, "https://example.com/", nil},
		testCase{[]string{"foo", "bar", "a"}, "", errLinkNotFound},
	}

	cont := &Controller{DB: db}

	for _, testCase := range testCases {
		link, err := cont.getLink(testCase.segments)
		assert.Equalf(t, testCase.expectedLink, link, "segments = %+#v", testCase.segments)
		assert.Equalf(t, testCase.expectedError, err, "segments = %+#v", testCase.segments)
	}
}

func TestControllerRedirect(t *testing.T) {

	// clean up after this test finishes
	defer func() {
		db.Delete(&Node{})
	}()

	fooNode := Node{PathSegment: "foo"}
	err := db.Create(&fooNode).Error
	assert.Nil(t, err)

	barNode := Node{PathSegment: "bar", ParentID: &fooNode.ID, URL: "https://example.com/"}
	err = db.Create(&barNode).Error
	assert.Nil(t, err)

	e := echo.New()
	cont := &Controller{DB: db}

	t.Run("postRoot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Nil(t, cont.Redirect(c))
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("getRoot", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Nil(t, cont.Redirect(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("getFooBar", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		assert.Nil(t, cont.Redirect(c))
		assert.Equal(t, http.StatusFound, rec.Code)

		location, err := rec.Result().Location()
		assert.Nil(t, err)

		assert.Equal(t, "https://example.com/", location.String())
	})
}

func TestControllerInsertNewLink(t *testing.T) {

	cont := &Controller{DB: db}

	// clean up after this test finishes
	defer func() {
		cont.DB.Delete(&Node{})
	}()

	resetDB := func() {
		cont.DB.Error = nil
		cont.DB.Delete(&Node{})

		fooNode := Node{PathSegment: "foo", URL: "https://foo/"}
		err := cont.DB.Create(&fooNode).Error
		assert.Nil(t, err)

		barNode := Node{PathSegment: "bar", ParentID: &fooNode.ID, URL: "https://bar/"}
		err = cont.DB.Create(&barNode).Error
		assert.Nil(t, err)

		bazNode := Node{PathSegment: "baz", ParentID: &barNode.ID, URL: "https://baz/"}
		err = cont.DB.Create(&bazNode).Error
		assert.Nil(t, err)
	}

	assertNoDBChanges := func(t *testing.T) {
		var count int
		err := cont.DB.Model(&Node{}).Count(&count).Error
		assert.Nil(t, err)
		assert.Equal(t, 3, count)

		var node Node
		err = cont.DB.Find(&node, &Node{PathSegment: "foo"}).Error
		assert.Nil(t, err)
		expectedNode := Node{URL: "https://foo/", PathSegment: "foo", ID: node.ID}
		assert.Equal(t, expectedNode, node)

		parentID := node.ID
		node = Node{}
		err = cont.DB.Find(&node, &Node{PathSegment: "bar", ParentID: &parentID}).Error
		assert.Nil(t, err)
		expectedNode = Node{URL: "https://bar/", PathSegment: "bar", ID: node.ID, ParentID: &parentID}
		assert.Equal(t, expectedNode, node)

		parentID = node.ID
		node = Node{}
		err = cont.DB.Find(&node, &Node{PathSegment: "baz", ParentID: &parentID}).Error
		assert.Nil(t, err)
		expectedNode = Node{URL: "https://baz/", PathSegment: "baz", ID: node.ID, ParentID: &parentID}
		assert.Equal(t, expectedNode, node)

	}

	insertedURL := "https://insertedurl/"
	errDummy := errors.New("dummy error")

	t.Run("emptyPath", func(t *testing.T) {
		resetDB()
		err := cont.insertNewLink(insertedURL, []string{})
		assert.Equal(t, errEmptyPath, err)
		assertNoDBChanges(t)
	})

	t.Run("newLinkRoot", func(t *testing.T) {

		segments := []string{"new"}

		t.Run("OK", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink(insertedURL, segments)
			assert.Nil(t, err)

			var nodes []Node
			err = cont.DB.Find(&nodes).Error
			assert.Nil(t, err)
			assert.Equal(t, 4, len(nodes))

			err = cont.DB.Find(&nodes, Node{PathSegment: "new"}).Error
			assert.Nil(t, err)
			expectedNode := Node{PathSegment: "new", ID: nodes[0].ID, URL: insertedURL}
			assert.Equal(t, expectedNode, nodes[0])
		})

		t.Run("DatabaseError", func(t *testing.T) {
			resetDB()
			cont.DB.AddError(errDummy)
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, errDummy, err)

			cont.DB.Error = nil
			assertNoDBChanges(t)
		})
	})

	t.Run("existingLinkRoot", func(t *testing.T) {

		segments := []string{"foo"}

		t.Run("SameURL", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink("https://foo/", segments)
			assert.Equal(t, err, errLinkExists)
			assertNoDBChanges(t)
		})

		t.Run("DifferentURL", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, err, errLinkPointsElsewhere)
			assertNoDBChanges(t)
		})

		t.Run("OverwriteEmpty", func(t *testing.T) {
			resetDB()
			// temporarily set the foo url to empty string
			// resetting fields to empty string (or other zero-values) is hard with normal GORM
			query := "UPDATE " + Node{}.TableName() + " SET url='' WHERE path_segment='foo'"
			_, err := cont.DB.Raw(query).Rows()
			assert.Nil(t, err)

			err = cont.insertNewLink(insertedURL, segments)
			assert.Nil(t, err)

			var count int
			err = cont.DB.Model(&Node{}).Count(&count).Error
			assert.Nil(t, err)
			assert.Equal(t, 3, count)

			var nodes []Node
			err = cont.DB.Find(&nodes, &Node{PathSegment: "foo"}).Error
			assert.Nil(t, err)
			assert.Equal(t, 1, len(nodes))
			expectedNode := Node{PathSegment: "foo", URL: insertedURL, ID: nodes[0].ID}
			assert.Equal(t, expectedNode, nodes[0])
		})

		t.Run("DatabaseError", func(t *testing.T) {
			resetDB()
			cont.DB.AddError(errDummy)
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, errDummy, err)

			cont.DB.Error = nil
			assertNoDBChanges(t)
		})
	})

	t.Run("newLinkOneDeep", func(t *testing.T) {

		segments := []string{"foo", "new"}

		t.Run("OK", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink(insertedURL, segments)
			assert.Nil(t, err)

			var nodes []Node
			err = cont.DB.Find(&nodes).Error
			assert.Nil(t, err)
			assert.Equal(t, 4, len(nodes))

			err = cont.DB.Find(&nodes, Node{PathSegment: "foo"}).Error
			assert.Nil(t, err)
			assert.Equal(t, 1, len(nodes))
			parentID := nodes[0].ID

			err = cont.DB.Find(&nodes, Node{PathSegment: "new"}).Error
			assert.Nil(t, err)
			expectedNode := Node{PathSegment: "new", ID: nodes[0].ID,
				URL: insertedURL, ParentID: &parentID}
			assert.Equal(t, expectedNode, nodes[0])
		})

		t.Run("DatabaseError", func(t *testing.T) {
			resetDB()
			cont.DB.AddError(errDummy)
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, errDummy, err)

			cont.DB.Error = nil
			assertNoDBChanges(t)
		})
	})

	t.Run("existingLinkOneDeep", func(t *testing.T) {

		segments := []string{"foo", "bar"}

		t.Run("SameURL", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink("https://bar/", segments)
			assert.Equal(t, err, errLinkExists)
			assertNoDBChanges(t)
		})

		t.Run("DifferentURL", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, err, errLinkPointsElsewhere)
			assertNoDBChanges(t)
		})

		t.Run("OverwriteEmpty", func(t *testing.T) {
			resetDB()
			// temporarily set the foo url to empty string
			// resetting fields to empty string (or other zero-values) is hard with normal GORM
			query := "UPDATE " + Node{}.TableName() + " SET url='' WHERE path_segment='bar'"
			_, err := cont.DB.Raw(query).Rows()
			assert.Nil(t, err)

			err = cont.insertNewLink(insertedURL, segments)
			assert.Nil(t, err)

			var count int
			err = cont.DB.Model(&Node{}).Count(&count).Error
			assert.Nil(t, err)
			assert.Equal(t, 3, count)

			var nodes []Node
			err = cont.DB.Find(&nodes, &Node{PathSegment: "bar"}).Error
			assert.Nil(t, err)
			assert.Equal(t, 1, len(nodes))
			expectedNode := Node{PathSegment: "bar", URL: insertedURL, ID: nodes[0].ID,
				ParentID: nodes[0].ParentID}
			assert.Equal(t, expectedNode, nodes[0])
		})

		t.Run("DatabaseError", func(t *testing.T) {
			resetDB()
			cont.DB.AddError(errDummy)
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, errDummy, err)

			cont.DB.Error = nil
			assertNoDBChanges(t)
		})
	})

	t.Run("NewLinkOneDeepWithNewParent", func(t *testing.T) {

		segments := []string{"new", "new"}

		t.Run("DatabaseError", func(t *testing.T) {
			resetDB()
			cont.DB.AddError(errDummy)
			err := cont.insertNewLink(insertedURL, segments)
			assert.Equal(t, errDummy, err)

			cont.DB.Error = nil
			assertNoDBChanges(t)
		})

		t.Run("OK", func(t *testing.T) {
			resetDB()
			err := cont.insertNewLink(insertedURL, segments)
			assert.Nil(t, err)

			var count int
			err = cont.DB.Model(&Node{}).Count(&count).Error
			assert.Nil(t, err)
			assert.Equal(t, 5, count)

			var node Node
			query := "SELECT * FROM redirect_node WHERE path_segment='new' AND parent_id IS NULL"
			err = cont.DB.Raw(query).Scan(&node).Error
			assert.Nil(t, err)
			expectedNode := Node{PathSegment: "new", ID: node.ID}
			assert.Equal(t, expectedNode, node)

			parentID := node.ID
			node = Node{}
			err = cont.DB.Find(&node, &Node{PathSegment: "new", ParentID: &parentID}).Error
			assert.Nil(t, err)
			expectedNode = Node{PathSegment: "new", ID: node.ID,
				ParentID: &parentID, URL: insertedURL}
			assert.Equal(t, expectedNode, node)
		})
	})
}

func TestControllerNewLinkPost(t *testing.T) {

	e := echo.New()

	mockCaptchaApprover := &internal.MockCaptchaVerifier{}
	mockCaptchaApprover.On("Verify", mock.Anything).Times(99).
		Return(&recaptcha.Response{Success: true}, nil)

	cont := &Controller{DB: db, Captcha: mockCaptchaApprover}

	// clean up after this test finishes
	defer func() {
		cont.DB.Delete(&Node{})
	}()

	tester := func(t *testing.T, body io.Reader,
		expectedStatusCode int, expectedJSONResponse interface{}, expectedDBNodeCount int) {

		req := httptest.NewRequest(http.MethodPost, "/api/link", body)
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := cont.PostLink(c)
		assert.Nil(t, err)
		assert.Equal(t, expectedStatusCode, rec.Code)

		expectedJSONResponseBytes, err := json.Marshal(expectedJSONResponse)
		assert.JSONEq(t, string(expectedJSONResponseBytes), rec.Body.String())

		// we can reset the error here, since the request has been done already
		cont.DB.Error = nil

		var count int
		err = cont.DB.Model(&Node{}).Count(&count).Error
		assert.Nil(t, err)
		assert.Equal(t, count, expectedDBNodeCount)
	}

	t.Run("CaptchaFail", func(t *testing.T) {

		body := PostLinkBody{Path: "", URL: ""}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		mockCaptchaFailer := &internal.MockCaptchaVerifier{}
		mockCaptchaFailer.On("Verify", mock.Anything).Once().
			Return(&recaptcha.Response{Success: false}, nil)

		cont.Captcha = mockCaptchaFailer
		defer func() {
			cont.Captcha = mockCaptchaApprover
		}()

		expectedStatusCode := http.StatusBadRequest
		expectedJSON := ErrorResponse{"Recaptcha verification failed"}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("CaptchaErrorFailer", func(t *testing.T) {

		body := PostLinkBody{Path: "", URL: ""}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		mockCaptchaFailer := &internal.MockCaptchaVerifier{}
		mockCaptchaFailer.On("Verify", mock.Anything).Once().
			Return((*recaptcha.Response)(nil), errors.New(""))

		cont.Captcha = mockCaptchaFailer
		defer func() {
			cont.Captcha = mockCaptchaApprover
		}()

		expectedStatusCode := http.StatusBadRequest
		expectedJSON := ErrorResponse{"Recaptcha verification failed"}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("InvalidShortCut", func(t *testing.T) {
		body := PostLinkBody{Path: "", URL: ""}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		expectedStatusCode := http.StatusBadRequest
		expectedJSON := ErrorResponse{"Invalid shortcut"}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("InvalidShortcut", func(t *testing.T) {
		body := PostLinkBody{Path: "", URL: "a"}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		expectedStatusCode := http.StatusBadRequest
		expectedJSON := ErrorResponse{"Invalid shortcut"}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("InvalidRedirectLink", func(t *testing.T) {
		body := PostLinkBody{Path: "a", URL: "heylu.uk"}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		expectedStatusCode := http.StatusBadRequest
		expectedJSON := ErrorResponse{"Invalid redirect link"}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("DBError", func(t *testing.T) {
		body := PostLinkBody{Path: "a", URL: "a"}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		errDummy := errors.New("dummy error")
		cont.DB.AddError(errDummy)
		defer func() {
			cont.DB.Error = nil
		}()

		expectedStatusCode := http.StatusInternalServerError
		expectedJSON := ErrorResponse{"Saving new link failed: " + errDummy.Error()}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 0)
	})

	t.Run("OK", func(t *testing.T) {
		body := PostLinkBody{Path: "a", URL: "b"}
		bodyBytes, err := json.Marshal(body)
		assert.Nil(t, err)

		expectedStatusCode := http.StatusCreated
		expectedJSON := CreateLinkResponse{Shortcut: "/" + body.Path, Redirect: "http://" + body.URL}
		tester(t, bytes.NewBuffer(bodyBytes), expectedStatusCode, expectedJSON, 1)
	})
}

func TestControllerGetNode(t *testing.T) {

	cont := &Controller{DB: db}
	e := echo.New()

	// clean up after this test finishes
	defer func() {
		cont.DB.Delete(&Node{})
	}()

	fooNode := Node{PathSegment: "foo", URL: "http://foo/"}
	err := cont.DB.Create(&fooNode).Error
	assert.Nil(t, err)

	barNode := Node{PathSegment: "bar", URL: "http://bar/", ParentID: &fooNode.ID}
	err = cont.DB.Create(&barNode).Error
	assert.Nil(t, err)

	expectedDBNodeCount := 2

	tester := func(t *testing.T, ID string, expectedStatusCode int, expectedJSONResponse interface{}) {

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/node/:id")
		c.SetParamNames("id")
		c.SetParamValues(ID)

		err := cont.GetNode(c)
		assert.Nil(t, err)
		assert.Equal(t, expectedStatusCode, rec.Code)

		expectedJSONResponseBytes, err := json.Marshal(expectedJSONResponse)
		assert.JSONEq(t, string(expectedJSONResponseBytes), rec.Body.String())

		// we can reset the error here, since the request has been done already
		cont.DB.Error = nil

		var count int
		err = cont.DB.Model(&Node{}).Count(&count).Error
		assert.Nil(t, err)
		assert.Equal(t, count, expectedDBNodeCount)
	}

	t.Run("InvalidParameter", func(t *testing.T) {
		expectedJSONResponse := ErrorResponse{"Invalid id parameter"}
		tester(t, "broken", http.StatusBadRequest, expectedJSONResponse)
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := fmt.Sprintf("%d", fooNode.ID+barNode.ID)
		tester(t, nonExistentID, http.StatusNotFound, nil)
	})

	t.Run("DBError", func(t *testing.T) {

		cont.DB.Error = errors.New("")
		defer func() {
			cont.DB.Error = nil
		}()

		ID := fmt.Sprintf("%d", fooNode.ID)
		tester(t, ID, http.StatusInternalServerError, nil)
	})

	t.Run("Found", func(t *testing.T) {
		ID := fmt.Sprintf("%d", barNode.ID)
		tester(t, ID, http.StatusOK, barNode)
	})

	t.Run("FoundWithParent", func(t *testing.T) {
		ID := fmt.Sprintf("%d", fooNode.ID)
		tester(t, ID, http.StatusOK, fooNode)
	})
}

func TestControllerNodeRoot(t *testing.T) {
	cont := &Controller{DB: db}
	e := echo.New()

	// clean up after this test finishes
	defer func() {
		cont.DB.Delete(&Node{})
	}()

	tester := func(t *testing.T, expectedStatusCode int, expectedJSONResponse interface{}) {

		req := httptest.NewRequest(http.MethodGet, "/api/node/root", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := cont.GetNodeRoot(c)
		assert.Nil(t, err)
		assert.Equal(t, expectedStatusCode, rec.Code)

		expectedJSONResponseBytes, err := json.Marshal(expectedJSONResponse)
		assert.JSONEq(t, string(expectedJSONResponseBytes), rec.Body.String())
	}

	t.Run("NoItems", func(t *testing.T) {
		tester(t, http.StatusOK, []Node{})
	})

	t.Run("DBError", func(t *testing.T) {
		cont.DB.AddError(errors.New(""))
		defer func() {
			cont.DB.Error = nil
		}()

		tester(t, http.StatusInternalServerError, nil)
	})

	fooNode := Node{PathSegment: "foo"}
	err := cont.DB.Create(&fooNode).Error
	assert.Nil(t, err)

	barNode := Node{PathSegment: "bar", URL: "http://bar/", ParentID: &fooNode.ID}
	err = cont.DB.Create(&barNode).Error
	assert.Nil(t, err)

	bazNode := Node{PathSegment: "baz"}
	err = cont.DB.Create(&bazNode).Error
	assert.Nil(t, err)

	t.Run("ItemsFound", func(t *testing.T) {
		tester(t, http.StatusOK, []Node{fooNode, bazNode})
	})

}
