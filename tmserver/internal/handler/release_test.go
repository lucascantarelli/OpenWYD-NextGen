package handler

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/content"
)

// readRelease is os.ReadFile for tests that need the Release/ content tree.
// A missing file skips instead of failing: the content catalogs moved into
// Postgres and out of the repository (issue 58, fase 5), so a checkout without
// the proprietary tree must still run the rest of the suite.
func readRelease(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("Release content unavailable: %v", err)
	}
	return b, err
}

// requireItemList loads the item catalog from a content tree, skipping the test
// when the tree is not mounted.
func requireItemList(t *testing.T, path string) (*content.ItemList, error) {
	t.Helper()
	l, err := content.LoadItemList(path)
	if err != nil {
		t.Skipf("Release content unavailable: %v", err)
	}
	return l, err
}

// releasePath returns the path unchanged, skipping the test when the file is not
// present — for loaders that take a path and wrap the open error themselves.
func releasePath(t *testing.T, path string) string {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("Release content unavailable: %v", err)
	}
	return path
}
