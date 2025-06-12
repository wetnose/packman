package file

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	local
	path string
}

func (e entry) String() string {
	return e.path
}

func (e entry) GetPath() string {
	return e.path
}

func (e entry) GetData() ([]byte, error) {
	path := filepath.Join(string(e.local), e.path)
	return os.ReadFile(path)
}

func (e entry) GetSize() (int64, error) {
	path := filepath.Join(string(e.local), e.path)
	s, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return s.Size(), nil
}

type local string

func (l local) Pack() ([]byte, error) {
	return nil, errors.ErrUnsupported
}

func (l local) Get(path string) (Entry, error) {
	path, err := l.abs(path)
	if err != nil {
		return nil, err
	}
	s, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s.IsDir() {
		return nil, os.ErrInvalid
	}
	return &entry{l, path[len(l)+1:]}, nil
}

func (l local) Find(path string) iter.Seq2[string, Entry] {
	return func(yield func(string, Entry) bool) {
		path, err := l.abs(path)
		if err != nil {
			return
		}

		_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			rel, err := filepath.Rel(path, p)
			if err != nil {
				return err
			}
			if !yield(ToSlash(rel), entry{l, Clean(p[len(l)+1:])}) {
				return filepath.SkipAll
			}
			return nil
		})
	}
}

func (l local) abs(path string) (string, error) {
	p, err := filepath.Abs(filepath.Join(string(l), path))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(p, string(l)) || len(p) > len(l) && p[len(l)] != filepath.Separator {
		return "", fmt.Errorf("invalid file path %s", path)
	}
	return p, nil
}

func (l local) Remove(path string, ln func(path string)) (err error) {
	path, err = l.abs(path)
	if err != nil {
		return err
	}
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	return l.remove(path, s.IsDir(), ln)
}

func (l local) removeDir(path string, ln func(path string)) error {
	dir, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, e := range dir {
		path := filepath.Join(path, e.Name())
		if err := l.remove(path, e.IsDir(), ln); err != nil {
			return err
		}
	}
	return nil
}

func (l local) remove(path string, dir bool, ln func(path string)) error {
	if dir {
		if err := l.removeDir(path, ln); err != nil {
			return err
		}
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if ln != nil && !dir {
		ln(path[len(l)+1:])
	}
	return nil
}

func (l local) Store(path string, data []byte) (e Entry, err error) {
	path, err = l.abs(path)
	if err != nil {
		return nil, err
	}
	dir, _ := filepath.Split(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0770); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(path, data, 0660); err != nil {
		return nil, err
	}
	return entry{l, Clean(path[len(l)+1:])}, nil
}

func (l local) Put(e Entry) (Entry, error) {
	return Store(l, e)
}

func LocalTree(dir string) (Tree, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return local(dir), nil
}
