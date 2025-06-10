package file

import "iter"

type Tree interface {
	Pack() []byte
	Find(path string) iter.Seq2[string, Entry]
	Store(path string, data []byte) error
}

type Entry interface {
	AbsPath() string
	GetData() []byte
}
