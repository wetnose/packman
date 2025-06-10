package vpk

import (
	_ "embed"
	"github.com/stretchr/testify/assert"
	"maps"
	"slices"
	"strings"
	"testing"
	. "vpk/test"
)

//go:embed test/local.vpk
var localVpk []byte

//go:embed test/list-all.txt
var listAll []byte

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
	pak, err := Parse(localVpk)
	Check(t, assert.NoError(t, err))

	files := maps.Collect(pak.Find("."))

	list := slices.Collect(maps.Keys(files))
	slices.Sort(list)

	Check(t, assert.Equal(t, string(listAll), strings.Join(list, "\n")))
}
