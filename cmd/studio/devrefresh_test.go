package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDevCapabilityRefreshValidatesArgs(t *testing.T) {
	var buf bytes.Buffer
	if err := runDevCapabilityRefresh(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("refresh without tenant err = %v", err)
	}
}
