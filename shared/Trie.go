package shared

// TrieNode a node inside a suffix tree
type TrieNode struct {
	suffix   []byte
	children []*TrieNode
	parent   *TrieNode
	Payload  interface{}
	PathLen  int
}

// NewTrie creates a new root TrieNode
func NewTrie(data []byte, payload interface{}) *TrieNode {
	return &TrieNode{
		parent:   nil,
		suffix:   data,
		children: []*TrieNode{},
		Payload:  payload,
		PathLen:  len(data),
	}
}

func (node *TrieNode) addNewChild(data []byte, payload interface{}, pathLen int) {
	child := &TrieNode{
		parent:   node,
		suffix:   data,
		children: []*TrieNode{},
		Payload:  payload,
		PathLen:  pathLen,
	}
	node.children = append(node.children, child)
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
	return node.addPath(data, payload, len(data))
}

func (node *TrieNode) addPath(data []byte, payload interface{}, pathLen int) *TrieNode {
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
			node.Payload = payload // may overwrite
			return node            // ### return, path already stored ###
		}

		data = data[splitIdx:]
		if suffixLen > 0 {
			for _, child := range node.children {
				if child.suffix[0] == data[0] {
					child.addPath(data, payload, pathLen)
					return node // ### return, continue on path ###
				}
			}
		}

		node.addNewChild(data, payload, pathLen)
		return node // ### return, new leaf ###

	case splitIdx == dataLen:
		// Make current node a subpath of new data node (full data match)
		// This case implies that dataLen < suffixLen as splitIdx == suffixLen
		// did not match.

		newParent := NewTrie(data, payload)
		newParent.PathLen = pathLen
		newParent.children = []*TrieNode{node}

		if node.parent != nil {
			node.parent.replace(node, newParent)
		}

		node.parent = newParent
		node.suffix = node.suffix[splitIdx:]
		return newParent // ### return, rotation ###

	default:
		// New parent required with both nodes as children (partial match)

		newParent := NewTrie(data[:splitIdx], nil)
		newParent.PathLen = 0
		newParent.children = []*TrieNode{node}

		newParent.addNewChild(data[splitIdx:], payload, pathLen)

		if node.parent != nil {
			node.parent.replace(node, newParent)
		}

		node.suffix = node.suffix[splitIdx:]
		node.parent = newParent
		return newParent // ### return, new parent ###
	}
}

// Match compares the trie to the given data stream.
// Match returns true if data can be completely matched to the trie.
func (node *TrieNode) Match(data []byte) *TrieNode {
	dataLen := len(data)
	suffixLen := len(node.suffix)
	if dataLen < suffixLen {
		return nil // ### return, cannot be fully matched ###
	}

	for i := 0; i < suffixLen; i++ {
		if data[i] != node.suffix[i] {
			return nil // ### return, no match ###
		}
	}

	if dataLen == suffixLen {
		if node.PathLen > 0 {
			return node // ### return, full match ###
		}
		return nil // ### return, invalid match ###
	}

	data = data[suffixLen:]
	for i := 0; i < len(node.children); i++ {
		matchedNode := node.children[i].Match(data)
		if matchedNode != nil {
			return matchedNode // ### return, match found ###
		}
	}

	return nil // ### return, no valid path ###
}

// MatchStart compares the trie to the beginning of the given data stream.
// MatchStart returns true if the beginning of data can be matched to the trie.
func (node *TrieNode) MatchStart(data []byte) *TrieNode {
	dataLen := len(data)
	suffixLen := len(node.suffix)
	if dataLen < suffixLen {
		return nil // ### return, cannot be fully matched ###
	}

	for i := 0; i < suffixLen; i++ {
		if data[i] != node.suffix[i] {
			return nil // ### return, no match ###
		}
	}

	// Match longest path first

	data = data[suffixLen:]
	for i := 0; i < len(node.children); i++ {
		matchedNode := node.children[i].MatchStart(data)
		if matchedNode != nil {
			return matchedNode // ### return, match found ###
		}
	}

	// May be only a part of data but we have a valid match

	if node.PathLen > 0 {
		return node // ### return, full match ###
	}
	return nil // ### return, no valid path ###
}
