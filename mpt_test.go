package mpt_tree

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"os"
	"testing"
)

func TestNewMptTree(t *testing.T) {
	_ = os.RemoveAll("/tmp/tree")
	fmt.Println(NewMptTree("/tmp/tree", common.SingleHash([]byte("test"))))
}
