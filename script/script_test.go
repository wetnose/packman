package script

import (
	_ "embed"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"maps"
	"os"
	"packman/file"
	"packman/file/vpk"
	"slices"
	"strings"
	"testing"
)

//go:embed test/export.pman
var exportPman []byte

//go:embed test/import.pman
var importPman []byte

//go:embed test/patch.pman
var patchPman []byte

//go:embed test/list-exp.txt
var listExp []byte

func TestExport(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	s, err := Parse(exportPman)
	require.NoError(t, err)

	require.NoError(t, s.Run(log.Printf))
	loc, err := file.LocalTree("test/tmp")
	require.NoError(t, err)

	files := maps.Collect(loc.Find(""))
	require.Equal(t, 7, len(files))

	names := slices.Collect(maps.Keys(files))
	slices.Sort(names)

	assert.Equal(t, string(listExp), strings.Join(names, "\n"))
}

func TestImport(t *testing.T) {
	impPath := "test/tmp/imp.vpk"
	_ = os.RemoveAll(impPath)
	_, err := os.Stat(impPath)
	require.True(t, errors.Is(err, os.ErrNotExist))

	s, err := Parse(importPman)
	require.NoError(t, err)

	require.NoError(t, s.Run(log.Printf))

	exp, err := os.ReadFile("test/local.vpk")
	require.NoError(t, err)

	act, err := os.ReadFile(impPath)
	require.NoError(t, err)

	assert.Equal(t, exp, act)
}

func TestUnknown(t *testing.T) {
	_, err := Parse([]byte(`check X:`))
	require.Error(t, err)
}

func TestPatch(t *testing.T) {
	patchPath := "test/tmp/patch.vpk"
	_ = os.RemoveAll(patchPath)
	_, err := os.Stat(patchPath)
	require.True(t, errors.Is(err, os.ErrNotExist))

	s, err := Parse(patchPman)
	require.NoError(t, err)

	require.NoError(t, s.Run(log.Printf))

	d, err := vpk.Read(patchPath)
	require.NoError(t, err)

	var data []string
	for _, e := range d.Find("") {
		buf, err := e.GetData()
		require.NoError(t, err)
		data = append(data, string(buf))
	}

	slices.Sort(data)
	require.Equal(t, "file01 file02 file11 file12 file121 file22", strings.Join(data, " "))
}

func TestCopyFile(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	s, err := Parse([]byte(`
		bind A .:test/tmp
		bind B .:test/local.vpk
		copy B:dir1/file12.txt A:dirX/f1.txt
	`))
	require.NoError(t, err)
	require.NoError(t, s.Run(log.Printf))

	require.FileExists(t, "test/tmp/dirX/f1.txt")
}

func TestMem(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	s, err := Parse([]byte(`
		bind  A
		bind  B .:test/local.vpk
		bind  T .:test/tmp
		clone B:dir2 A:
		clone A: T:
	`))
	require.NoError(t, err)
	require.NoError(t, s.Run(log.Printf))
	require.FileExists(t, "test/tmp/dir2/file22.txt")
}

func TestRegex(t *testing.T) {
	_ = os.RemoveAll("test/tmp")
	require.NoError(t, os.Mkdir("test/tmp", 0770))

	s, err := Parse([]byte(`
		bind  B .:test/local.vpk
		bind  T .:test/tmp
		clone -e B:file1 T:
	`))
	require.NoError(t, err)
	require.NoError(t, s.Run(log.Printf))
	//require.FileExists(t, "test/tmp/dir2/file22.txt")
}
