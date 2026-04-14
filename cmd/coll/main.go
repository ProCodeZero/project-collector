package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var codeExtensions = map[string]struct{}{
	".py": {}, ".js": {}, ".ts": {}, ".jsx": {}, ".tsx": {},
	".html": {}, ".css": {}, ".scss": {}, ".sass": {},
	".java": {}, ".cpp": {}, ".c": {}, ".h": {}, ".hpp": {},
	".cs": {}, ".go": {}, ".rs": {}, ".rb": {}, ".php": {},
	".sh": {}, ".bash": {}, ".sql": {}, ".json": {},
	".yaml": {}, ".yml": {}, ".xml": {}, ".md": {}, ".txt": {},
	".vue": {}, ".svelte": {}, ".swift": {}, ".kt": {}, ".kts": {},
	".pl": {}, ".pm": {}, ".r": {}, ".R": {}, ".dart": {}, ".lua": {},
	".scala": {}, ".groovy": {}, ".clj": {}, ".cljs": {}, ".edn": {},
	".toml": {}, ".ini": {}, ".cfg": {}, ".conf": {}, ".gradle": {},
	".dockerfile": {}, ".env": {}, ".lock": {}, ".puml": {}, ".pu": {}, ".dbml": {},
}

var ignoreNamesLower = map[string]struct{}{
	"node_modules": {}, "__pycache__": {}, ".git": {}, ".svn": {}, ".hg": {},
	"venv": {}, "env": {}, ".venv": {}, "dist": {}, "build": {}, "out": {},
	"coverage": {}, ".next": {}, ".nuxt": {}, ".cache": {}, ".idea": {},
	".vscode": {}, ".vs": {}, ".pytest_cache": {}, ".mypy_cache": {}, ".tox": {},
	".eggs": {}, "tmp": {}, "temp": {}, "vendor": {}, ".env": {},
}

// Special files to include regardless of extension
var specialFiles = map[string]struct{}{
	".gitignore": {}, "package-lock.json": {}, "yarn.lock": {},
}

func main() {
	outputFile := flag.String("o", "", "Name of the output file")
	allFiles := flag.Bool("all", false, "Collect all files regardless of extension")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: coll [path] [-o output.txt] [--all]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Default to current directory if no path provided
	rootPath := "."
	if flag.NArg() > 0 {
		rootPath = flag.Arg(0)
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with an absolute path: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a directory.\n", rootPath)
		os.Exit(1)
	}

	type fileEntry struct {
		RelPath string
		Content string
	}
	var entries []fileEntry

	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			lower := strings.ToLower(name)
			if _, ignored := ignoreNamesLower[lower]; ignored {
				return filepath.SkipDir
			}
			// Skip hidden directories (except . and ..)
			if strings.HasPrefix(name, ".") && name != "." && name != ".." {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(rootAbs, path)
		relPath = filepath.ToSlash(relPath)

		if pathContainsIgnored(relPath) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		filename := d.Name()

		_, isCode := codeExtensions[ext]
		_, isSpecial := specialFiles[strings.ToLower(filename)]

		if *allFiles || isCode || isSpecial {
			entries = append(entries, fileEntry{RelPath: relPath, Content: readFileContent(path)})
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during the work with directory: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "Didn't found any comparable files.")
		os.Exit(1)
	}

	outName := *outputFile
	if outName == "" {
		outName = filepath.Base(rootAbs) + ".txt"
	}

	f, err := os.Create(outName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create a file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	for _, e := range entries {
		fmt.Fprintf(f, "./%s:\n```\n%s\n```\n", e.RelPath, e.Content)
	}

	outAbs, _ := filepath.Abs(outName)
	fmt.Printf("Done! The result: %s\n", outAbs)
}

func pathContainsIgnored(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		lower := strings.ToLower(part)
		if _, ok := ignoreNamesLower[lower]; ok {
			return true
		}
		if strings.HasPrefix(part, ".") && part != parts[len(parts)-1] {
			return true
		}
	}
	return false
}

func readFileContent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("[Error reading file: %v]", err)
	}
	if utf8.Valid(data) {
		return string(data)
	}
	var b strings.Builder
	b.Grow(len(data))
	for _, c := range data {
		b.WriteRune(rune(c))
	}
	return b.String()
}
