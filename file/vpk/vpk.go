package vpk

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"iter"
	"os"
	"packman/file"
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
	ErrInvalidPath    = errors.New("invalid path")
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

func (f *File) GetSize() (int64, error) {
	return int64(len(f.data)), nil
}

func (f *File) GetData() ([]byte, error) {
	return f.data, nil
}

func (f *File) SetData(data []byte) {
	f.crc = 0
	f.data = data
}

func (t *Tree) Pack() []byte {
	treeSz, dataSz := t.estimateSecSize()
	// header v2 + tree + data + checksums
	vpk := make([]byte, 28+treeSz+dataSz+48)
	hdr := vpk[:0]
	hdr = binary.LittleEndian.AppendUint32(hdr, 0x55aa1234)
	hdr = binary.LittleEndian.AppendUint32(hdr, 2) // version
	hdr = binary.LittleEndian.AppendUint32(hdr, uint32(treeSz))
	hdr = binary.LittleEndian.AppendUint32(hdr, uint32(dataSz))
	hdr = binary.LittleEndian.AppendUint32(hdr, 0)  // ArchMD5
	hdr = binary.LittleEndian.AppendUint32(hdr, 48) // Checksums
	hdr = binary.LittleEndian.AppendUint32(hdr, 0)  // Signature
	tree := vpk[28:28]
	data, off := vpk[28+treeSz:cap(vpk)-48], 0
	for _, ext := range *t {
		tree = append(tree, ext.Name...)
		tree = append(tree, 0)
		for _, dir := range ext.Dirs {
			tree = append(tree, dir.Path...)
			tree = append(tree, 0)
			for _, e := range dir.Entries {
				tree = append(tree, e.Name...)
				tree = append(tree, 0)
				if e.crc == 0 {
					e.crc = crc32.ChecksumIEEE(e.data)
				}
				tree = binary.LittleEndian.AppendUint32(tree, e.crc)
				tree = binary.LittleEndian.AppendUint16(tree, 0)      // preloaded
				tree = binary.LittleEndian.AppendUint16(tree, 0x7fff) // arch index
				tree = binary.LittleEndian.AppendUint32(tree, uint32(off))
				tree = binary.LittleEndian.AppendUint32(tree, uint32(len(e.data)))
				tree = binary.LittleEndian.AppendUint16(tree, 0xffff) // terminator
				copy(data[off:], e.data)
				off += len(e.data)
			}
			tree = append(tree, 0)
		}
		tree = append(tree, 0)
	}
	tree = append(tree, 0)
	if len(tree) != treeSz || off != len(data) {
		panic("illegal state")
	}

	sum := func(b []byte) []byte {
		s := md5.Sum(b)
		return s[:]
	}

	sums := vpk[len(vpk)-48:]
	copy(sums, sum(tree))
	copy(sums[16:], sum([]byte{}))
	copy(sums[32:], sum(vpk[:len(vpk)-16]))
	return vpk
}

func (t *Tree) estimateSecSize() (treeSz int, dataSz int) {
	treeSz++ // final ext.
	for _, ext := range *t {
		treeSz += len(ext.Name) + 1 + 1 // + end of ext.
		for _, dir := range ext.Dirs {
			treeSz += len(dir.Path) + 1 + 1 // + end of dir.
			for _, e := range dir.Entries {
				treeSz += len(e.Name) + 1
				treeSz += e.estimateEntrySize()
				dataSz += len(e.data)
			}
		}
	}
	return
}

