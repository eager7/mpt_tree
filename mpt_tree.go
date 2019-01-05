package mpt_tree

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"github.com/eager7/mpt_tree/store"
	"sync"
)

type MptTree struct {
	path    string
	mpt     Trie
	db      Database
	levelDB *store.LevelDBStore
	lock    sync.RWMutex
}

func NewMptTree(path string, root common.Hash) (tree *MptTree, err error) {
	tree = new(MptTree)
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

func (m *MptTree) Set(key, value []byte) error {
	return m.mpt.TryUpdate(key, value)
}

func (m *MptTree) Get(key []byte) ([]byte, error) {
	return m.mpt.TryGet(key)
}

func (m *MptTree) Del(key []byte) error {
	return m.mpt.TryDelete(key)
}

func (m *MptTree) Commit() error {
	root, err := m.mpt.Commit(nil)
	if err != nil {
		return err
	}
	return m.db.TrieDB().Commit(root, false)
}

func (m *MptTree) Hash() common.Hash {
	return m.mpt.Hash()
}
