package file

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"maps"
	"os"
	. "packman/test"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
			if !yield(fmt.Sprintf("%s:%s", filepath.Base(e.GetPath()), e.GetData())) {
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

	dir1 := slices.Collect(maps.Keys(maps.Collect(loc.Find("dir1/"))))
	slices.Sort(dir1)
	Check(t, assert.Equal(t, string(listDir1), strings.Join(dir1, "\n")))
}

func TestLocalListPrefix(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	dir := maps.Collect(loc.Find("dir"))
	assert.Equal(t, 0, len(dir))
}

func TestLocalListFile(t *testing.T) {
	loc, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	files := maps.Collect(loc.Find("dir1/dir11/file111.txt"))
	Check(t, assert.Equal(t, 1, len(files)))
	e, ok := files["file111.txt"]
	Check(t, assert.True(t, ok))
	Check(t, assert.Equal(t, "file111", string(e.GetData())))
}

func TestStore(t *testing.T) {
	Check(t, assert.NoError(t, os.RemoveAll("test/tmp")))
	loc, err := LocalTree("test/tmp")
	Check(t, assert.NoError(t, err))
	_, err = loc.Store("dir/f1.txt", []byte("data"))
	Check(t, assert.NoError(t, err))
	data, err := os.ReadFile("test/tmp/dir/f1.txt")
	Check(t, assert.NoError(t, err))
	Check(t, assert.Equal(t, "data", string(data)))
}

func TestRemove(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	Check(t, assert.NoError(t, os.Mkdir("test/tmp", 0770)))

	src, err := LocalTree("test/local")
	Check(t, assert.NoError(t, err))

	loc, err := LocalTree("test/tmp")
	Check(t, assert.NoError(t, err))

	for _, e := range src.Find("") {
		_, err := loc.Put(e)
		Check(t, assert.NoError(t, err))
	}

	Check(t, assert.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readDir("test/tmp"), " "))

	Check(t, assert.NoError(t, loc.Remove("dir1/dir11/file111.txt")))
	Check(t, assert.Equal(t, "file01 file02 file11 file12 file121 file22", readDir("test/tmp"), " "))

	Check(t, assert.NoError(t, loc.Remove("dir1")))
	Check(t, assert.Equal(t, "file01 file02 file22", readDir("test/tmp"), " "))

	Check(t, assert.NoError(t, loc.Remove("")))
	Check(t, assert.Equal(t, "", readDir("test/tmp"), " "))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Supplementary classes & routines                                                                               //
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func readDir(path string) string {
	data := readFiles(path)
	slices.Sort(data)
	return strings.Join(data, " ")
}

func readFiles(path string) []string {
	e, _ := os.ReadDir(path)
	var data []string
	for _, f := range e {
		p := filepath.Join(path, f.Name())
		if f.IsDir() {
			data = append(data, readFiles(p)...)
		} else {
			data = append(data, readFile(p))
		}
	}
	return data
}

func readFile(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
