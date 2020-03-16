package redirect

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
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

	db.LogMode(true)
}

func TestControllerSplitPath(t *testing.T) {
	cont := &Controller{}
	assert.Equal(t, ([]string)(nil), cont.splitPath(""))
	assert.Equal(t, ([]string)(nil), cont.splitPath("/"))
	assert.Equal(t, ([]string)(nil), cont.splitPath("//"))
	assert.Equal(t, ([]string)(nil), cont.splitPath("///"))
	assert.Equal(t, []string{"1"}, cont.splitPath("/1"))
	assert.Equal(t, []string{"1"}, cont.splitPath("/1/"))
	assert.Equal(t, []string{"1", "2"}, cont.splitPath("1/2"))
	assert.Equal(t, []string{"1", "2"}, cont.splitPath("/1/2"))
	assert.Equal(t, []string{"1", "2"}, cont.splitPath("1/2/"))
	assert.Equal(t, []string{"1", "2"}, cont.splitPath("/1/2/"))
	assert.Equal(t, []string{"1", "2"}, cont.splitPath("/1/////2/"))
}

func TestControllerGetLink(t *testing.T) {

	// clean up after this test
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
		testCase{[]string{}, "", errNoPathSegments},
		testCase{[]string{"a"}, "", errLinkNotFound},
		testCase{[]string{"foo"}, "", errEmptyRedirectURL},
		testCase{[]string{"foo", "a"}, "", errLinkNotFound},
		testCase{[]string{"foo", "bar"}, "https://example.com/", nil},
		testCase{[]string{"foo", "bar", "a'"}, "", errLinkNotFound},
	}

	cont := &Controller{DB: db}

	for _, testCase := range testCases {
		link, err := cont.getLink(testCase.segments)
		assert.Equalf(t, testCase.expectedLink, link, "segments = %+#v", testCase.segments)
		assert.Equalf(t, testCase.expectedError, err, "segments = %+#v", testCase.segments)
	}
}

func TestControllerRedirect(t *testing.T) {
	// TODO
}
