package tree

import "sort"

// Node represents a single file or directory in the tree.
type Node struct {
	Name     string
	IsDir    bool
	Size     int64
	Children []*Node
}

// AddChild appends a child node.
func (n *Node) AddChild(c *Node) {
	n.Children = append(n.Children, c)
}

// SortChildren sorts children: directories first, then alphabetically within each group.
func (n *Node) SortChildren() {
	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // dirs before files
		}
		return a.Name < b.Name
	})
	for _, c := range n.Children {
		if c.IsDir {
			c.SortChildren()
		}
	}
}

// CountAll returns total dir count and file count under this node (inclusive of children only, not self).
func (n *Node) CountAll() (dirs, files int) {
	for _, c := range n.Children {
		if c.IsDir {
			dirs++
			cd, cf := c.CountAll()
			dirs += cd
			files += cf
		} else {
			files++
		}
	}
	return
}
