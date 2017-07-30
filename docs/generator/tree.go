// Code for working with Golang's Abstract Syntax Trees

package main

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

/**
*ast.Package
    *ast.File
        *ast.Ident
        *ast.GenDecl
            *ast.ImportSpec
                *ast.BasicLit
        [....]
        *ast.GenDecl
            *ast.CommentGroup
                *ast.Comment
            *ast.TypeSpec
                *ast.Ident
                *ast.StructType
                    *ast.FieldList
                        *ast.Field
                            *ast.Ident
                            *ast.Ident
        [....[
**/

// TreeNode represents one node in a tree and links to its parent and children.
// Trees of TreeNodes are used to represent the AST returned by the go/ast package.
type TreeNode struct {
	AstNode  ast.Node
	Depth    int
	Parent   *TreeNode
	Children []*TreeNode
}

// PatternNode represents one node in a search pattern tree. Pattern trees are used
// to search for subtrees in TreeNode trees.
type PatternNode struct {
	Comparison string
	Callback   func(ast.Node) bool
	Children   []PatternNode
}

// A cursor for generating a tree of TreeNodes
type treeCursor struct {
	root         *TreeNode
	current      *TreeNode
	currentDepth int
}

// NewTree creates a new tree of TreeNodes from the contents of the Go AST rooted at rootAstNode.
// Returns a pointer to the root TreeNode.
func NewTree(rootAstNode ast.Node) *TreeNode {
	treeCursor := treeCursor{}

	// Stub root
	treeCursor.root = &TreeNode{
		AstNode:  nil,
		Parent:   nil,
		Children: []*TreeNode{},
	}
	treeCursor.current = treeCursor.root
	treeCursor.currentDepth = 0

	// Populate the tree
	ast.Inspect(rootAstNode, treeCursor.insert)

	// Return the actual root
	return treeCursor.root.Children[0]
}

// Callback for inserting data into the tree.
// go.ast.Inspect() calls this for each node in the AST
func (treeCursor *treeCursor) insert(astNode ast.Node) bool {
	if astNode == nil {
		// End of branch, go up
		treeCursor.currentDepth--
		treeCursor.current = treeCursor.current.Parent
		return false
	}

	// Allocate memory for a new TreeNode
	treeNode := &TreeNode{}

	treeNode.AstNode = astNode
	treeNode.Depth = treeCursor.currentDepth
	treeNode.Parent = treeCursor.current

	// Append new node to children of the current node
	treeCursor.current.Children = append(treeCursor.current.Children, treeNode)

	// Update cursor, go down
	treeCursor.current = treeNode
	treeCursor.currentDepth++
	return true
}

// Search iterates recursively through all subtrees of TreeNode and compares them to
// the pattern three rooted at patternNode. It returns a flat list containing the
// matching subtrees' roots.
func (treeNode *TreeNode) Search(patternNode PatternNode) []*TreeNode {
	results := []*TreeNode{}

	// Current node
	if treeNode.compare(patternNode) {
		results = append(results, treeNode)
	}

	// Current node's children
	for _, child := range treeNode.Children {
		for _, subResult := range child.Search(patternNode) {
			results = append(results, subResult)
		}
	}

	return results
}

// Compares tree rooted at `treeNode` to pattern tree rooted at `patternNode`.
// Iterates recursively through both, returns true if they match.
//
// Note: patternNode matches subtrees; it doesn't have to specify all child nodes
// down to the leaves. If the pattern tree does contains child nodes at any level,
// then _all_ child nodes must match between treeNode and pattern at that level.
func (treeNode *TreeNode) compare(pattern PatternNode) bool {
	// Check that this level matches
	if treeNode.getComparison() != pattern.Comparison {
		return false
	}
	if pattern.Callback != nil && !pattern.Callback(treeNode.AstNode) {
		return false
	}

	// Check children iff pattern specifies them, otherwise skip them
	if len(pattern.Children) == 0 {
		return true
	}

	// Mismatch in child count => no match
	if len(pattern.Children) != len(treeNode.Children) {
		return false
	}

	// Check children; assume same order
	for i := 0; i < len(treeNode.Children); i++ {
		if !treeNode.Children[i].compare(pattern.Children[i]) {
			return false
		}
	}

	// All children matched
	return true
}

// Returns a value to use when comparing this treeNode to a PatternNode.
// Currently simply the string representation of the node's Go type is used.
func (treeNode *TreeNode) getComparison() string {
	return reflect.TypeOf(treeNode.AstNode).String()
}

// Dump prints a dumpString of the tree rooted at `treeNode` to stdout
func (treeNode *TreeNode) Dump() {
	fmt.Printf("%s<%p>%q\n", strings.Repeat(" ", 4*treeNode.Depth), treeNode, treeNode.astNodeDump())
	for _, child := range treeNode.Children {
		child.Dump()
	}
}

func (treeNode *TreeNode) astNodeDump() string {
	if treeNode.AstNode == nil {
		return "*nil*"
	}
	str := reflect.TypeOf(treeNode.AstNode).String()

	switch t := treeNode.AstNode.(type) {
	case *ast.Ident:
		// Relative reference within the same package ("SimpleConsumer")
		str += "[" + treeNode.AstNode.(*ast.Ident).Name + "]"

	case *ast.SelectorExpr:
		// Reference to another package ("core.SimpleConsumer")
		sel := treeNode.AstNode.(*ast.SelectorExpr)
		str += "." + sel.Sel.Name

	case *ast.Comment:
		str += "[" + treeNode.AstNode.(*ast.Comment).Text + "]"

	default:
		_ = t
	}

	return str
}
