package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"vpk/file"
	"vpk/file/vpk"
)

func main() {
	out := vpk.Tree{}
	in, err := file.LocalTree("file/test")
	if err != nil {
		log.Fatal(err)
	}
	for f, e := range in.Find(".") {
		if strings.Contains(f, "local") {
			out.Store(f, e.GetData())
		}
	}
	c := 0
	for _, ext := range out {
		fmt.Println(ext.Name)
		for _, dir := range ext.Dirs {
			fmt.Println("   ", dir.Path)
			for _, e := range dir.Entries {
				c++
				fmt.Println("      ", c, e.Name, len(e.GetData()))
			}
		}
	}
	os.WriteFile("file/test/local.vpk", out.Pack(), 0660)
}
