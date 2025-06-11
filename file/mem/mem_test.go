package mem

import (
	_ "embed"
	"github.com/stretchr/testify/require"
	"maps"
	"slices"
	"strings"
	"testing"
)

//go:embed test/list-all.txt
var listAll []byte

//go:embed test/list-dir1.txt
var listDir1 []byte

func TestFind(t *testing.T) {
	s := prepareStore()
	require.Equal(t, string(listAll), find(s, ""))
	require.Equal(t, string(listDir1), find(s, "dir1"))
	require.Equal(t, string(listDir1), find(s, "dir1/"))
	require.Equal(t, ".", find(s, "dir1/dir11/file111.md"))
}

func TestPrefix(t *testing.T) {
	s := prepareStore()
	require.Equal(t, "", find(s, "dir"))
}

func TestRemove(t *testing.T) {
	s := prepareStore()
	require.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readAll(s))

	require.NoError(t, s.Remove("dir1/dir11/file111.md"))
	require.Equal(t, "file01 file02 file11 file12 file121 file22", readAll(s))

	require.NoError(t, s.Remove("dir1"))
	require.Equal(t, "file01 file02 file22", readAll(s))

	require.NoError(t, s.Remove(""))
	require.Equal(t, "", readAll(s))
}

func TestStore(t *testing.T) {
	s := make(Store)

	var err error

	_, err = s.Store("file01.txt", []byte("file01"))
	require.NoError(t, err)

	_, err = s.Store("dir1/file11.txt", []byte("file11"))
	require.NoError(t, err)

	_, err = s.Store("dir1/dir11/file111.md", []byte("file111"))
	require.NoError(t, err)

	require.Equal(t, "dir1/dir11/file111.md\ndir1/file11.txt\nfile01.txt", find(s, ""))
}

func TestPut(t *testing.T) {
	s := prepareStore()
	d := make(Store)

	for _, e := range s {
		_, err := d.Put(e)
		require.NoError(t, err)
	}

	require.Equal(t, s, d)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Supplementary classes & routines                                                                               //
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func prepareStore() Store {
	s := make(Store)
	s.store("file01.txt", []byte("file01"))
	s.store("file02.md", []byte("file02"))
	s.store("dir1/dir11/file111.md", []byte("file111"))
	s.store("dir1/dir12/file121.txt", []byte("file121"))
	s.store("dir1/file11.txt", []byte("file11"))
	s.store("dir1/file12.txt", []byte("file12"))
	s.store("dir2/file22.txt", []byte("file22"))
	return s
}

func find(s Store, path string) string {
	files := slices.Collect(maps.Keys(maps.Collect(s.Find(path))))
	slices.Sort(files)
	return strings.Join(files, "\n")
}

func readAll(s Store) string {
	data := slices.Collect(func(yield func(string) bool) {
		for _, e := range s {
			if !yield(string(e.data)) {
				return
			}
		}
	})
	slices.Sort(data)
	return strings.Join(data, " ")
}
