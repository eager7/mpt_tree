package mpt_test

import (
	"fmt"
	"github.com/eager7/mpt_tree"
	"github.com/eager7/mpt_tree/common"
	"os"
	"testing"
)

func TestNewMptTree(t *testing.T) {
	_ = os.RemoveAll("/tmp/tree")
	fmt.Println(mpt.NewMptTree("/tmp/tree", common.SingleHash([]byte("test"))))
}
