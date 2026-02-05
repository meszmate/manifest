package binreader

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	"github.com/google/uuid"
)

func newLEReader(data []byte) *Reader {
	return NewReader(bytes.NewReader(data), binary.LittleEndian)
}

func newBEReader(data []byte) *Reader {
	return NewReader(bytes.NewReader(data), binary.BigEndian)
}

// --- ReadUint8 ---

func TestReadUint8(t *testing.T) {
	r := newLEReader([]byte{0x42})
	v, err := r.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0x42 {
		t.Errorf("got %d, want %d", v, 0x42)
	}
}

func TestReadUint8_EOF(t *testing.T) {
	r := newLEReader([]byte{})
	_, err := r.ReadUint8()
	if err == nil {
		t.Fatal("expected error on empty reader")
	}
}

// --- ReadUint16 ---

func TestReadUint16_LE(t *testing.T) {
	r := newLEReader([]byte{0x01, 0x02})
	v, err := r.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0x0201 {
		t.Errorf("got 0x%04X, want 0x0201", v)
	}
}

func TestReadUint16_BE(t *testing.T) {
	r := newBEReader([]byte{0x01, 0x02})
	v, err := r.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0x0102 {
		t.Errorf("got 0x%04X, want 0x0102", v)
	}
}

// --- ReadUint32 ---

func TestReadUint32(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 12345678)
	r := newLEReader(buf)
	v, err := r.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if v != 12345678 {
		t.Errorf("got %d, want %d", v, 12345678)
	}
}

// --- ReadUint64 ---

func TestReadUint64(t *testing.T) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, 0xDEADBEEFCAFEBABE)
	r := newLEReader(buf)
	v, err := r.ReadUint64()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0xDEADBEEFCAFEBABE {
		t.Errorf("got 0x%X, want 0xDEADBEEFCAFEBABE", v)
	}
}

// --- ReadInt8 ---

func TestReadInt8(t *testing.T) {
	r := newLEReader([]byte{0xFF}) // -1
	v, err := r.ReadInt8()
	if err != nil {
		t.Fatal(err)
	}
	if v != -1 {
		t.Errorf("got %d, want -1", v)
	}
}

// --- ReadInt16 ---

func TestReadInt16(t *testing.T) {
	var val int16 = -256
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(val))
	r := newLEReader(buf)
	v, err := r.ReadInt16()
	if err != nil {
		t.Fatal(err)
	}
	if v != -256 {
		t.Errorf("got %d, want -256", v)
	}
}

// --- ReadInt32 ---

func TestReadInt32(t *testing.T) {
	var val int32 = -100000
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	r := newLEReader(buf)
	v, err := r.ReadInt32()
	if err != nil {
		t.Fatal(err)
	}
	if v != -100000 {
		t.Errorf("got %d, want -100000", v)
	}
}

// --- ReadInt64 ---

func TestReadInt64(t *testing.T) {
	var val int64 = -9999999999
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	r := newLEReader(buf)
	v, err := r.ReadInt64()
	if err != nil {
		t.Fatal(err)
	}
	if v != -9999999999 {
		t.Errorf("got %d, want -9999999999", v)
	}
}

// --- ReadFloat32 ---

func TestReadFloat32(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(3.14))
	r := newLEReader(buf)
	v, err := r.ReadFloat32()
	if err != nil {
		t.Fatal(err)
	}
	if v != 3.14 {
		t.Errorf("got %f, want 3.14", v)
	}
}

// --- ReadFloat64 ---

func TestReadFloat64(t *testing.T) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(2.718281828))
	r := newLEReader(buf)
	v, err := r.ReadFloat64()
	if err != nil {
		t.Fatal(err)
	}
	if v != 2.718281828 {
		t.Errorf("got %f, want 2.718281828", v)
	}
}

// --- ReadBool ---

func TestReadBool(t *testing.T) {
	tests := []struct {
		input byte
		want  bool
	}{
		{0x00, false},
		{0x01, true},
		{0xFF, true},
	}
	for _, tt := range tests {
		r := newLEReader([]byte{tt.input})
		v, err := r.ReadBool()
		if err != nil {
			t.Fatal(err)
		}
		if v != tt.want {
			t.Errorf("input 0x%02X: got %v, want %v", tt.input, v, tt.want)
		}
	}
}

// --- ReadBytes ---

func TestReadBytes_Normal(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := newLEReader(data)
	n, out, err := r.ReadBytes(3)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("n = %d, want 3", n)
	}
	if !bytes.Equal(out, []byte{1, 2, 3}) {
		t.Errorf("got %v, want [1 2 3]", out)
	}
}

