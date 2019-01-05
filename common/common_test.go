package common_test

import (
	"fmt"
	"github.com/eager7/mpt_tree/common"
	"testing"
)

func TestNewIndex(t *testing.T) {
	var char = []byte("abcdefghigklmnopqrstuvwxyz")
	for i := 0; i < 0; i ++ {
		name := common.NameToIndex(fmt.Sprintf("tester%c", char[i]))
		fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%8+1)
	}
	size := uint64(3)
	name := common.NameToIndex("root")
	fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%size+1)
	name = common.NameToIndex("testeru")
	fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%size+1)
	name = common.NameToIndex("testerh")
	fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%size+1)
	name = common.NameToIndex("testerl")
	fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%size+1)
	name = common.NameToIndex("testerp")
	fmt.Println(name, fmt.Sprintf("%d", name), "shard:", uint64(name)%999, uint64(name)%999%size+1)
}
