package tree

import (
	"path/filepath"
	"strings"

	"github.com/pythonicvarun/tw/internal/walker"
)

// Build assembles a tree of Nodes from a flat list of walker entries.
// rootName is used as the label for the top-level node.
func Build(rootName string, entries []walker.Entry) *Node {
	root := &Node{Name: rootName, IsDir: true}

	// index by relative path for quick parent lookup
	index := map[string]*Node{"": root}

	for _, e := range entries {
		node := &Node{Name: filepath.Base(e.RelPath), IsDir: e.IsDir, Size: e.Size}
		index[e.RelPath] = node

		parentPath := ""
		if idx := strings.LastIndex(e.RelPath, "/"); idx != -1 {
			parentPath = e.RelPath[:idx]
		}
		parent, ok := index[parentPath]
		if !ok {
			// Parent not yet indexed (shouldn't normally happen since walker
			// discovers directories before their children); fall back to root.
			parent = root
		}
		parent.AddChild(node)
	}

	root.SortChildren()
	return root
}
