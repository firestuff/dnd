package main

import (
	"archive/zip"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/firestuff/dnd/internal"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

var removeWords = map[string]bool{
	"Diamond":       true,
	"Gridless":      true,
	"High":          true,
	"PSD":           true,
	"Res":           true,
	"Roll20+Tokens": true,
}

var actions = map[string]string{
	"./*.psd":                   "Maps/{MAPNAME}",
	"*/Gridless/*.jpg":          "Maps/{MAPNAME}",
	"*/High Res/Gridless/*.jpg": "Maps/{MAPNAME}",
	"*/High Res/Grid/*.jpg":     "{SKIP}",
	"*/Creature Tokens/*.png":   "Creatures",
	"*/Map Tokens/*.png":        "Maps/{MAPNAME}/Objects",
	"*/Roll20/Grid/*.jpg":       "{SKIP}",
	"*/Roll20/Gridless/*.jpg":   "{SKIP}",
}

var root = flag.String("root", "", "root directory to write to")

func main() {
	l := slog.New(slog.NewTextHandler(os.Stderr, nil))

	flag.Parse()

	if *root == "" {
		l.Error("please specify --root")
		os.Exit(1)
	}

	for _, path := range flag.Args() {
		zipL := l.With("zipPath", path)

		s := lo.Must(os.Stat(path))
		if s.Size() == 0 {
			zipL.Warn("SKIP")
			continue
		}

		ok := importZIP(zipL, path)
		if !ok {
			os.Exit(1)
		}

		lo.Must0(os.Remove(path))
	}
}

func importZIP(l *slog.Logger, path string) bool {
	name := mapName(path)
	l = l.With("mapName", name)

	z := lo.Must(zip.OpenReader(path))
	defer z.Close()

	for _, file := range z.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}

		sig := internal.PathSig(file.Name)

		dst := actions[sig]

		if dst == "" {
			parts := strings.Split(sig, "/")
			dst = actions["*/"+filepath.Join(parts[1:]...)]
		}

		if dst == "" {
			l.Error("unknown file signature",
				"filePath", file.Name,
				"signature", sig,
			)
			return false
		}

		if dst == "{SKIP}" {
			l.Info("SKIP",
				"src", file.Name,
			)
			continue
		}

		dst = strings.ReplaceAll(dst, "{MAPNAME}", name)
		dst = filepath.Join(*root, dst, filepath.Base(file.Name))

		l.Info("COPY",
			"src", file.Name,
			"dst", dst,
		)

		lo.Must0(os.MkdirAll(filepath.Dir(dst), 0755))
		dstFile := lo.Must(os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644))
		srcFile := lo.Must(file.Open())

		lo.Must(io.Copy(dstFile, srcFile))

		dstFile.Close()
		srcFile.Close()
	}

	return true
}

func mapName(path string) string {
	withoutZIP := strings.TrimSuffix(filepath.Base(path), ".zip")
	parts := strings.Split(withoutZIP, " ")

	i := len(parts) - 1
	for i >= 0 && removeWords[parts[i]] {
		i--
	}

	return strings.Join(parts[:i+1], " ")
}
