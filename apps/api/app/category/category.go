// Package category is the tree a space organises its documents by: at most
// three levels, ordered among siblings.
//
// The depth cap lives here rather than in the schema because a CHECK cannot
// look at other rows to say "the parent's parent has no parent". It is checked against the prospective
// parent's own depth on every create and move, so the only way to exceed it is
// to write the rows directly.
package category

import (
	"api-kms/app"

	"github.com/google/uuid"
)

// MaxDepth is FR-2's three levels, counted from 1 for a root category.
const MaxDepth = 3

type Category struct {
	app.BaseModel `gorm:"embedded"`
	SpaceID       uuid.UUID  `json:"space_id" gorm:"column:space_id"`
	ParentID      *uuid.UUID `json:"parent_id" gorm:"column:parent_id"`
	Name          string     `json:"name" gorm:"column:name"`
	Slug          string     `json:"slug" gorm:"column:slug"`
	// Order is the position among siblings, renumbered 0..n-1 whenever the
	// list is reordered.
	Order int `json:"order" gorm:"column:order"`
}

func (Category) TableName() string {
	return "categories"
}

// Node is one category with its children attached — the shape the tree is
// returned in. Built in the service from a single flat query, not by a query
// per level.
type Node struct {
	Category
	Children []Node
	// DocumentCount includes everything beneath this node, because that is
	// what clicking it lists. A number that counted only direct children
	// would read 0 beside a category holding a whole subtree of work.
	DocumentCount int64
}

// BuildTree assembles the flat rows into roots-with-children, preserving the
// order the rows arrived in (the query sorts by `order`).
//
// A row whose parent is not in the slice is dropped rather than promoted to a
// root: that only happens when the caller filtered the query, and silently
// re-parenting it would draw the category in the wrong place.
func BuildTree(rows []Category) []Node {
	byParent := make(map[uuid.UUID][]Category, len(rows))
	for _, row := range rows {
		parent := uuid.Nil
		if row.ParentID != nil {
			parent = *row.ParentID
		}
		byParent[parent] = append(byParent[parent], row)
	}
	return subtree(byParent, uuid.Nil, 1)
}

// depth stops the recursion at MaxDepth. The service already refuses to create
// a cycle, so this only guards rows written straight to the table — but an
// unbounded walk over one is a stack overflow, not a wrong answer.
func subtree(byParent map[uuid.UUID][]Category, parent uuid.UUID, depth int) []Node {
	rows := byParent[parent]
	if len(rows) == 0 || depth > MaxDepth {
		return nil
	}

	nodes := make([]Node, len(rows))
	for i, row := range rows {
		nodes[i] = Node{Category: row, Children: subtree(byParent, row.ID, depth+1)}
	}
	return nodes
}

// PathedCategory is one category flattened back to a row, carrying its full
// path ("Backend / Database / Migrations") instead of its bare name.
//
// A dropdown built from bare names is ambiguous the moment two branches both
// hold a "Migrations", and the flat list gives no clue which is which.
type PathedCategory struct {
	ID   uuid.UUID
	Path string
}

// FlattenPaths walks the tree depth-first, so the options come out in the same
// order the tree is drawn in.
func FlattenPaths(nodes []Node) []PathedCategory {
	var out []PathedCategory
	appendPaths(nodes, "", &out)
	return out
}

func appendPaths(nodes []Node, prefix string, out *[]PathedCategory) {
	for i := range nodes {
		path := nodes[i].Name
		if prefix != "" {
			path = prefix + " / " + path
		}
		*out = append(*out, PathedCategory{ID: nodes[i].ID, Path: path})
		appendPaths(nodes[i].Children, path, out)
	}
}

// Rollup adds each node's own document count to its descendants', so the
// number beside a category is the number clicking it will show.
//
// Returns the total across the whole tree, which the caller uses for the
// "all documents" row without summing the roots itself.
func Rollup(nodes []Node, own map[uuid.UUID]int64) int64 {
	var total int64
	for i := range nodes {
		below := Rollup(nodes[i].Children, own)
		nodes[i].DocumentCount = own[nodes[i].ID] + below
		total += nodes[i].DocumentCount
	}
	return total
}
