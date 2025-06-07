package vpk

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"iter"
	"strings"
)

var (
	ErrNotVPK         = errors.New("not a VPK file")
	ErrUnsupportedVer = errors.New("unsupported VPK version")
	ErrUnexpectedArch = errors.New("unexpected archive MD5 section")
	ErrUnexpectedSign = errors.New("unexpected signature section")
	ErrUnexpectedPre  = errors.New("unexpected preloaded data")
	ErrInvalidDataSec = errors.New("data size mismatch")
	ErrInvalidMd5Sec  = errors.New("checksum section size mismatch")
	ErrFileCorrupted  = errors.New("file corrupted")
)

type Tree []Ext

type Ext struct {
	Name string
	Dirs []Dir
}

type Dir struct {
	Path    string
	Entries []File
}

type File struct {
	Name string
	data []byte
	crc  uint32
}

func (f *File) GetData() []byte {
	return f.data
}

func (f *File) SetData(data []byte) {
	f.crc = 0
	f.data = data
}

func Parse(vpk []byte) (Tree, error) {
	magic := binary.LittleEndian.Uint32(vpk)
	if magic != 0x55aa1234 {
		return nil, ErrNotVPK
	}
	ver := binary.LittleEndian.Uint32(vpk[4:])
	switch ver {
	case 2:
		return parse2(vpk)
	default:
		return nil, ErrUnsupportedVer
	}
}

func parse2(vpk []byte) (Tree, error) {
	if len(vpk) < 8+20+48 {
		return nil, ErrFileCorrupted
	}
	vpkSum := md5.Sum(vpk[:len(vpk)-16])
	md5Sec, vpk := vpk[len(vpk)-48:], vpk[8:]
	treeSz, vpk := binary.LittleEndian.Uint32(vpk), vpk[4:]
	dataSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	archSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if archSecSz != 0 {
		return nil, ErrUnexpectedArch
	}
	md5SecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if md5SecSz != 48 {
		return nil, ErrInvalidMd5Sec
	}
	sigSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if sigSecSz != 0 {
		return nil, ErrUnexpectedSign
	}
	tree := vpk[:treeSz]
	data := vpk[treeSz : len(vpk)-archSecSz-md5SecSz-sigSecSz]
	if len(data) != dataSecSz {
		return nil, ErrInvalidDataSec
	}
	if act, exp := md5.Sum(tree), md5Sec[:16]; !bytes.Equal(act[:], exp) {
		return nil, ErrFileCorrupted
	}
	if act, exp := md5.Sum([]byte{}), md5Sec[16:32]; !bytes.Equal(act[:], exp) {
		return nil, ErrFileCorrupted
	}
	if exp := md5Sec[32:]; !bytes.Equal(vpkSum[:], exp) {
		return nil, ErrFileCorrupted
	}
	return readDir(tree, data)
}

func readString(sec []byte) ([]byte, string) {
	i := bytes.IndexByte(sec, 0)
	if i >= 0 {
		return sec[i+1:], string(sec[:i])
	}
	return sec, ""
}

func readDir(tree []byte, data []byte) (root Tree, err error) {
	for {
		ext := Ext{}
		if tree, ext.Name = readString(tree); ext.Name == "" {
			break
		}
		for {
			dir := Dir{}
			if tree, dir.Path = readString(tree); dir.Path == "" {
				break
			}
			for {
				f := File{}
				if tree, f.Name = readString(tree); f.Name == "" {
					break
				}
				if tree, err = f.read(tree, data); err != nil {
					return nil, err
				}
				dir.Entries = append(dir.Entries, f)
			}
			ext.Dirs = append(ext.Dirs, dir)
		}
		root = append(root, ext)
	}
	return root, nil
}

func (f *File) read(tree []byte, data []byte) (rem []byte, err error) {
	rem, err = tree, ErrFileCorrupted
	if len(tree) < 18 {
		return
	}
	f.crc, tree = binary.LittleEndian.Uint32(tree), tree[4:]
	preload, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	archIdx, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	offset, tree := binary.LittleEndian.Uint32(tree), tree[4:]
	length, tree := binary.LittleEndian.Uint32(tree), tree[4:]
	term, tree := binary.LittleEndian.Uint16(tree), tree[2:]
	if term != 0xffff {
		return
	}
	if archIdx != 0x7fff {
		return
	}
	if preload != 0 {
		return rem, ErrUnexpectedPre
	}
	f.data = data[offset : offset+length]
	if f.crc != crc32.ChecksumIEEE(f.data) {
		return nil, err
	}
	return tree, nil
}

type Entry struct {
	Ext  string
	Path string
	File
}

func (e Entry) AbsPath() string {
	return fmt.Sprintf("%s/%s/%s", e.Ext, e.Path, e.Name)
}

func (tree Tree) List() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, ext := range tree {
			for _, dir := range ext.Dirs {
				for _, e := range dir.Entries {
					if !yield(Entry{ext.Name, dir.Path, e}) {
						return
					}
				}
			}
		}
	}
}

func (tree Tree) Find(path string) *Entry {
	e := strings.Split(path, "/")
	if len(e) < 3 {
		return nil
	}
	for _, ext := range tree {
		if ext.Name == e[0] {
			path = strings.Join(e[1:len(e)-1], "/")
			for _, dir := range ext.Dirs {
				if dir.Path == path {
					name := e[len(e)-1]
					for _, e := range dir.Entries {
						if e.Name == name {
							return &Entry{ext.Name, dir.Path, e}
						}
					}
				}
			}
		}
	}
	return nil
}
