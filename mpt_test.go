package mpt_tree

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"testing"
)

func TestNewMptTree(t *testing.T) {
	fmt.Println(NewMptTree("/tmp/tree", common.Hash{}))
}
