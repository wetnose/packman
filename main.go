package main

import (
	"log"
	"os"
	"slices"
	"vpk/vpk"
)

func main() {
	data, err := os.ReadFile("pak01_dir.vpk")
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	tree, err := vpk.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	c := 0
	_, _ = c, tree
	//for _, ext := range tree {
	//	fmt.Println(ext.Name)
	//	for _, dir := range ext.Dirs {
	//		fmt.Println("   ", dir.Path)
	//		for _, e := range dir.Entries {
	//			c++
	//			fmt.Println("      ", c, e.Name, len(e.Data))
	//		}
	//	}
	//}
	//for e := range tree.List() {
	//	c++
	//	fmt.Printf("%d %s/%s/%s %d\n", c, e.Ext, e.Path, e.Name, len(e.Data))
	//	if c == 2000 {
	//		break
	//	}
	//}

	//if e := tree.Find("vmesh_c/models/heroes/earth_spirit/earth_spirit_arms"); e != nil {
	//	fmt.Println(e.AbsPath(), len(e.GetData()))
	//}

	tree = vpk.Tree{}

	hello := vpk.File{Name: "hello"}
	hello.SetData([]byte("Hello, World!"))

	dir := vpk.Dir{Path: "my/path"}
	dir.Entries = append(dir.Entries, hello)

	test := vpk.Ext{Name: "test"}
	test.Dirs = append(test.Dirs, dir)
	tree = slices.Insert(tree, 0, test)

	if err := os.WriteFile("test.vpk", tree.Pack(), 0660); err != nil {
		panic(err)
	}

	//p := tree.Pack()
	//fmt.Println(len(data), len(p))
	//fmt.Println(bytes.Equal(data, p))
}
