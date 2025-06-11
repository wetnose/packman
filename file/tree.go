package file

import "iter"

type Tree interface {
	Pack() []byte
	Find(path string) iter.Seq2[string, Entry]
	Empty(path string) error
	Store(path string, data []byte) (Entry, error)
	Put(e Entry) (Entry, error)
}

type Entry interface {
	GetPath() string
	GetData() []byte
}
