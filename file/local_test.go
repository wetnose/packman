package file

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	. "vpk/test"
)

//go:embed test/list-all.txt
var listAll []byte

//go:embed test/list-dir1.txt
var listDir1 []byte

//go:embed test/list-pref.txt
var listPref []byte

//go:embed test/list-data.txt
var listData []byte

func TestLocalFindAll(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	files1 := maps.Collect(loc.Find("."))

	list1 := slices.Collect(maps.Keys(files1))
	list2 := slices.Collect(maps.Keys(maps.Collect(loc.Find("/"))))
	slices.Sort(list1)
	slices.Sort(list2)
	Check(t, assert.Equal(t, list1, list2))
	Check(t, assert.Equal(t, string(listAll), strings.Join(list1, "\n")))

	data := slices.Collect(func(yield func(string) bool) {
		for e := range maps.Values(files1) {
			if !yield(fmt.Sprintf("%s:%s", filepath.Base(e.AbsPath()), e.GetData())) {
				return
			}
		}
	})
	slices.Sort(data)
	Check(t, assert.Equal(t, string(listData), strings.Join(data, "\n")))
}

func TestLocalListDir(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	dir1 := slices.Collect(maps.Keys(maps.Collect(loc.Find("dir1"))))
	slices.Sort(dir1)
	Check(t, assert.Equal(t, string(listDir1), strings.Join(dir1, "\n")))
}

func TestLocalListPrefix(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	dir := slices.Collect(maps.Keys(maps.Collect(loc.Find("dir"))))
	slices.Sort(dir)
	Check(t, assert.Equal(t, string(listPref), strings.Join(dir, "\n")))
}

func TestLocalListFile(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	files := maps.Collect(loc.Find("dir1/dir11/file111.txt"))
	Check(t, assert.Equal(t, 1, len(files)))
	e, ok := files["."]
	Check(t, assert.True(t, ok))
	Check(t, assert.Equal(t, "file111", string(e.GetData())))
}

func TestStore(t *testing.T) {
	Check(t, assert.NoError(t, os.RemoveAll("test/tmp")))
	loc, err := LocalTree("test/tmp")
	Check(t, assert.NoError(t, err))
	Check(t, assert.NoError(t, loc.Store("dir/f1.txt", []byte("data"))))
	data, err := os.ReadFile("test/tmp/dir/f1.txt")
	Check(t, assert.NoError(t, err))
	Check(t, assert.Equal(t, "data", string(data)))
}
