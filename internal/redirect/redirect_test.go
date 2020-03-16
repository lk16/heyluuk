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
)

const postgresHost = "test_db"

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

func dummyDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("host=%s sslmode=disable user=%s password=%s dbname=%s", postgresHost,
		postgresUser, postgresPassword, postgresDB)

	log.Printf("dns = %s", dsn)

	db, err := gorm.Open("postgres", dsn)
	assert.Nil(t, err)

	err = Migrate(db)
	assert.Nil(t, err)

	return db
}

func TestControllerGetLink(t *testing.T) {
	db := dummyDB(t)

	// TODO
	_ = db
}
