package mpt_tree

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"github.com/eager7/mpt_tree/store"
	"sync"
)

type MptTree struct {
	path   string
	trie   Trie
	db     Database
	diskDb *store.LevelDBStore
	lock   sync.RWMutex
}

func NewMptTree(path string, root common.Hash) (tree *MptTree, err error) {
	tree = new(MptTree)
	tree.diskDb, err = store.NewLevelDBStore(path, 0, 0)
	if err != nil {
		return nil, err
	}
	tree.db = NewDatabase(tree.diskDb)
	fmt.Println("Open Trie Hash:", root.HexString())
	tree.trie, err = tree.db.OpenTrie(root)
	if err != nil {
		tree.trie, _ = tree.db.OpenTrie(common.Hash{})
	}
	return tree, nil
}
