package testutils

import "testing"

func id(x int) int { return x }

func TestAssert(t *testing.T) {
	tt := T{T: t}

	tt.Run("the-test", func(t T) {
		t.Check(id(1) == 1, "id is not 1")
		t.CheckEqual(1, id(1))
		t.CheckDeepEqual(1, id(1))
		t.Assert(id(1) == 1)
		t.AssertEqual(1, id(1))
		t.AssertDeepEqual(1, id(1))
		t.CheckRegexpEqual("hello", "h.*o")
	})
}
