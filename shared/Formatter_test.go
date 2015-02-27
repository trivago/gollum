package shared

import "testing"

func TestItoLen(t *testing.T) {
	expect := NewExpect(t)

	expect.IntEq(1, ItoLen(0))
	expect.IntEq(1, ItoLen(1))
	expect.IntEq(2, ItoLen(10))
	expect.IntEq(2, ItoLen(33))
	expect.IntEq(3, ItoLen(100))
	expect.IntEq(3, ItoLen(555))
}

func TestItob(t *testing.T) {
	expect := NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itob(0, buffer[:])
	expect.StringEq("011", string(buffer[:]))

	Itob(123, buffer[:])
	expect.StringEq("123", string(buffer[:]))

	Itob(45, buffer[:])
	expect.StringEq("45", string(buffer[:2]))
}

func TestItobe(t *testing.T) {
	expect := NewExpect(t)
	buffer := [3]byte{'1', '1', '1'}

	Itobe(0, buffer[:])
	expect.StringEq("110", string(buffer[:]))

	Itobe(123, buffer[:])
	expect.StringEq("123", string(buffer[:]))

	Itobe(45, buffer[:])
	expect.StringEq("45", string(buffer[1:]))
}

func TestBtoi(t *testing.T) {
	expect := NewExpect(t)

	result, length := Btoi([]byte("0"))
	expect.IntEq(0, int(result))
	expect.IntEq(1, length)

	result, length = Btoi([]byte("test"))
	expect.IntEq(0, int(result))
	expect.IntEq(0, length)

	result, length = Btoi([]byte("10"))
	expect.IntEq(10, int(result))
	expect.IntEq(2, length)

	result, length = Btoi([]byte("10x"))
	expect.IntEq(10, int(result))
	expect.IntEq(2, length)

	result, length = Btoi([]byte("33"))
	expect.IntEq(33, int(result))
	expect.IntEq(2, length)

	result, length = Btoi([]byte("100"))
	expect.IntEq(100, int(result))
	expect.IntEq(3, length)

	result, length = Btoi([]byte("333"))
	expect.IntEq(333, int(result))
	expect.IntEq(3, length)
}
