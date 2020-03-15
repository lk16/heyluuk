package redirect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
