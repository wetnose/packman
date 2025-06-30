package script

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"packman/file"
	"packman/file/mem"
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
	patPack = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
)

func errInvalidPack(lno int, p string) error {
	return fmt.Errorf("invalid binding name %s at line %d", p, lno)
}

func errUnknownPack(p string) error {
	return fmt.Errorf("unknown binding %s", p)
}

func errIllegalArgCount(lno int, cmd string) error {
	return fmt.Errorf("illegal argument count of command '%s' at line %d", cmd, lno)
}

func errInvalidRef(lno int, ref string) error {
	return fmt.Errorf("invalid reference '%s' at line %d", ref, lno)
}

func errUnknownFlag(lno int, flag string) error {
	return fmt.Errorf("unknown flag '%s' at line %d", flag, lno)
}

type Script struct {
	commands []Command
}

type Command interface {
	run(env env) error
	String() string
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

var noref ref

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

func (l *bind) String() string {
	if l.ref == noref {
		return fmt.Sprintf("bind %s", l.name)
	}
	return fmt.Sprintf("bind %s %s", l.name, l.ref)
}

func (l *bind) run(env env) error {
	if l.ref == noref {
		s := make(mem.Store)
		env.packs[l.name] = &pack{tree: &s}
		return nil
	}
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

func (c *cpy) String() string {
	var buf []byte
	buf = append(buf, "copy"...)
	for _, s := range c.src {
		buf = append(buf, ' ')
		buf = append(buf, s.String()...)
	}
	buf = append(buf, ' ')
	buf = append(buf, c.dst.String()...)
	return string(buf)
}

func (c *cpy) run(env env) error {
	dst, ok := env.packs[c.dst.pack]
	if !ok {
		return errUnknownPack(c.dst.pack)
	}

	store := func(path string, data []byte) error {
		if _, err := dst.tree.Store(path, data); err != nil {
			return err
		}
		dst.mod = true
		return nil
	}

	first := len(c.src) == 1
	for _, s := range c.src {
		src, ok := env.packs[s.pack]
		if !ok {
			return errUnknownPack(s.pack)
		}
		for f, e := range src.tree.Find(s.path) {
			buf, err := e.GetData()
			if err != nil {
				return err
			}
			if first {
				first = false
				if e.GetPath() == s.path {
					if p := c.dst.path; p != "" && p[len(p)-1] != '/' {
						return store(p, buf)
					}
				}
			}
			if err = store(file.Join(c.dst.path, f), buf); err != nil {
				return err
			}
		}
	}
	return nil
}

const (
	fRegex = 1
)

type clone struct {
	src   []ref
	dst   string
	flags int
}

func (c *clone) String() string {
	var buf []byte
	buf = append(buf, "clone"...)
	if c.flags&fRegex != 0 {
		buf = append(buf, " -e"...)
	}
	for _, s := range c.src {
		buf = append(buf, ' ')
		buf = append(buf, s.String()...)
	}
	buf = append(buf, ' ')
	buf = append(buf, c.dst...)
	buf = append(buf, ':')
	return string(buf)
}

func (c *clone) run(env env) error {
	dst, ok := env.packs[c.dst]
	if !ok {
		return errUnknownPack(c.dst)
	}
	for _, s := range c.src {
		src, ok := env.packs[s.pack]
		if !ok {
			return errUnknownPack(s.pack)
		}
		if c.flags&fRegex != 0 {
			r, err := regexp.Compile(s.path)
			if err != nil {
				return err
			}
			for _, e := range src.tree.Find(".") {
				if r.FindString(e.GetPath()) == "" {
					continue
				}
				if _, err := dst.tree.Put(e); err != nil {
					return err
				}
				dst.mod = true
			}
		} else {
			for _, e := range src.tree.Find(s.path) {
				if _, err := dst.tree.Put(e); err != nil {
					return err
				}
				dst.mod = true
			}
		}
	}
	return nil
}

type remove ref

func (e *remove) String() string {
	return fmt.Sprintf("remove %s", *e)
}

func (e *remove) run(env env) error {
	dst, ok := env.packs[e.pack]
	if !ok {
		return errUnknownPack(e.pack)
	}
	dst.mod = true
	return dst.tree.Remove(e.path, nil)
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
			if err != nil {
				return nil, fmt.Errorf("sytaxt error at %d:%d", no, s.Column)
			}
			s.buf = append(s.buf, t...)
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
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		elem, err := lp.parse(lno, line)
		if err != nil {
			return s, err
		}
		switch cmd, args := elem[0], elem[1:]; cmd {
		case "bind":
			c := len(args)
			if c != 1 && c != 2 {
				return s, errIllegalArgCount(lno, cmd)
			}
			if !patPack.MatchString(args[0]) {
				return s, errInvalidPack(lno, args[0])
			}
			if c == 1 {
				s.commands = append(s.commands, &bind{args[0], ref{}})
			} else {
				p, ok := parseRef(filepath.Clean(args[1]))
				if !ok {
					return s, errInvalidRef(lno, args[1])
				}
				s.commands = append(s.commands, &bind{args[0], p})
			}
		case "remove":
			if len(args) != 1 {
				return s, errIllegalArgCount(lno, cmd)
			}
			p, ok := parseRef(filepath.Clean(args[0]))
			if !ok {
				return s, errInvalidRef(lno, args[0])
			}
			s.commands = append(s.commands, (*remove)(&p))
		case "copy", "clone":
			end := len(args) - 1
			if end < 1 {
				return s, errIllegalArgCount(lno, cmd)
			}
			flags := 0
			if args[0] == "-e" {
				if cmd != "clone" {
					return s, errUnknownFlag(lno, args[0])
				}
				flags |= fRegex
				args, end = args[1:], end-1
			}
			if end == 0 {
				return s, errIllegalArgCount(lno, cmd)
			}
			dst, ok := parseRef(args[end])
			if !ok || cmd == "clone" && dst.path != "" && dst.path != "." {
				return s, errInvalidRef(lno, args[end])
			}
			src := make([]ref, end)
			for i, e := range args[:end] {
				r, ok := parseRef(e)
				if !ok {
					return s, errInvalidRef(lno, e)
				}
				src[i] = r
			}
			var c Command
			if cmd == "clone" {
				c = &clone{src, dst.pack, flags}
			} else {
				c = &cpy{src, dst}
			}
			s.commands = append(s.commands, c)
		default:
			return s, fmt.Errorf("unknown command %s at line %d", cmd, lno)
		}
	}
	return s, nil
}

func (s Script) Run(log func(string, ...any)) error {
	if log == nil {
		log = func(s string, a ...any) {
		}
	}
	env := env{make(map[string]*pack), log}
	for _, c := range s.commands {
		log("%s", c)
		if err := c.run(env); err != nil {
			return err
		}
	}
	for _, p := range env.packs {
		if p.mod {
			tree, ok := p.tree.(*vpk.Tree)
			if !ok {
				continue
			}
			if len(*tree) == 0 {
				if err := os.Remove(p.path); err != nil {
					return err
				}
			}
			data, err := tree.Pack()
			if err != nil {
				return err
			}
			dir, _ := filepath.Split(p.path)
			if dir != "" {
				if err := os.MkdirAll(dir, 0770); err != nil {
					return err
				}
			}
			if err := os.WriteFile(p.path, data, 0660); err != nil {
				return err
			}
		}
	}
	return nil
}
