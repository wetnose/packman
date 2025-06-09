package test

import (
	"testing"
)

func Check(t *testing.T, ok bool) {
	if !ok {
		t.FailNow()
	}
}
