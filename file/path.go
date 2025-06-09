package file

import (
	"path/filepath"
	"strings"
)

func Split(path string) (dir, file string) {
	return filepath.Split(path)
}

func Split2(path string) (string, string) {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
}

func Join(elem ...string) string {
	return strings.Join(elem, "/")
}

func Clean(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func Base(path, base string) (rel string, ok bool) {
	if ok = len(path) >= len(base) && path[:len(base)] == base; ok {
		rel = path[len(base):]
		if len(rel) != 0 && rel[0] == '/' {
			rel = rel[1:]
		}
	}
	return
}

//func HasBase(path, base string) bool {
//	return strings.HasPrefix(path, base)
//}
//
//func Rel(base, target string) string {
//	r, err := filepath.Rel(base, target)
//	if err != nil {
//		panic(err)
//	}
//	return r
//}
