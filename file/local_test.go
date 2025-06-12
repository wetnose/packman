package file

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

//go:embed test/list-all.txt
var listAll []byte

//go:embed test/list-dir1.txt
var listDir1 []byte

//go:embed test/list-data.txt
var listData []byte

func TestLocalFindAll(t *testing.T) {
	loc, err := LocalTree("test/local")
	require.NoError(t, err)

	files1 := maps.Collect(loc.Find("."))

	list1 := slices.Collect(maps.Keys(files1))
	list2 := slices.Collect(maps.Keys(maps.Collect(loc.Find("/"))))
	slices.Sort(list1)
	slices.Sort(list2)
	require.Equal(t, list1, list2)
	require.Equal(t, string(listAll), strings.Join(list1, "\n"))

	data := slices.Collect(func(yield func(string) bool) {
		for e := range maps.Values(files1) {
			data, err := e.GetData()
			require.NoError(t, err)
			if !yield(fmt.Sprintf("%s:%s", filepath.Base(e.GetPath()), data)) {
				return
			}
		}
	})
	slices.Sort(data)
	require.Equal(t, string(listData), strings.Join(data, "\n"))
}

func TestLocalListDir(t *testing.T) {
	loc, err := LocalTree("test/local")
	require.NoError(t, err)

	dir1 := slices.Collect(maps.Keys(maps.Collect(loc.Find("dir1/"))))
	slices.Sort(dir1)
	require.Equal(t, string(listDir1), strings.Join(dir1, "\n"))
}

func TestLocalListPrefix(t *testing.T) {
	loc, err := LocalTree("test/local")
	require.NoError(t, err)

	dir := maps.Collect(loc.Find("dir"))
	assert.Equal(t, 0, len(dir))
}

func TestLocalListFile(t *testing.T) {
	loc, err := LocalTree("test/local")
	require.NoError(t, err)

	files := maps.Collect(loc.Find("dir1/dir11/file111.md"))
	require.Equal(t, 1, len(files))
	e, ok := files["."]
	require.True(t, ok)
	d, err := e.GetData()
	require.NoError(t, err)
	require.Equal(t, "file111", string(d))
}

func TestStore(t *testing.T) {
	require.NoError(t, os.RemoveAll("test/tmp"))
	loc, err := LocalTree("test/tmp")
	require.NoError(t, err)
	_, err = loc.Store("dir/f1.txt", []byte("data"))
	require.NoError(t, err)
	data, err := os.ReadFile("test/tmp/dir/f1.txt")
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
}

func TestRemove(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	src, err := LocalTree("test/local")
	require.NoError(t, err)

	loc, err := LocalTree("test/tmp")
	require.NoError(t, err)

	for _, e := range src.Find("") {
		_, err := loc.Put(e)
		require.NoError(t, err)
	}

	require.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readDir("test/tmp"), " ")

	require.NoError(t, loc.Remove("dir1/dir11/file111.md", nil))
	require.Equal(t, "file01 file02 file11 file12 file121 file22", readDir("test/tmp"), " ")

	require.NoError(t, loc.Remove("dir1", nil))
	require.Equal(t, "file01 file02 file22", readDir("test/tmp"), " ")

	require.NoError(t, loc.Remove("", nil))
	require.Equal(t, "", readDir("test/tmp"), " ")
}

func TestRemoveWithListener(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	src, err := LocalTree("test/local")
	require.NoError(t, err)

	loc, err := LocalTree("test/tmp")
	require.NoError(t, err)

	for _, e := range src.Find("") {
		_, err := loc.Put(e)
		require.NoError(t, err)
	}

	rem := []string{}
	ln := func() func(string) {
		rem = rem[:0]
		return func(path string) {
			rem = append(rem, path)
			slices.Sort(rem)
		}
	}

	require.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readDir("test/tmp"), " ")

	require.NoError(t, loc.Remove("dir1/dir11/file111.md", ln()))
	require.Equal(t, "file01 file02 file11 file12 file121 file22", readDir("test/tmp"), " ")
	require.Equal(t, "dir1/dir11/file111.md", strings.Join(rem, " "))

	require.NoError(t, loc.Remove("dir1", ln()))
	require.Equal(t, "file01 file02 file22", readDir("test/tmp"), " ")
	require.Equal(t, "dir1/dir12/file121.txt dir1/file11.txt dir1/file12.txt", strings.Join(rem, " "))

	require.NoError(t, loc.Remove("", ln()))
	require.Equal(t, "", readDir("test/tmp"), " ")
	require.Equal(t, "dir2/file22.txt file01.txt file02.md", strings.Join(rem, " "))
}

func TestLookup(t *testing.T) {
	loc, err := LocalTree("test/local")
	require.NoError(t, err)

	file111 := slices.Collect(maps.Keys(maps.Collect(loc.Find("dir1/dir11/file111.md"))))
	require.Equal(t, []string{"."}, file111)
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
