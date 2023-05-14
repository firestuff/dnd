package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

var mapSigs = map[string]string{
	"./*.jpg;./*.psd;Objects/*.png": "patreon-drmapzo-diamond",
}

func main() {
	root := flag.String("root", "", "root directory to validate")

	flag.Parse()

	rootFS := os.DirFS(*root)
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))

	l.Info("validating...",
		"root", *root,
	)

	ok := validateMaps(l, lo.Must(fs.Sub(rootFS, "Maps")))
	if !ok {
		os.Exit(1)
	}
}

func validateMaps(l *slog.Logger, mapsFS fs.FS) bool {
	entries := lo.Must(mapsFS.(fs.ReadDirFS).ReadDir("."))

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		ok := validateMap(
			l.With("map", entry.Name()),
			lo.Must(fs.Sub(mapsFS, entry.Name())),
		)
		if !ok {
			return false
		}
	}

	return true
}

func validateMap(l *slog.Logger, mapFS fs.FS) bool {
	sig := getDirSig(mapFS)

	t := mapSigs[sig]
	if t == "" {
		l.Error("unrecognized signature",
			"signature", sig,
		)
		return false
	}

	l.Info("valid map",
		"source", t,
	)
	return true
}

func getDirSig(root fs.FS) string {
	sigSet := map[string]bool{}

	fs.WalkDir(root, ".", func(path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			return nil
		}

		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}

		sigSet[getPathSig(path)] = true

		return nil
	})

	sigs := lo.Keys(sigSet)
	sort.Strings(sigs)

	return strings.Join(sigs, ";")
}

func getPathSig(path string) string {
	parts := strings.Split(filepath.Base(path), ".")
	return fmt.Sprintf("%s/*.%s", filepath.Dir(path), lo.Must(lo.Last(parts)))
}
