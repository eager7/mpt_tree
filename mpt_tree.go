package mpt

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"github.com/eager7/mpt_tree/store"
	"sync"
)

type Tree struct {
	path    string
	mpt     Trie
	db      Database
	levelDB *store.LevelDBStore
	lock    sync.RWMutex
}

func NewMptTree(path string, root common.Hash) (tree *Tree, err error) {
	tree = new(Tree)
	tree.levelDB, err = store.NewLevelDBStore(path, 0, 0)
	if err != nil {
		return nil, err
	}
	tree.db = NewDatabase(tree.levelDB)
	fmt.Println("Open Trie Hash:", root.HexString())
	tree.mpt, err = tree.db.OpenTrie(root)
	if err != nil {
		fmt.Println("open mpt failed:", err)
		if tree.mpt, err = tree.db.OpenTrie(common.Hash{}); err != nil {
			return nil, err
		}
	}
	return tree, nil
}

func (m *Tree) Set(key, value []byte) error {
	return m.mpt.TryUpdate(key, value)
}

func (m *Tree) Get(key []byte) ([]byte, error) {
	return m.mpt.TryGet(key)
}

func (m *Tree) Del(key []byte) error {
	return m.mpt.TryDelete(key)
}

func (m *Tree) Commit() error {
	root, err := m.mpt.Commit(nil)
	if err != nil {
		return err
	}
	return m.db.TrieDB().Commit(root, false)
}

func (m *Tree) Hash() common.Hash {
	return m.mpt.Hash()
}
