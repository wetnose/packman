package script

import (
	_ "embed"
	"errors"
	"github.com/stretchr/testify/assert"
	"log"
	"maps"
	"os"
	"packman/file"
	. "packman/test"
	"slices"
	"strings"
	"testing"
)

//go:embed test/export.pman
var exportPman []byte

//go:embed test/import.pman
var importPman []byte

//go:embed test/list-exp.txt
var listExp []byte

func TestExport(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	Check(t, assert.NoError(t, os.Mkdir("test/tmp", 0770)))

	s, err := Parse(exportPman)
	Check(t, assert.NoError(t, err))

	s.Run(log.Printf)
	loc, err := file.LocalTree("test/tmp")
	Check(t, assert.NoError(t, err))

	files := maps.Collect(loc.Find(""))
	Check(t, assert.Equal(t, 4, len(files)))

	names := slices.Collect(maps.Keys(files))
	slices.Sort(names)

	assert.Equal(t, string(listExp), strings.Join(names, "\n"))
}

func TestImport(t *testing.T) {
	impPath := "test/tmp/imp.vpk"
	_ = os.RemoveAll(impPath)
	_, err := os.Stat(impPath)
	Check(t, assert.True(t, errors.Is(err, os.ErrNotExist)))

	s, err := Parse(importPman)
	Check(t, assert.NoError(t, err))

	s.Run(log.Printf)

	exp, err := os.ReadFile("test/local.vpk")
	Check(t, assert.NoError(t, err))

	act, err := os.ReadFile(impPath)
	Check(t, assert.NoError(t, err))

	assert.Equal(t, exp, act)
}
