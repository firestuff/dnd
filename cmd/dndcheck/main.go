package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samber/lo"
)

var mapSigs = map[string]string{
	"./*.jpg;./*.psd;Objects/*.png": "patreon-drmapzo-diamond",
}

func main() {
	root := flag.String("root", "", "root directory to validate")

	flag.Parse()

	if *root == "" {
		log.Fatal("please specify --root")
	}

	log.Printf("validating: %s", *root)

	rootFS := os.DirFS(*root)

	validateMaps(lo.Must(fs.Sub(rootFS, "Maps")))
}

func validateMaps(mapsFS fs.FS) {
	entries := lo.Must(mapsFS.(fs.ReadDirFS).ReadDir("."))

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		log.Printf("map: %s", entry.Name())

		mapFS := lo.Must(fs.Sub(mapsFS, entry.Name()))
		validateMap(mapFS)
	}
}

func validateMap(mapFS fs.FS) {
	sig := getDirSig(mapFS)

	t := mapSigs[sig]
	if t == "" {
		log.Fatalf("\tunrecognized signature: %s", sig)
	}

	log.Printf("\ttype: %s", t)
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
