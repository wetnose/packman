package script

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
	"vpk/dir"
	"vpk/file"
	"vpk/vpk"
)

var (
	ErrNonScript = errors.New("not a script")
)

var space = regexp.MustCompile("\\s+")

type Script struct {
	commands []Command
}

type Command interface {
	run(env Env) error
}

type Env struct {
	trees map[string]file.Tree
	log   func(fmt string, a ...any)
}

func NewEnv(log func(string, ...any)) Env {
	if log == nil {
		log = func(s string, a ...any) {
		}
	}
	return Env{make(map[string]file.Tree), log}
}

type bind struct {
	name string
	path []byte
}

type load struct {
	name string
	path string
}

type save struct {
	name string
	path string
}

func (l *load) run(env Env) error {
	buf, err := os.ReadFile(l.path)
	if err != nil {
		return err
	}
	tree, err := vpk.Parse(buf)
	if err != nil {
		return err
	}
	env.trees[l.name] = tree
	return nil
}

func (s *save) run(env Env) error {
	tree, ok := env.trees[s.path]
	if !ok {
		return fmt.Errorf("pack %s not defined", s.name)
	}
	return os.WriteFile(s.path, tree.Pack(), 0660)
}

type copy struct {
	srcTree, dstTree string
	srcPath, dstPath string
}

func (c *copy) run(env Env) error {
	srcTree, ok := env.trees[c.srcTree]
	if !ok {
		return fmt.Errorf("pack %s not defined", srcTree)
	}
	dstTree, ok := env.trees[c.dstTree]
	if !ok {
		return fmt.Errorf("pack %s not defined", dstTree)
	}

}

func Parse(src []byte) (s Script, err error) {
	if !utf8.Valid(src) {
		return s, ErrNonScript
	}
	for len(src) != 0 {
		var line string
		i := bytes.IndexByte(src, '\n')
		if i < 0 {
			line, src = string(src), src[len(src):]
		} else {
			line, src = string(src[:i]), src[i+1:]
		}
		line = strings.Trim(line, " \t\r")
		if len(line) == 0 {
			continue
		}
		elem := space.Split(line, -1)
		switch cmd := elem[0]; cmd {
		case "save", "load":
			if len(elem) != 3 {
				return s, fmt.Errorf("illegal argument count of command '%s'", cmd)
			}
			p := filepath.Clean(elem[2])
			if cmd == "load" {
				s.commands = append(s.commands, &load{elem[1], p})
			} else {
				s.commands = append(s.commands, &save{elem[1], p})
			}
		case "copy":

		}
	}
}
