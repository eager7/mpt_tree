package elog_test

import (
	"testing"
	"github.com/eager7/mpt_tree/common/elog"
)

func TestLogger(t *testing.T) {
	l := elog.NewLogger("", 0)
	l.Debug("debug------------------")
	l.Info("info----------------------")
	l.Warn("warn------------------")
	l.Error("error---------------------")
	//l.Panic("panic------------")
	l.ErrStack()
}
