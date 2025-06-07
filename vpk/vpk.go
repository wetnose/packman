package vpk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

var (
	ErrNotVPK         = errors.New("not a VPK file")
	ErrUnsupportedVer = errors.New("unsupported VPK version")
	ErrUnexpectedArch = errors.New("unexpected archive section")
	ErrUnexpectedSign = errors.New("unexpected signature section")
	ErrUnexpectedPre  = errors.New("unexpected preloaded data")
	ErrInvalidDataSec = errors.New("data size mismatch")
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
	Data []byte
	crc  uint32
}

func Parse(vpk []byte) (Tree, error) {
	magic, vpk := binary.LittleEndian.Uint32(vpk), vpk[4:]
	if magic != 0x55aa1234 {
		return nil, ErrNotVPK
	}
	ver, vpk := binary.LittleEndian.Uint32(vpk), vpk[4:]
	switch ver {
	case 2:
		return parse2(vpk)
	default:
		return nil, ErrUnsupportedVer
	}
}

func parse2(vpk []byte) (Tree, error) {
	treeSz, vpk := binary.LittleEndian.Uint32(vpk), vpk[4:]
	dataSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	archSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if archSecSz != 0 {
		return nil, ErrUnexpectedArch
	}
	md5SecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	sigSecSz, vpk := int(binary.LittleEndian.Uint32(vpk)), vpk[4:]
	if sigSecSz != 0 {
		return nil, ErrUnexpectedSign
	}
	tree := vpk[:treeSz]
	data := vpk[treeSz : len(vpk)-archSecSz-md5SecSz-sigSecSz]
	if len(data) != dataSecSz {
		return nil, ErrInvalidDataSec
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
	f.Data = data[offset : offset+length]
	if f.crc != crc32.ChecksumIEEE(f.Data) {
		return nil, err
	}
	return tree, nil
}
