package redirect

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
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
	// TODO
}
