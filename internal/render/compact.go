package render

import (
	"bufio"
	"fmt"

	"github.com/pythonicvarun/tw/internal/tree"
)

// Compact writes an indentation-based tree (no box-drawing chars) to w.
// This is the default for non-TTY output, optimized for token count over aesthetics.
func Compact(w *bufio.Writer, root *tree.Node) {
	fmt.Fprintln(w, root.Name+"/")
	for _, c := range root.Children {
		compactNode(w, c, 1)
	}
	w.Flush()
}

func compactNode(w *bufio.Writer, n *tree.Node, depth int) {
	for range depth {
		w.WriteString("  ")
	}

	if n.IsDir {
		w.WriteString(n.Name)
		w.WriteString("/\n")
		for _, c := range n.Children {
			compactNode(w, c, depth+1)
		}
	} else {
		w.WriteString(n.Name)
		w.WriteString("\n")
	}
}
