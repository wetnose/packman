package file

import "iter"

type Tree[E Entry] interface {
	Pack() []byte
	Find(path string) iter.Seq2[string, E]
	Store(path string, data []byte) error
}

type Entry interface {
	AbsPath() string
	GetData() []byte
}
