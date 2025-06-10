package file

import "iter"

type Tree interface {
	Pack() []byte
	Find(path string) iter.Seq2[string, Entry]
	Store(path string, data []byte) (Entry, error)
}

type Entry interface {
	GetPath() string
	GetData() []byte
}
