package script

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"packman/file"
	"packman/file/vpk"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

var (
	ErrNonScript   = errors.New("not a script")
	ErrUnsupported = errors.New("unsupported")
)

var (
	patSpace = regexp.MustCompile("\\s+")
	patPack  = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
)

func errInvalidPack(p string) error {
	return fmt.Errorf("invalid binding name %s", p)
}

func errUnknownPack(p string) error {
	return fmt.Errorf("unknown binding %s", p)
}

func errIllegalArgCount(cmd string) error {
	return fmt.Errorf("illegal argument count of command '%s'", cmd)
}

func errInvalidRef(ref string) error {
	return fmt.Errorf("invalid reference '%s'", ref)
}

type Script struct {
	commands []Command
}

type Command interface {
	run(env env) error
}

type pack struct {
	tree file.Tree
	path string
	mod  bool
}

type ref struct {
	pack string
	path string
}

func (r ref) String() string {
	return fmt.Sprintf("%s:%s", r.pack, r.path)
}

func parseRef(r string) (ref, bool) {
	i := strings.IndexByte(r, ':')
	if i <= 0 {
		return ref{}, false
	}
	return ref{r[:i], file.Clean(r[i+1:])}, true
}

type env struct {
	packs map[string]*pack
	log   func(fmt string, a ...any)
}

type bind struct {
	name string
	ref
}

func (l *bind) run(env env) error {
	if l.pack == "." {
		s, err := os.Stat(l.path)
		exists := true
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				exists = false
			} else {
				return err
			}
		}
		ext := filepath.Ext(l.path)
		isVpk := strings.EqualFold(ext, ".vpk")
		if !exists && !isVpk || exists && s.IsDir() {
			loc, err := file.LocalTree(l.path)
			if err != nil {
				return err
			}
			env.packs[l.name] = &pack{loc, l.path, false}
			env.log("bound %s as a directory tree (%s)", l.name, l.path)
			return nil
		}

		var tree vpk.Tree
		if exists {
			buf, err := os.ReadFile(l.path)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			} else {
				if tree, err = vpk.Parse(buf); err != nil {
					return err
				}
			}
		}

		env.packs[l.name] = &pack{&tree, l.path, false}
		env.log("bound %s as VPK (%s)", l.name, l.path)
		return nil
	} else {
		_, ok := env.packs[l.pack]
		if !ok {
			return errUnknownPack(l.pack)
		}
		return ErrUnsupported
	}
}

type cpy struct {
	src []ref
	dst ref
}

func (c *cpy) run(env env) error {
	dst, ok := env.packs[c.dst.pack]
	if !ok {
		return errUnknownPack(c.dst.pack)
	}
	for _, s := range c.src {
		src, ok := env.packs[s.pack]
		if !ok {
			return errUnknownPack(s.pack)
		}
		for f, e := range src.tree.Find(s.path) {
			d := file.Join(c.dst.path, f)
			t, err := dst.tree.Store(d, e.GetData())
			if err != nil {
				return err
			}
			dst.mod = true
			env.log("copied %s:%s to %s:%s", s.pack, e.GetPath(), c.dst.pack, t.GetPath())
		}
	}
	return nil
}

type syntaxErr struct {
	pos int
}

type lineParser struct {
	scanner.Scanner
	buf []byte
}

func (s *lineParser) parse(no int, line string) (elem []string, err error) {
	s.Init(strings.NewReader(line))
	s.Whitespace = 0
	s.Mode = scanner.ScanIdents | scanner.ScanStrings
	s.buf = s.buf[:0]
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.Ident:
			s.buf = append(s.buf, s.TokenText()...)
		case scanner.String:
			t, err := strconv.Unquote(s.TokenText())
			if err == nil {
				s.buf = append(s.buf, t...)
			}
			return nil, fmt.Errorf("sytaxt error at %d:%d", no, s.Column)
		case ' ', '\t':
			if len(s.buf) != 0 {
				elem = append(elem, string(s.buf))
				s.buf = s.buf[:0]
			}
		default:
			s.buf = utf8.AppendRune(s.buf, tok)
		}
	}
	if len(s.buf) != 0 {
		elem = append(elem, string(s.buf))
	}
	return
}

func Parse(src []byte) (s Script, err error) {
	if !utf8.Valid(src) {
		return s, ErrNonScript
	}
	var lp lineParser
	lno := 0
	for len(src) != 0 {
		lno++
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
		elem, err := lp.parse(lno, line)
		if err != nil {
			return s, err
		}
		switch cmd := elem[0]; cmd {
		case "bind":
			if len(elem) != 3 {
				return s, errIllegalArgCount(cmd)
			}
			if !patPack.MatchString(elem[1]) {
				return s, errInvalidPack(elem[1])
			}
			p, ok := parseRef(filepath.Clean(elem[2]))
			if !ok {
				return s, errInvalidRef(elem[2])
			}
			s.commands = append(s.commands, &bind{elem[1], p})
		case "copy":
			end := len(elem) - 1
			if end < 2 {
				return s, errIllegalArgCount(cmd)
			}
			dst, ok := parseRef(elem[end])
			if !ok {
				return s, errInvalidRef(elem[end])
			}
			src := make([]ref, end-1)
			for i, e := range elem[1:end] {
				r, ok := parseRef(e)
				if !ok {
					return s, errInvalidRef(e)
				}
				src[i] = r
			}
			s.commands = append(s.commands, &cpy{src, dst})
		}
	}
	return s, nil
}

func (s Script) Run(log func(string, ...any)) {
	if log == nil {
		log = func(s string, a ...any) {
		}
	}
	env := env{make(map[string]*pack), log}
	for _, c := range s.commands {
		if err := c.run(env); err != nil {
			log(err.Error())
			return
		}
	}
	for _, p := range env.packs {
		if p.mod {
			tree, ok := p.tree.(*vpk.Tree)
			if !ok {
				continue
			}
			data := tree.Pack()
			dir, _ := filepath.Split(p.path)
			if err := os.MkdirAll(dir, 0770); err != nil {
				log(err.Error())
				return
			}
			if err := os.WriteFile(p.path, data, 0660); err != nil {
				log(err.Error())
				return
			}
		}
	}
}
