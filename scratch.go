// package scratch implements a scratch buffer for working with temporary byte slices.
package scratch

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode/utf8"
	"unsafe"
)

var (
	_ io.Writer       = (*Buf)(nil)
	_ io.StringWriter = (*Buf)(nil)
	_ io.ByteWriter   = (*Buf)(nil)
	_ io.Closer       = (*Buf)(nil)
)

// SizedMarshaler describes objects that can marshal themselves in a single allocation.
// The most common implementations are protobuf messages.
type SizedMarshaler interface {
	// Size returns the maximum number of bytes required to marshal the object.
	Size() int
	// MarshalToSizedBuffer appends the marshaled object to the pre-sized buffer buf.
	// It returns the number of bytes written and any error.
	MarshalToSizedBuffer(buf []byte) (int, error)
}

// DeterministicMarshaler describes objects that can marshal themselves deterministically.
// The most common implementations are protobuf messages.
type DeterministicMarshaler interface {
	// XXX_Size returns the maximum number of bytes required to marshal the object.
	XXX_Size() int
	// XXX_Marshal appends the marshaled object to the pre-sized buffer buf.
	// It returns the final buffer and any error.
	//
	// deterministic is true if deterministic marshaling is desired.
	XXX_Marshal(buf []byte, deterministic bool) ([]byte, error)
}

// Buf is a scratch buffer for working with temporary byte slices.
type Buf struct {
	s []byte
}

// Len returns the length of the buffer.
func (b *Buf) Len() int {
	return len(b.s)
}

// Cap returns the capacity of the buffer.
func (b *Buf) Cap() int {
	return cap(b.s)
}

// Bytes returns the buffered bytes as s[:len(s):len(s)].
// To access the full slice, use Scratch().
func (b *Buf) Bytes() []byte {
	return b.s[:len(b.s):len(b.s)]
}

// String returns a copy buffered bytes as a string.
func (b *Buf) String() string {
	return string(b.s)
}

// UnsafeString returns a *reference* to the underlying slice as a string.
//
// NOTE: the string should not be used again after calling other methods,
// of re-using the buffer, as it might change the contents of the string.
func (b *Buf) UnsafeString() string {
	return *(*string)(unsafe.Pointer(&b.s))
}

// Reader returns a new bytes.Reader over the underlying slice.
func (b *Buf) Reader() *bytes.Reader {
	return bytes.NewReader(b.s)
}

// Reset sets the buffer's length to 0 in preparation for re-use.
func (b *Buf) Reset() *Buf {
	b.s = b.s[:0]
	return b
}

// Grow ensures the buffer has enough capacity to fit n more bytes without re-allocation.
// Grow panics if n is negative.
func (b *Buf) Grow(n int) *Buf {
	if n < 0 {
		panic("scratch.Buf.Grow: negative count")
	}
	if n == 0 {
		return b
	}
	if b.Cap()-b.Len() >= n {
		return b
	}
	p := make([]byte, b.Len(), b.Len()+n)
	copy(p, b.s)
	b.s = p
	return b
}

// Scratch exposes the entire underlying slice to the function f.
// The underlying slice is replaced with the slice returned by f.
//
// It's useful as an escape hatch or to allow easy use of append() directly.
func (b *Buf) Scratch(f func([]byte) []byte) *Buf {
	b.s = f(b.s)
	return b
}

// Append appends s to buffer.
func (b *Buf) Append(s []byte) *Buf {
	b.s = append(b.s, s...)
	return b
}

// AppendString appends s to buffer.
func (b *Buf) AppendString(s string) *Buf {
	b.s = append(b.s, s...)
	return b
}

// AppendByte appends c to the buffer.
func (b *Buf) AppendByte(c byte) *Buf {
	b.s = append(b.s, c)
	return b
}

// AppendRune appends r to the buffer.
func (b *Buf) AppendRune(r rune) *Buf {
	b.appendRune(r)
	return b
}

// appendRune appends r to the buffer and returns its encoded length.
func (b *Buf) appendRune(r rune) int {
	if r < utf8.RuneSelf {
		b.AppendByte(byte(r))
		return 1
	}
	i := len(b.s)
	j := utf8.EncodeRune(b.Tail(utf8.UTFMax), r)
	b.s = b.s[:i+j]
	return j
}

// Write implements io.Writer.
// Write never returns an error.
func (b *Buf) Write(s []byte) (int, error) {
	b.Append(s)
	return len(s), nil
}

// WriteString implements io.StringWriter.
// WriteString never returns an error.
func (b *Buf) WriteString(s string) (int, error) {
	b.AppendString(s)
	return len(s), nil
}

// WriteByte implements io.ByteWriter.
// WriteByte never returns an error.
func (b *Buf) WriteByte(c byte) error {
	b.AppendByte(c)
	return nil
}

// WriteRune writes to the buffer and returns the encoded length of r.
// WriteRune never returns an error.
func (b *Buf) WriteRune(r rune) (int, error) {
	n := b.appendRune(r)
	return n, nil
}

// Close implements io.Closer as no-op.
// Close never returns an error.
func (b *Buf) Close() error {
	return nil
}

// Tail resizes the buffer to len()+n and returns a slice s[:len(s):len(s)] over the new space.
// See also PutUint64, etc.
func (b *Buf) Tail(n int) []byte {
	b.Grow(n)
	sp := b.Len()
	ep := sp + n
	b.s = b.s[:ep]
	return b.s[sp:ep]
}

// PutUint64 appends n to the buffer in big-endian order.
func (b *Buf) PutUint64(n uint64) *Buf {
	binary.BigEndian.PutUint64(b.Tail(8), n)
	return b
}

// PutUint32 appends n to the buffer in big-endian order.
func (b *Buf) PutUint32(n uint32) *Buf {
	binary.BigEndian.PutUint32(b.Tail(4), n)
	return b
}

// PutUint16 appends n to the buffer in big-endian order.
func (b *Buf) PutUint16(n uint16) *Buf {
	binary.BigEndian.PutUint16(b.Tail(2), n)
	return b
}

// Marshal appends the marshaled form of msg to the buffer.
// Buf.Bytes() is returned if marshaling succeeded.
// The most common implementations of SizedMarshaler are protobuf messages.
func (b *Buf) Marshal(msg SizedMarshaler) ([]byte, error) {
	i := b.Len()
	s := b.Tail(msg.Size())
	n, err := msg.MarshalToSizedBuffer(s)
	if err != nil {
		return nil, err
	}
	b.s = s[:i+n]
	return b.Bytes(), nil
}

// DeterministicallyMarshal appends the marshaled form of msg to the buffer.
// Buf.Bytes() is returned if marshaling succeeded.
// The most common implementations of DeterministicMarshaler are protobuf messages.
func (b *Buf) DeterministicallyMarshal(msg DeterministicMarshaler) ([]byte, error) {
	b.Grow(msg.XXX_Size())
	s, err := msg.XXX_Marshal(b.s, true)
	if err != nil {
		return nil, err
	}
	b.s = s
	return b.Bytes(), nil
}

// NewBuf returns a new buffer capable of holding cap bytes without re-allocation.
func NewBuf(cap int) *Buf {
	b := &Buf{}
	return b.Grow(cap)
}
