package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pythonicvarun/tw/internal/render"
	"github.com/pythonicvarun/tw/internal/tree"
	"github.com/pythonicvarun/tw/internal/walker"
)

// reorderArgs partitions args into flags (and their values) first, positionals last.
// Go's flag package stops parsing at the first non-flag token, which would otherwise
// make `tw somedir --hidden` silently ignore --hidden. Flags known to take a value
// (currently just --depth) need their following token kept attached.
func reorderArgs(args []string, flagsWithValue map[string]bool) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// handle "--depth 2" (space-separated); "--depth=2" needs no special case
			if flagsWithValue[strings.TrimLeft(a, "-")] && !strings.Contains(a, "=") {
				if i+1 < len(args) {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			positionals = append(positionals, a)
		}
	}
	return append(flags, positionals...)
}

func main() {
	depth := flag.Int("depth", 0, "max recursion depth (0 = unlimited)")
	dirsOnly := flag.Bool("dirs-only", false, "show directories only")
	filesOnly := flag.Bool("files-only", false, "show files only")
	hidden := flag.Bool("hidden", false, "show hidden (dot) files/dirs")
	noIgnore := flag.Bool("no-ignore", false, "disable .gitignore-aware filtering")

	reordered := reorderArgs(os.Args[1:], map[string]bool{"depth": true})
	flag.CommandLine.Parse(reordered)

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tw: error resolving path:", err)
		os.Exit(1)
	}

	opt := walker.Options{
		MaxDepth:         *depth,
		RespectGitignore: !*noIgnore,
		ShowHidden:       *hidden,
		DirsOnly:         *dirsOnly,
		FilesOnly:        *filesOnly,
	}

	entries, err := walker.Walk(absRoot, opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tw: error walking directory:", err)
		os.Exit(1)
	}

	rootName := filepath.Base(absRoot)
	t := tree.Build(rootName, entries)

	out := bufio.NewWriter(os.Stdout)
	render.Compact(out, t)
}
