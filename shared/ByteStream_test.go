package shared

import (
	"testing"
)

func TestByteStream(t *testing.T) {
	expect := NewExpect(t)

	stream := NewByteStream(1)
	expect.IntEq(1, stream.Cap())
	expect.IntEq(0, stream.Len())
	expect.IntEq(stream.Len(), len(stream.Bytes()))
	expect.IntEq(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("a"))
	expect.IntEq(1, stream.Cap())
	expect.IntEq(1, stream.Len())
	expect.IntEq(stream.Len(), len(stream.Bytes()))
	expect.IntEq(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("bc"))
	expect.StringEq("abc", string(stream.Bytes()))
	expect.IntEq(3, stream.Cap())
	expect.IntEq(3, stream.Len())
	expect.IntEq(stream.Len(), len(stream.Bytes()))
	expect.IntEq(stream.Cap(), cap(stream.Bytes()))

	stream.Reset()
	expect.StringEq("", string(stream.Bytes()))
	expect.IntEq(3, stream.Cap())
	expect.IntEq(0, stream.Len())
	expect.IntEq(stream.Len(), len(stream.Bytes()))
	expect.IntEq(stream.Cap(), cap(stream.Bytes()))

	stream.Write([]byte("bc"))
	expect.StringEq("bc", string(stream.Bytes()))
	expect.IntEq(3, stream.Cap())
	expect.IntEq(2, stream.Len())
	expect.IntEq(stream.Len(), len(stream.Bytes()))
	expect.IntEq(stream.Cap(), cap(stream.Bytes()))
}
