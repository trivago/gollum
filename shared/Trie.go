package shared

// TrieNode a node inside a suffix tree
type TrieNode struct {
	suffix    []byte
	children  []*TrieNode
	parent    *TrieNode
	Payload   interface{}
	endOfPath bool
}

// NewTrie creates a new root TrieNode
func NewTrie(data []byte, payload interface{}) *TrieNode {
	return &TrieNode{
		parent:    nil,
		suffix:    data,
		children:  []*TrieNode{},
		Payload:   payload,
		endOfPath: true,
	}
}

func (node *TrieNode) replace(oldChild *TrieNode, newChild *TrieNode) {
	for i, child := range node.children {
		if child == oldChild {
			node.children[i] = newChild
			return // ### return, replaced ###
		}
	}
}

// Add adds a new data path to the suffix tree.
// The TrieNode returned is the (new) root node.
func (node *TrieNode) Add(data []byte, payload interface{}) *TrieNode {
	suffixLen := len(node.suffix)
	dataLen := len(data)

	testLen := suffixLen
	if dataLen < suffixLen {
		testLen = dataLen
	}

	var splitIdx int
	for splitIdx = 0; splitIdx < testLen; splitIdx++ {
		if data[splitIdx] != node.suffix[splitIdx] {
			break // ### break, split found ###
		}
	}

	switch {
	case splitIdx == suffixLen:
		// Continue down or stop here (full suffix match)

		if splitIdx == dataLen {
			node.endOfPath = true
			node.Payload = payload // may overwrite
			return node            // ### return, path already stored ###
		}

		data = data[splitIdx:]
		if suffixLen > 0 {
			for _, child := range node.children {
				if child.suffix[0] == data[0] {
					child.Add(data, payload)
					return node // ### return, continue on path ###
				}
			}
		}

		newChild := NewTrie(data, payload)
		newChild.parent = node
		node.children = append(node.children, newChild)
		return node // ### return, new leaf ###

	case splitIdx == dataLen:
		// Make current node is a subpath of new data node (full data match)
		// This case implies that dataLen < suffixLen as splitIdx == suffixLen
		// did not match.

		newParent := NewTrie(data, payload)
		newParent.children = []*TrieNode{node}

		if node.parent != nil {
			node.parent.replace(node, newParent)
		}

		node.parent = newParent
		node.suffix = node.suffix[splitIdx:]
		return newParent // ### return, rotation ###

	default:
		// New parent required with both nodes as children (partial match)

		node.suffix = node.suffix[splitIdx:]
		newChild := NewTrie(data[splitIdx:], payload)

		newParent := NewTrie(data[:splitIdx], nil)
		newParent.children = []*TrieNode{node, newChild}
		newParent.endOfPath = false

		if node.parent != nil {
			node.parent.replace(node, newParent)
		}

		newChild.parent = newParent
		node.parent = newParent
		return newParent // ### return, new parent ###
	}
}

// Match compares the trie to the given data stream.
// Match returns true if data can be completely matched to the trie.
func (node *TrieNode) Match(data []byte) (*TrieNode, int) {
	dataLen := len(data)
	suffixLen := len(node.suffix)
	if dataLen < suffixLen {
		return nil, 0 // ### return, cannot be fully matched ###
	}

	for i := 0; i < suffixLen; i++ {
		if data[i] != node.suffix[i] {
			return nil, 0 // ### return, no match ###
		}
	}

	if dataLen == suffixLen {
		if node.endOfPath {
			return node, suffixLen // ### return, full match ###
		}
		return nil, 0 // ### return, invalid match ###
	}

	data = data[suffixLen:]
	for i := 0; i < len(node.children); i++ {
		matchedNode, length := node.children[i].Match(data)
		if matchedNode != nil {
			return matchedNode, length + suffixLen // ### return, match found ###
		}
	}

	return nil, 0 // ### return, no valid path ###
}

// MatchStart compares the trie to the beginning of the given data stream.
// MatchStart returns true if the beginning of data can be matched to the trie.
func (node *TrieNode) MatchStart(data []byte) (*TrieNode, int) {
	dataLen := len(data)
	suffixLen := len(node.suffix)
	if dataLen < suffixLen {
		return nil, 0 // ### return, cannot be fully matched ###
	}

	for i := 0; i < suffixLen; i++ {
		if data[i] != node.suffix[i] {
			return nil, 0 // ### return, no match ###
		}
	}

	// Match longest path first

	data = data[suffixLen:]
	for i := 0; i < len(node.children); i++ {
		matchedNode, length := node.children[i].MatchStart(data)
		if matchedNode != nil {
			return matchedNode, length + suffixLen // ### return, match found ###
		}
	}

	// May be only a part of data but we have a valid match

	if node.endOfPath {
		return node, suffixLen // ### return, full match ###
	}
	return nil, 0 // ### return, no valid path ###
}
