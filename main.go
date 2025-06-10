package main

import (
	"fmt"
	"log"
	"os"
	"packman/file"
	"packman/file/vpk"
	"packman/script"
	"path/filepath"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "run":
			if len(os.Args) != 3 {
				break
			}
			src, err := os.ReadFile(os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
			s, err := script.Parse(src)
			if err != nil {
				log.Fatal(err)
			}
			s.Run(log.Printf)
			return
		case "list":
			if len(os.Args) != 3 {
				break
			}
			s, err := os.Stat(os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
			var tree file.Tree
			if s.IsDir() {
				tree, err = file.LocalTree(os.Args[2])
			} else {
				tree, err = vpk.Read(os.Args[2])
			}
			if err != nil {
				log.Fatal(err)
			}
			for f := range tree.Find("") {
				fmt.Println(f)
			}
			return
		}
	}

	_, exe := filepath.Split(os.Args[0])
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("   ", exe, "<command> [arguments]")
	fmt.Println()
	fmt.Println("The commands and their arguments:")
	fmt.Println()
	fmt.Println("    run  <path>     run the script")
	fmt.Println("    list <path>     read file tree")
	os.Exit(1)
}
