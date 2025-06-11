package file

import "iter"

type Tree interface {
	Pack() []byte
	Find(path string) iter.Seq2[string, Entry]
	Remove(path string) error
	Store(path string, data []byte) (Entry, error)
	Put(e Entry) (Entry, error)
}

type Entry interface {
	String() string
	GetPath() string
	GetData() ([]byte, error)
	GetSize() (int64, error)
}
