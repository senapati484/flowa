package vm

import (
	"testing"
)

func TestGlobalVariablesVM(t *testing.T) {
	tests := []vmTestCase{
		{"x = 1", 1},
		{"x = 1\nx", 1},
		{"x = 1\ny = 2\nx + y", 3},
	}

	runVmTests(t, tests)
}
