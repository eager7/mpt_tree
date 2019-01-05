package mpt

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"github.com/eager7/mpt_tree/store"
	"sync"
)

type Mpt struct {
	path    string
	trie    Trie
	db      Database
	levelDB *store.LevelDBStore
	lock    sync.RWMutex
}

func NewMptTree(path string, root common.Hash) (mpt *Mpt, err error) {
	mpt = new(Mpt)
	mpt.levelDB, err = store.NewLevelDBStore(path, 0, 0)
	if err != nil {
		return nil, err
	}
	mpt.db = NewDatabase(mpt.levelDB)
	fmt.Println("Open Trie Hash:", root.HexString())
	mpt.trie, err = mpt.db.OpenTrie(root)
	if err != nil {
		fmt.Println("open mpt failed:", err)
		if mpt.trie, err = mpt.db.OpenTrie(common.Hash{}); err != nil {
			return nil, err
		}
	}
	return mpt, nil
}

func (m *Mpt) Put(key, value []byte) error {
	return m.trie.TryUpdate(key, value)
}

func (m *Mpt) Get(key []byte) ([]byte, error) {
	return m.trie.TryGet(key)
}

func (m *Mpt) Del(key []byte) error {
	return m.trie.TryDelete(key)
}

func (m *Mpt) Commit() error {
	root, err := m.trie.Commit(nil)
	if err != nil {
		return err
	}
	return m.db.TrieDB().Commit(root, false)
}

func (m *Mpt) Close() error {
	return m.levelDB.Close()
}

func (m *Mpt) Hash() common.Hash {
	return m.trie.Hash()
}