func Read(path string) (file.Tree, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tree, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return &tree, nil
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

func (f *File) estimateEntrySize() int {
	return 18
}

type Entry struct {
	Ext  string
	Path string
	File
}

func buildName(name, ext string) string {
	if ext == " " {
		return name
	}
	return fmt.Sprintf("%s.%s", name, ext)
}

func buildPath(dir, name, ext string) string {
	if dir == "" || dir == " " {
		return buildName(name, ext)
	}
	if ext == " " {
		return fmt.Sprintf("%s/%s", dir, name)
	}
	return fmt.Sprintf("%s/%s.%s", dir, name, ext)
}

func (e Entry) GetPath() string {
	return buildPath(e.Path, e.Name, e.Ext)
}

func (t *Tree) List() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, ext := range *t {
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

func (t *Tree) FindFirst(path string) *file.Entry {
	for _, e := range t.Find(path) {
		return &e
	}
	return nil
}

func splitExt(path string) (file string, ext string) {
	if path != "" && path[len(path)-1] != '"' {
		for i := len(path) - 1; i > 0 && path[i] != '/'; i-- {
			if path[i] != '.' {
				continue
			}
			if file, ext = path[:i], path[i+1:]; ext == "" {
				break
			}
			return
		}
	}
	return path, " "
}

func cleanPath(path string) string {
	if path = file.Clean(path); path == "" {
		return ""
	}
	if path[0] == '/' {
		if path = path[1:]; path == "" {
			return ""
		}
	}
	if end := len(path) - 1; path[end] == '/' {
		if path = path[:end]; path == "" {
			return ""
		}
	}
	if path == "." {
		return ""
	}
	return path
}

func (t *Tree) Find(path string) iter.Seq2[string, file.Entry] {
	if path = cleanPath(path); path == "" {
		return func(yield func(string, file.Entry) bool) {
			for e := range t.List() {
				if !yield(e.GetPath(), &e) {
					return
				}
			}
		}
	}

	return func(yield func(string, file.Entry) bool) {
		for _, ext := range *t {
			for _, dir := range ext.Dirs {
				if dir.Path == path {
					for _, e := range dir.Entries {
						if !yield(buildName(e.Name, ext.Name), &Entry{ext.Name, dir.Path, e}) {
							return
						}
					}
					continue
				}
				if strings.HasPrefix(dir.Path, path) {
					if dir.Path[len(path)] != '/' {
						continue
					}
					root := dir.Path[len(path)+1:]
					for _, e := range dir.Entries {
						if !yield(buildPath(root, e.Name, ext.Name), &Entry{ext.Name, dir.Path, e}) {
							return
						}
					}
					continue
				}
				if strings.HasPrefix(path, dir.Path) && path[len(dir.Path)] == '/' {
					f := path[len(dir.Path)+1:]
					name, ename := splitExt(f)
					if ext.Name != ename {
						continue
					}
					for _, e := range dir.Entries {
						if e.Name == name && !yield(f, &Entry{ext.Name, dir.Path, e}) {
							return
						}
					}
				}
			}
		}
	}
}

func (t *Tree) Remove(path string) error {
	if path = cleanPath(path); path == "" {
		*t = (*t)[:0]
		return nil
	}

	u := (*t)[:0]
	for _, ext := range *t {
		dirs := ext.Dirs[:0]
		for _, dir := range ext.Dirs {
			if dir.Path == path {
				continue
			}
			if strings.HasPrefix(dir.Path, path) && dir.Path[len(path)] == '/' {
				continue
			}
			if strings.HasPrefix(path, dir.Path) && path[len(dir.Path)] == '/' {
				f := path[len(dir.Path)+1:]
				name, ename := splitExt(f)
				if ext.Name == ename {
					entries := dir.Entries[:0]
					for _, e := range dir.Entries {
						if e.Name == name {
							continue
						}
						entries = append(entries, e)
					}
					if len(entries) == 0 {
						continue
					}
					dir.Entries = entries
				}
			}
			dirs = append(dirs, dir)
		}
		if len(dirs) == 0 {
			continue
		}
		ext.Dirs = dirs
		u = append(u, ext)
	}
	*t = u
	return nil
}

func (t *Tree) Store(path string, data []byte) (file.Entry, error) {
	var name string
	if path, name = file.Split(path); name == "" {
		return nil, ErrInvalidPath
	}
	if path == "" {
		path = " "
	}
	name, ext := splitExt(name)
	entry := t.put(ext, path, name, data)
	return &entry, nil
}

func (t *Tree) put(ext, path, file string, data []byte) Entry {
	var e *Ext
	for i := range *t {
		if ex := &(*t)[i]; ex.Name == ext {
			e = ex
			break
		}
	}
	if e == nil {
		n := len(*t)
		*t = append(*t, Ext{ext, nil})
		e = &(*t)[n]
	}
	var dir *Dir
	for i := range e.Dirs {
		if d := &e.Dirs[i]; d.Path == path {
			dir = d
			break
		}
	}
	if dir == nil {
		n := len(e.Dirs)
		e.Dirs = append(e.Dirs, Dir{path, nil})
		dir = &e.Dirs[n]
	}

	for _, f := range dir.Entries {
		if f.Name == file {
			f.SetData(data)
			return Entry{ext, path, f}
		}
	}

	entry := Entry{ext, path, File{file, data, 0}}
	dir.Entries = append(dir.Entries, entry.File)
	return entry
}

func (t *Tree) Put(e file.Entry) (file.Entry, error) {
	if te, ok := e.(*Entry); ok {
		t.put(te.Ext, te.Path, te.Name, te.data)
	}
	data, err := e.GetData()
	if err != nil {
		return nil, err
	}
	return t.Store(e.GetPath(), data)
}
