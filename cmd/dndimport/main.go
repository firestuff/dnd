package main

import (
	"archive/zip"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
	"github.com/firestuff/dnd/internal"
	"github.com/gobwas/glob"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

var removeWords = map[string]bool{
	"+":             true,
	"-":             true,
	"":              true,
	"-diamond":      true,
	"1":             true,
	"1)":            true,
	"2":             true,
	"2)":            true,
	"$5":            true,
	"rewards":       true,
	"diamond":       true,
	"gridless":      true,
	"(gridless)":    true,
	"(gridless":     true,
	"high":          true,
	"high-res":      true,
	"part":          true,
	"pt":            true,
	"pt.":           true,
	"pt.1":          true,
	"pt.1)":         true,
	"pt.2":          true,
	"pt.2)":         true,
	"pt.3":          true,
	"psd":           true,
	"res":           true,
	"roll20":        true,
	"roll20+tokens": true,
	"support":       true,
	"tier":          true,
	"tokens":        true,
	"(tokens)":      true,
	"rewards(1)":    true,
	"transparent":   true,
	"pngs":          true,
}

var actions = map[string]string{
	"Gridless/*.jpg":                      "Maps/{MAPNAME}",
	"*/*.png":                             "Maps/{MAPNAME}",
	"*/*.jpg":                             "Maps/{MAPNAME}",
	"**ds_store":                          "{SKIP}",
	"**/*.db":                             "{SKIP}",
	"**/*.pdf":                            "{SKIP}",
	"**/*.mp3":                            "{SKIP}",
	"**/*.zip":                            "{SKIP}",
	"__macosx/**":                         "{SKIP}",
	"**/high resolution/*.jpg":            "Maps/{MAPNAME}", // Trust that it's gridless
	"**/high resolution/*.png":            "Maps/{MAPNAME}", // Trust that it's gridless
	"**/gridless/*.jpg":                   "Maps/{MAPNAME}",
	"**/gridless/*.jpeg":                  "Maps/{MAPNAME}",
	"**/high-res/gridless/*.jpg":          "Maps/{MAPNAME}",
	"**/high-res/gridless/*.png":          "Maps/{MAPNAME}",
	"**/high res/gridless/*.jpg":          "Maps/{MAPNAME}",
	"**/high res/gridless/*.png":          "Maps/{MAPNAME}",
	"**/high res/gridless/attic/*.jpg":    "Maps/{MAPNAME}",
	"**/high res/gridless/attic/*.png":    "Maps/{MAPNAME}",
	"**/high res/gridless/basement/*.jpg": "Maps/{MAPNAME}",
	"**/high res/gridless/basement/*.png": "Maps/{MAPNAME}",
	"**/high res/gridless/floor 1/*.jpg":  "Maps/{MAPNAME}",
	"**/high res/gridless/floor 1/*.png":  "Maps/{MAPNAME}",
	"**/high res/gridless/floor 2/*.jpg":  "Maps/{MAPNAME}",
	"**/high res/gridless/floor 2/*.png":  "Maps/{MAPNAME}",
	"**/high resolution/gridless/*.jpg":   "Maps/{MAPNAME}",
	"**/grid/*.jpg":                       "{SKIP}",
	"**/gridded/*.jpg":                    "{SKIP}",
	"**/high-res/grid/*.jpg":              "{SKIP}",
	"**/high res/grid/*.jpg":              "{SKIP}",
	"**/high resolution/grid/*.jpg":       "{SKIP}",
	"**/creature tokens/*.png":            "Creatures",
	"**/creature tokens/variants/*.png":   "Creatures",
	"**/map tokens/*.png":                 "Maps/{MAPNAME}/Objects",
	"**/tokens/*.png":                     "Maps/{MAPNAME}/Objects",
	"**/tokens/*.jpg":                     "Maps/{MAPNAME}/Objects",
	"**/roll20/**":                        "{SKIP}",
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

		sig := strings.ToLower(internal.PathSig(file.Name))
		dst := ""

		for pattern, action := range actions {
			g := glob.MustCompile(pattern)
			if g.Match(sig) {
				dst = action
				break
			}
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
		dst = printable(filepath.Join(*root, dst, filepath.Base(file.Name)))

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

	if len(parts) == 1 {
		parts = camelcase.Split(parts[0])
	}

	i := len(parts) - 1
	for i >= 0 && (removeWords[strings.ToLower(parts[i])] || strings.HasPrefix(parts[i], "[")) {
		i--
	}

	parts = parts[:i+1]

	for len(parts) > 0 && removeWords[strings.ToLower(parts[0])] {
		parts = parts[1:]
	}

	if len(parts) == 0 {
		panic(path)
	}

	return strings.Join(parts, " ")
}

func printable(in string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, in)
}
