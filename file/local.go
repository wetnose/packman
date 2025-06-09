package file

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
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

		yieldDir := func(base, d string) bool {
			for f := range list(d) {
				if !yieldFile(base, f) {
					return false
				}
			}
			return true
		}

		s, err := os.Stat(path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return
			}
			path, pref := filepath.Split(path)
			if s, err = os.Stat(path); err != nil || !s.IsDir() {
				return
			}
			l, _ := os.ReadDir(path)
			for _, e := range l {
				if !strings.HasPrefix(e.Name(), pref) {
					continue
				}
				f := filepath.Join(path, e.Name())
				if dir := e.IsDir(); dir && !yieldDir(path, f) || !dir && !yieldFile(path, f) {
					return
				}
			}
			return
		}

		if s.IsDir() {
			yieldDir(path, path)
			return
		}
		yieldFile(path, path)
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
	return os.WriteFile(path, data, 0660)
}

func LocalTree(dir string) (Tree[Entry], error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return local(dir), nil
}
