package main

import (
	"bytes"
	"encoding/binary"
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
	for _, ext := range tree {
		fmt.Println(ext.Name)
		for _, dir := range ext.Dirs {
			fmt.Println("   ", dir.Path)
			for _, e := range dir.Entries {
				fmt.Println("      ", e.Name, len(e.Data))
			}
		}
	}
}

func parse1(vpk []byte) {
	panic("not implemented yet")
}

func parse2(vpk []byte) {
	treeSz, vpk := binary.LittleEndian.Uint32(vpk), vpk[4:]
	fmt.Println("TreeSize", treeSz)
	dataSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	fmt.Println("FileDataSectionSize", dataSecSz)
	archSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if archSecSz != 0 {
		panic("unexpected archive section")
	}
	fmt.Println("ArchiveMD5SectionSize", archSecSz)
	md5SecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	fmt.Println("OtherMD5SectionSize", md5SecSz)
	sigSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	fmt.Println("SignatureSectionSize", sigSecSz)
	tree := vpk[:treeSz]
	data := vpk[treeSz : len(vpk)-archSecSz-md5SecSz-sigSecSz]
	fmt.Println("TreeSecSize", len(tree))
	fmt.Println("DataSecSize", len(data))
	readDir(tree, data)
}

func readString(sec []byte) ([]byte, string) {
	i := bytes.IndexByte(sec, 0)
	if i >= 0 {
		return sec[i+1:], string(sec[:i])
	}
	return sec, ""
}

func readDir(tree []byte, data []byte) {
	for {
		var ext string
		if tree, ext = readString(tree); ext == "" {
			break
		}
		fmt.Println(ext)
		for {
			var path string
			if tree, path = readString(tree); path == "" {
				break
			}
			fmt.Println("-", path)
			for {
				var name string
				if tree, name = readString(tree); name == "" {
					break
				}
				var e dirEntry
				tree, e = readFileInformationAndPreloadData(tree)
				content := data[e.offset : e.offset+e.length]
				_ = content
				fmt.Printf("-- %s %d\n", name, e.length)
				//if name == "_colorwarp3d0_png_b961482a" {
				//	fmt.Println(string(content))
				//}
			}
		}
	}
}

type dirEntry struct {
	crc    uint32
	offset uint32
	length uint32
	//archIdx uint16
	//preloaded []byte
}

func readFileInformationAndPreloadData(tree []byte) ([]byte, dirEntry) {
	e := dirEntry{}
	e.crc, tree = binary.LittleEndian.Uint32(tree), tree[4:]
	preload, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	archIdx, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	e.offset, tree = binary.LittleEndian.Uint32(tree), tree[4:]
	e.length, tree = binary.LittleEndian.Uint32(tree), tree[4:]
	term, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	if term != 0xffff {
		panic("file corrupted")
	}
	if archIdx != 0x7fff {
		panic("unexpected archive index")
	}
	if preload != 0 {
		panic("found file with preloaded data")
	}
	//e.preloaded, tree = tree[:preload], tree[preload:]
	return tree, e
}

// 1234 dec (decimal)
//  4D2 hex
