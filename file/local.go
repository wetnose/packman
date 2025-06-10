package file

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	path string
	data []byte
}

func (e entry) GetPath() string {
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
	return func(yield func(string, Entry) bool) {
		path, err := filepath.Abs(filepath.Join(string(l), path))
		if err != nil {
			return
		}

		yieldFile := func(base, f string) bool {
			if rel, err := filepath.Rel(base, f); err == nil {
				if data, err := os.ReadFile(f); err == nil {
					return yield(ToSlash(rel), entry{Clean(f[len(l)+1:]), data})
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

func (l local) Store(path string, data []byte) (Entry, error) {
	p, err := filepath.Abs(filepath.Join(string(l), path))
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(p, string(l)) || p[len(l)] != filepath.Separator {
		return nil, fmt.Errorf("invalid file path %s", path)
	}
	path = p
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0660); err != nil {
		return nil, err
	}
	return entry{Clean(path[len(l)+1:]), data}, nil
}

func LocalTree(dir string) (Tree, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return local(dir), nil
}