func TestReadBytes_Zero(t *testing.T) {
	r := newLEReader([]byte{1, 2, 3})
	n, out, err := r.ReadBytes(0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || len(out) != 0 {
		t.Errorf("expected empty result, got n=%d len=%d", n, len(out))
	}
}

func TestReadBytes_Negative(t *testing.T) {
	r := newLEReader([]byte{1, 2, 3})
	_, _, err := r.ReadBytes(-1)
	if err != ErrNegativeAmount {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestReadBytes_EOF(t *testing.T) {
	r := newLEReader([]byte{1})
	_, _, err := r.ReadBytes(5)
	if err == nil {
		t.Fatal("expected error on insufficient data")
	}
}

// --- ReadFString ---

func TestReadFString_Normal(t *testing.T) {
	// length=6, "hello\0"
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(6))
	buf.WriteString("hello\x00")

	r := newLEReader(buf.Bytes())
	s, err := r.ReadFString()
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("got %q, want %q", s, "hello")
	}
}

func TestReadFString_Empty(t *testing.T) {
	// length=0
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0)
	r := newLEReader(buf)
	s, err := r.ReadFString()
	if err != nil {
		t.Fatal(err)
	}
	if s != "" {
		t.Errorf("got %q, want empty string", s)
	}
}

func TestReadFString_NotNullTerminated(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	buf.WriteString("abc") // no null terminator

	r := newLEReader(buf.Bytes())
	_, err := r.ReadFString()
	if err == nil {
		t.Fatal("expected error for non-null-terminated string")
	}
}

// --- ReadFStringArray ---

func TestReadFStringArray(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // 2 strings

	// "ab\0" (length 3)
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	buf.WriteString("ab\x00")

	// "cd\0" (length 3)
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	buf.WriteString("cd\x00")

	r := newLEReader(buf.Bytes())
	arr, err := r.ReadFStringArray()
	if err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 || arr[0] != "ab" || arr[1] != "cd" {
		t.Errorf("got %v, want [ab cd]", arr)
	}
}

// --- ReadGUID ---

func TestReadGUID(t *testing.T) {
	// Write 4 uint32s in big endian
	var buf bytes.Buffer
	vals := []uint32{0x01020304, 0x05060708, 0x090A0B0C, 0x0D0E0F10}
	for _, v := range vals {
		binary.Write(&buf, binary.BigEndian, v)
	}

	r := newLEReader(buf.Bytes())
	guid, err := r.ReadGUID()
	if err != nil {
		t.Fatal(err)
	}
	if guid == uuid.Nil {
		t.Error("expected non-nil GUID")
	}
	// Verify first 4 bytes: the uint32 0x01020304 stored in LE → [04,03,02,01]
	if guid[0] != 0x04 || guid[1] != 0x03 || guid[2] != 0x02 || guid[3] != 0x01 {
		t.Errorf("first segment unexpected: %x", guid[:4])
	}
}

// --- Peek ---

func TestPeek(t *testing.T) {
	r := newLEReader([]byte{0xAA, 0xBB, 0xCC})
	peeked, err := r.Peek(2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(peeked, []byte{0xAA, 0xBB}) {
		t.Errorf("peeked %v, want [0xAA 0xBB]", peeked)
	}
	// Verify position hasn't advanced
	v, err := r.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0xAA {
		t.Errorf("after peek, got 0x%02X, want 0xAA", v)
	}
}

// --- ReadAll ---

func TestReadAll(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := newLEReader(data)
	out, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("got %v, want %v", out, data)
	}
}

// --- Seek ---

func TestSeek(t *testing.T) {
	r := newLEReader([]byte{0x10, 0x20, 0x30, 0x40})
	_, err := r.Seek(2, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	v, err := r.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if v != 0x30 {
		t.Errorf("got 0x%02X, want 0x30", v)
	}
}

// --- Sequential reads ---

func TestSequentialReads(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(42))
	binary.Write(&buf, binary.LittleEndian, uint16(7))
	buf.WriteByte(0xFF)

	r := newLEReader(buf.Bytes())

	u32, err := r.ReadUint32()
	if err != nil {
		t.Fatal(err)
	}
	if u32 != 42 {
		t.Errorf("uint32: got %d, want 42", u32)
	}

	u16, err := r.ReadUint16()
	if err != nil {
		t.Fatal(err)
	}
	if u16 != 7 {
		t.Errorf("uint16: got %d, want 7", u16)
	}

	u8, err := r.ReadUint8()
	if err != nil {
		t.Fatal(err)
	}
	if u8 != 0xFF {
		t.Errorf("uint8: got 0x%02X, want 0xFF", u8)
	}
}
