package trie

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/eager7/mpt_tree/common"
)

var (
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyState is the known hash of an empty state trie entry.
	//emptyState = crypto.Keccak256Hash(nil)
	emptyState = common.Hash{}
)

type LeafCallback func(leaf []byte, parent common.Hash) error

type Trie struct {
	db                   *Database
	root                 node
	originalRoot         common.Hash
	cacheGen, cacheLimit uint16
}

func New(root common.Hash, db *Database) (*Trie, error) {
	if db == nil {
		panic("trie.New called without a database")
	}
	trie := &Trie{
		db:           db,
		originalRoot: root,
	}
	if (root != common.Hash{}) && root != emptyRoot {
		rootNode, err := trie.resolveHash(root[:], nil)
		if err != nil {
			return nil, err
		}
		trie.root = rootNode
	}
	return trie, nil
}

func (t *Trie) resolveHash(n hashNode, prefix []byte) (node, error) {
	//cacheMissCounter.Inc(1)

	hash := common.BytesToHash(n)

	enc, err := t.db.Node(hash)
	if err != nil || enc == nil {
		return nil, errors.New(fmt.Sprintf("missing trie node %x (path %x)", hash, prefix))
	}
	return mustDecodeNode(n, enc, t.cacheGen), nil
}

func (t *Trie) SetCacheLimit(l uint16) {
	t.cacheLimit = l
}

// newFlag returns the cache flag value for a newly created node.
func (t *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true, gen: t.cacheGen}
}

func (t *Trie) NodeIterator(start []byte) NodeIterator {
	return newNodeIterator(t, start)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Trie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGet(key []byte) ([]byte, error) {
	key = keyBytesToHex(key)
	value, newRoot, didResolve, err := t.tryGet(t.root, key, 0)
	if err == nil && didResolve {
		t.root = newRoot
	}
	return value, err
}

func (t *Trie) tryGet(origNode node, key []byte, pos int) (value []byte, newNode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n, false, nil
		}
		value, newNode, didResolve, err = t.tryGet(n.Val, key, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newNode
			n.flags.gen = t.cacheGen
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newNode, didResolve, err = t.tryGet(n.Children[key[pos]], key, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.flags.gen = t.cacheGen
			n.Children[key[pos]] = newNode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := t.resolveHash(n, key[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newNode, _, err := t.tryGet(child, key, pos)
		return value, newNode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

func (t *Trie) Root() []byte {
	hash := t.Hash()
	return hash.Bytes()
}

func (t *Trie) Hash() common.Hash {
	hash, cached, _ := t.hashRoot(nil, nil)
	t.root = cached
	return common.BytesToHash(hash.(hashNode))
}
func (t *Trie) hashRoot(db *Database, onleaf LeafCallback) (node, node, error) {
	if t.root == nil {
		return hashNode(emptyRoot.Bytes()), nil, nil
	}
	h := newHasher(t.cacheGen, t.cacheLimit, onleaf)
	defer returnHasherToPool(h)
	return h.hash(t.root, db, true)
}

func (t *Trie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryUpdate(key, value []byte) error {
	k := keyBytesToHex(key)
	if len(value) != 0 {
		_, n, err := t.insert(t.root, nil, k, valueNode(value))
		if err != nil {
			return err
		}
		t.root = n
	} else {
		_, n, err := t.delete(t.root, nil, k)
		if err != nil {
			return err
		}
		t.root = n
	}
	return nil
}
func (t *Trie) insert(n node, prefix, key []byte, value node) (bool, node, error) {
	if len(key) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchLen := prefixLen(key, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchLen == len(n.Key) {
			dirty, nn, err := t.insert(n.Val, append(prefix, key[:matchLen]...), key[matchLen:], value)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, t.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: t.newFlag()}
		var err error
		_, branch.Children[n.Key[matchLen]], err = t.insert(nil, append(prefix, n.Key[:matchLen+1]...), n.Key[matchLen+1:], n.Val)
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key[matchLen]], err = t.insert(nil, append(prefix, key[:matchLen+1]...), key[matchLen+1:], value)
		if err != nil {
			return false, nil, err
		}
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchLen == 0 {
			return true, branch, nil
		}
		// Otherwise, replace it with a short node leading up to the branch.
		return true, &shortNode{key[:matchLen], branch, t.newFlag()}, nil

	case *fullNode:
		dirty, nn, err := t.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn
		return true, n, nil

	case nil:
		return true, &shortNode{key, value, t.newFlag()}, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.insert(rn, prefix, key, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

func (t *Trie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryDelete(key []byte) error {
	k := keyBytesToHex(key)
	_, n, err := t.delete(t.root, nil, k)
	if err != nil {
		return err
	}
	t.root = n
	return nil
}

// delete returns the new root of the trie with key deleted.
// It reduces the trie to minimal form by simplifying
// nodes on the way up after deleting recursively.
func (t *Trie) delete(n node, prefix, key []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchLen := prefixLen(key, n.Key)
		if matchLen < len(n.Key) {
			return false, n, nil // don't replace n on mismatch
		}
		if matchLen == len(key) {
			return true, nil, nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		dirty, child, err := t.delete(n.Val, append(prefix, key[:len(n.Key)]...), key[len(n.Key):])
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, t.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, t.newFlag()}, nil
		}

	case *fullNode:
		dirty, nn, err := t.delete(n.Children[key[0]], append(prefix, key[0]), key[1:])
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn

		// Check how many non-nil entries are left after deleting and
		// reduce the full node to a short node if only one entry is
		// left. Since n must've contained at least two children
		// before deletion (otherwise it would not be a full node) n
		// can never be reduced to nil.
		//
		// When the loop is done, pos contains the index of the single
		// value that is left in n or -2 if n contains at least two
		// values.
		pos := -1
		for i, cld := range n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				// If the remaining entry is a short node, it replaces
				// n and its key gets the missing nibble tacked to the
				// front. This avoids creating an invalid
				// shortNode{..., shortNode{...}}.  Since the entry
				// might not be loaded yet, resolve it just for this
				// check.
				cNode, err := t.resolve(n.Children[pos], prefix)
				if err != nil {
					return false, nil, err
				}
				if cNode, ok := cNode.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cNode.Key...)
					return true, &shortNode{k, cNode.Val, t.newFlag()}, nil
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], t.newFlag()}, nil
		}
		// n still contains at least two values and cannot be reduced.
		return true, n, nil

	case valueNode:
		return true, nil, nil

	case nil:
		return false, nil, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.delete(rn, prefix, key)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key))
	}
}

func concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

func (t *Trie) resolve(n node, prefix []byte) (node, error) {
	if n, ok := n.(hashNode); ok {
		return t.resolveHash(n, prefix)
	}
	return n, nil
}
func (t *Trie) Commit(onLeaf LeafCallback) (root common.Hash, err error) {
	if t.db == nil {
		panic("commit called on trie with nil database")
	}
	hash, cached, err := t.hashRoot(t.db, onLeaf)
	if err != nil {
		return common.Hash{}, err
	}
	t.root = cached
	t.cacheGen++
	return common.BytesToHash(hash.(hashNode)), nil
}
