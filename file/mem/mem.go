package mem

import (
	"errors"
	"iter"
	"os"
	"packman/file"
	"strings"
)

type entry struct {
	path string
	data []byte
}

func (e *entry) String() string {
	return e.path
}

func (e *entry) GetPath() string {
	return e.path
}

func (e *entry) GetData() ([]byte, error) {
	return e.data, nil
}

func (e *entry) GetSize() (int64, error) {
	return int64(len(e.data)), nil
}

type Store map[string]*entry

func (s *Store) Pack() ([]byte, error) {
	return nil, errors.ErrUnsupported
}

func cleanPath(path string) string {
	if path = file.Clean(path); path == "/" || path == "" || path == "." {
		return ""
	}
	return path
}

func (s *Store) Get(path string) (file.Entry, error) {
	if path = cleanPath(path); path == "" {
		return nil, os.ErrInvalid
	}
	if e, ok := (*s)[path]; ok {
		return e, nil
	}
	return nil, os.ErrNotExist
}

func (s *Store) Find(path string) iter.Seq2[string, file.Entry] {
	if path = cleanPath(path); path == "" {
		return func(yield func(string, file.Entry) bool) {
			for p, e := range *s {
				if !yield(p, e) {
					return
				}
			}
		}
	}
	return func(yield func(string, file.Entry) bool) {
		if e, ok := (*s)[path]; ok {
			yield(".", e)
			return
		}

		for p, e := range *s {
			if strings.HasPrefix(p, path) && p[len(path)] == '/' {
				if !yield(p[len(path)+1:], e) {
					return
				}
			}
		}
	}
}

func (s *Store) Remove(path string, ln func(path string)) error {
	if path = cleanPath(path); path == "" {
		if ln != nil {
			for p := range *s {
				delete(*s, p)
				ln(p)
			}
		} else {
			clear(*s)
		}
		return nil
	}
	if _, ok := (*s)[path]; ok {
		delete(*s, path)
		if ln != nil {
			ln(path)
		}
		return nil
	}
	for p := range *s {
		if strings.HasPrefix(p, path) && p[len(path)] == '/' {
			delete(*s, p)
			if ln != nil {
				ln(p)
			}
		}
	}
	return nil
}

func (s *Store) Store(path string, data []byte) (file.Entry, error) {
	if path = cleanPath(path); path == "" {
		return nil, os.ErrInvalid
	}
	return s.store(path, data), nil
}

func (s *Store) store(path string, data []byte) *entry {
	e := &entry{path, data}
	(*s)[path] = e
	return e
}

func (s *Store) Put(e file.Entry) (file.Entry, error) {
	if t, ok := e.(*entry); ok {
		return s.Store(t.path, t.data)
	}
	data, err := e.GetData()
	if err != nil {
		return nil, err
	}
	return s.Store(e.GetPath(), data)
}
