package internal

import (
	"io/fs"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samber/lo"
)

func DirSig(root fs.FS) string {
	sigSet := map[string]bool{}

	fs.WalkDir(root, ".", func(path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			return nil
		}

		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}

		sigSet[PathSig(path)] = true

		return nil
	})

	sigs := lo.Keys(sigSet)
	sort.Strings(sigs)

	return strings.Join(sigs, ";")
}

func PathSig(path string) string {
	parts := strings.Split(filepath.Base(path), ".")
	return fmt.Sprintf("%s/*.%s", filepath.Dir(path), lo.Must(lo.Last(parts)))
}
