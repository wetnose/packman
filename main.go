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
	for _, ext := range tree {
		fmt.Println(ext.Name)
		for _, dir := range ext.Dirs {
			fmt.Println("   ", dir.Path)
			for _, e := range dir.Entries {
				c++
				fmt.Println("      ", c, e.Name, len(e.Data))
			}
		}
	}
}
