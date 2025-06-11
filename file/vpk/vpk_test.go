package vpk

import (
	_ "embed"
	"github.com/stretchr/testify/assert"
	"maps"
	. "packman/test"
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
	//in, err := file.LocalTree("../test")
	//Check(t, assert.NoError(t, err))
	//for f, e := range in.Find(".") {
	//	if strings.Contains(f, "local") {
	//		Check(t, assert.NoError(t, out.Store(f, e.GetData())))
	//	}
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
	//os.WriteFile("../test/local.vpk", out.Pack(), 0660)
	tree, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))

	files := maps.Collect(tree.Find("/"))

	list := slices.Collect(maps.Keys(files))
	slices.Sort(list)

	Check(t, assert.Equal(t, string(listAll), strings.Join(list, "\n")))
}

func TestLocalListDir(t *testing.T) {
	tree, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))

	dir1 := slices.Collect(maps.Keys(maps.Collect(tree.Find("local/dir1"))))
	slices.Sort(dir1)
	Check(t, assert.Equal(t, string(listDir1), strings.Join(dir1, "\n")))

	dir1c := slices.Collect(maps.Keys(maps.Collect(tree.Find("local/dir1/"))))
	slices.Sort(dir1c)
	Check(t, assert.Equal(t, dir1, dir1c))
}

func TestLocalListPrefix(t *testing.T) {
	tree, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))

	dir := maps.Collect(tree.Find("local/dir"))
	assert.Equal(t, 0, len(dir))
}

func TestStore(t *testing.T) {
	tree, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))
	_, err = tree.Store("local/dir3/f1.txt", []byte("data"))
	Check(t, assert.NoError(t, err))
	for e := range tree.List() {
		if e.Ext == "local" && e.Path == "dir3" && e.Name == "f1.txt" {
			Check(t, assert.Equal(t, "data", string(e.GetData())))
			return
		}
	}
	t.Fail()
}

func TestRemove(t *testing.T) {
	tree, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))

	Check(t, assert.Equal(t, "file11 file111 file12 file22", readAll(tree)))

	Check(t, assert.NoError(t, tree.Remove("local/dir1/dir11/file111.txt")))
	Check(t, assert.Equal(t, "file11 file12 file22", readAll(tree)))

	Check(t, assert.NoError(t, tree.Remove("local/dir1")))
	Check(t, assert.Equal(t, "file22", readAll(tree)))

	Check(t, assert.NoError(t, tree.Remove("")))
	Check(t, assert.Equal(t, "", readAll(tree)))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Supplementary classes & routines                                                                               //
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func readAll(tree Tree) string {
	var data []string
	for _, e := range tree.Find("") {
		data = append(data, string(e.GetData()))
	}
	slices.Sort(data)
	return strings.Join(data, " ")
}
