package main

import (
	"flag"
	"io/fs"
	"os"
	"strings"

	"github.com/firestuff/dnd/internal"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

var mapSigs = map[string]string{
	"./*.jpg;Objects/*.png":          "drmapzo-diamond",
	"./*.jpg;./*.png;Objects/*.png":  "drmapzo-diamond",
	"./*.jpg": "czepeku-$5",
	"./*.png": "czepeku-$5",
	"./*.jpg;./*.png": "czepeku-$5",
	"./*.jpg;Objects/*.jpg":          "czepeku-$5",
}

func main() {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))

	root := flag.String("root", "", "root directory to validate")
	flag.Parse()

	if *root == "" {
		l.Error("please specify --root")
		os.Exit(1)
	}

	rootFS := os.DirFS(*root)

	l.Info("validating...",
		"root", *root,
	)

	ok := validateMaps(l, rootFS)
	if !ok {
		os.Exit(1)
	}
}

func validateMaps(l *slog.Logger, mapsFS fs.FS) bool {
	entries := lo.Must(fs.ReadDir(mapsFS, "."))

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
	sig := internal.DirSig(mapFS)

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
