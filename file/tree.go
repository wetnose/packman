package file

import "iter"

type Tree interface {
	Pack() ([]byte, error)
	Get(path string) (Entry, error)
	Find(path string) iter.Seq2[string, Entry]
	Remove(path string, ln func(path string)) error
	Store(path string, data []byte) (Entry, error)
	Put(e Entry) (Entry, error)
}

type Entry interface {
	String() string
	GetPath() string
	GetData() ([]byte, error)
	GetSize() (int64, error)
}

func Store(tree Tree, e Entry) (Entry, error) {
	data, err := e.GetData()
	if err != nil {
		return nil, err
	}
	return tree.Store(e.GetPath(), data)
}
