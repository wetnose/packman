package main

import (
	"fmt"
	"log"
	"os"
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
	_ = c
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
	if f, ok := tree.Find("vmesh_c/models/heroes/earth_spirit/earth_spirit_arms"); ok {
		fmt.Println(f.AbsPath(), len(f.Data))
	}
}
