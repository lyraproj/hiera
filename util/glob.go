package util

import (
	"path/filepath"
	"os"
	"github.com/gobwas/glob"
)

func Glob(root string, pattern string) (matches []string, e error) {
	return getDirectoryTree(root, glob.MustCompile(filepath.FromSlash(pattern), filepath.Separator))
}

func getDirectoryTree(root string, g glob.Glob) ([]string, error) {
	matches := make([]string, 0, 64)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() != `.` && g.Match(path) {
			matches = append(matches, path)
		}
		return err
	})
	if err != nil {
		matches = nil
	}
	return matches, err
}
