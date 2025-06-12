package vpk

import (
	_ "embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"maps"
	"slices"
	"strings"
	"testing"
)

//go:embed test/local.vpk
var localVpk []byte

//go:embed test/list-all.txt
var listAll []byte

//go:embed test/list-dir1.txt
var listDir1 []byte

func TestClone(t *testing.T) {
	//out := Tree{}
	//in, err := file.LocalTree("../test/local")
	//Check(t, assert.NoError(t, err))
	//for _, e := range in.Find("") {
	//	_, err := out.Put(e)
	//	Check(t, assert.NoError(t, err))
	//}
	//c := 0
	//for _, ext := range out {
	//	fmt.Println(ext.Name)
	//	for _, dir := range ext.Dirs {
	//		fmt.Println("   ", dir.Path)
	//		for _, e := range dir.Entries {
	//			c++
	//			fmt.Println("      ", c, e.Name, len(e.GetData()))
	//		}
	//	}
	//}
	//for _, e := range out.Find("") {
	//	fmt.Println(e.GetPath(), len(e.GetData()))
	//}
	//os.WriteFile("test/local.vpk", out.Pack(), 0660)
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	files := maps.Collect(tree.Find("/"))

	list := slices.Collect(maps.Keys(files))
	slices.Sort(list)

	require.Equal(t, string(listAll), strings.Join(list, "\n"))
}

func TestLocalListDir(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	dir1 := slices.Collect(maps.Keys(maps.Collect(tree.Find("dir1"))))
	slices.Sort(dir1)
	require.Equal(t, string(listDir1), strings.Join(dir1, "\n"))

	dir1c := slices.Collect(maps.Keys(maps.Collect(tree.Find("dir1/"))))
	slices.Sort(dir1c)
	require.Equal(t, dir1, dir1c)
}

func TestLocalListPrefix(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	dir := maps.Collect(tree.Find("dir"))
	assert.Equal(t, 0, len(dir))
}

func TestStore(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)
	_, err = tree.Store("dir3/f1.txt", []byte("data"))
	require.NoError(t, err)
	for e := range tree.List() {
		if e.Ext == "txt" && e.Path == "dir3" && e.Name == "f1" {
			require.Equal(t, "data", string(e.data))
			return
		}
	}
	t.Fail()
}

func TestRemove(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	require.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readAll(tree))

	require.NoError(t, tree.Remove("dir1/dir11/file111.md", nil))
	require.Equal(t, "file01 file02 file11 file12 file121 file22", readAll(tree))

	require.NoError(t, tree.Remove("dir1", nil))
	require.Equal(t, "file01 file02 file22", readAll(tree))

	require.NoError(t, tree.Remove("", nil))
	require.Equal(t, "", readAll(tree))
}

func TestRemoveWithListener(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	rem := []string{}
	ln := func() func(string) {
		rem = rem[:0]
		return func(path string) {
			rem = append(rem, path)
			slices.Sort(rem)
		}
	}

	require.Equal(t, "file01 file02 file11 file111 file12 file121 file22", readAll(tree))

	require.NoError(t, tree.Remove("dir1/dir11/file111.md", ln()))
	require.Equal(t, "file01 file02 file11 file12 file121 file22", readAll(tree))
	require.Equal(t, "dir1/dir11/file111.md", strings.Join(rem, " "))

	require.NoError(t, tree.Remove("dir1", ln()))
	require.Equal(t, "file01 file02 file22", readAll(tree))
	require.Equal(t, "dir1/dir12/file121.txt dir1/file11.txt dir1/file12.txt", strings.Join(rem, " "))

	require.NoError(t, tree.Remove("", ln()))
	require.Equal(t, "", readAll(tree))
	require.Equal(t, "dir2/file22.txt file01.txt file02.md", strings.Join(rem, " "))
}

func TestLookup(t *testing.T) {
	tree, err := Parse(localVpk)
	require.NoError(t, err)

	file111 := slices.Collect(maps.Keys(maps.Collect(tree.Find("dir1/dir11/file111.md"))))
	require.Equal(t, []string{"."}, file111)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Supplementary classes & routines                                                                               //
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func readAll(tree Tree) string {
	var data []string
	for _, e := range tree.Find("") {
		data = append(data, string(e.(*Entry).data))
	}
	slices.Sort(data)
	return strings.Join(data, " ")
}
