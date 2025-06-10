package file

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
)

type entry struct {
	path string
	data []byte
}

func (e entry) AbsPath() string {
	return e.path
}

func (e entry) GetData() []byte {
	return e.data
}

type local string

func (l local) Pack() []byte {
	panic(errors.ErrUnsupported)
}

func (l local) Find(path string) iter.Seq2[string, Entry] {
	path = filepath.Clean(filepath.Join(string(l), path))
	return func(yield func(string, Entry) bool) {

		yieldFile := func(base, f string) bool {
			if rel, err := filepath.Rel(base, f); err == nil {
				if data, err := os.ReadFile(f); err == nil {
					return yield(filepath.ToSlash(rel), entry{filepath.ToSlash(f), data})
				}
			}
			return true
		}

		s, err := os.Stat(path)
		if err != nil {
			return
		}

		if s.IsDir() {
			for f := range list(path) {
				if !yieldFile(path, f) {
					return
				}
			}
		}

		dir, _ := filepath.Split(path)
		yieldFile(dir, path)
	}
}

func list(dir string) iter.Seq[string] {
	return func(yield func(string) bool) {
		l, _ := os.ReadDir(dir)
		for _, e := range l {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() {
				for f := range list(path) {
					if !yield(f) {
						return
					}
				}
			}
			if !yield(path) {
				return
			}
		}
	}
}

func (l local) Store(path string, data []byte) error {
	path = filepath.Join(string(l), path)
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0660)
}

func LocalTree(dir string) (Tree, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return local(dir), nil
}
