package main

import (
	"log"
	"packman/script"
)

func main() {
	pm := `
bind A file/vpk/test/local.vpk
bind D tmp/1/xxx
copy A:local D:ttt
`
	s, err := script.Parse([]byte(pm))
	if err != nil {
		log.Fatal(err)
	}
	s.Run(log.Printf)

	//dst, err := file.LocalTree("tmp/1/xxx")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//src, err := os.ReadFile("tmp/pak01_dir.vpk")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//tree, err := vpk.Parse(buf)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//for e := range tree.List() {
	//	fmt.Println(e.AbsPath())
	//}
}
