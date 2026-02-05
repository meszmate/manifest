// Package binreader provides a binary reader that wraps an io.ReadSeeker
// and supports reading primitive types in a specified byte order.
package binreader

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/google/uuid"
)

// ErrNegativeAmount is returned when a negative byte count is passed to ReadBytes.
var ErrNegativeAmount = errors.New("negative bytes count")

// NewReader creates a new Reader that reads from r using the given byte order.
func NewReader(r io.ReadSeeker, order binary.ByteOrder) *Reader {
	return &Reader{
		r:     r,
		order: order,
	}
}

// Reader wraps an io.ReadSeeker and provides methods for reading binary data
// in a specified byte order.
type Reader struct {
	r     io.ReadSeeker
	order binary.ByteOrder
}

// ReadAll reads all remaining bytes from the underlying reader.
func (r *Reader) ReadAll() ([]byte, error) {
	return io.ReadAll(r.r)
}

// ReadBytes reads exactly count bytes from the reader.
// Returns the number of bytes read, the byte slice, and any error.
func (r *Reader) ReadBytes(count int) (n int, out []byte, err error) {
	if count < 0 {
		return 0, nil, ErrNegativeAmount
	}

	if count == 0 {
		return 0, []byte{}, nil
	}

	out = make([]byte, count)
	n, err = io.ReadFull(r.r, out)

	return
}

// ReadUint8 reads a single unsigned 8-bit integer.
func (r *Reader) ReadUint8() (uint8, error) {
	_, b, err := r.ReadBytes(1)
	if err != nil {
		return 0, err
	}

	return b[0], nil
}

// ReadBool reads a single byte and returns true if it is non-zero.
func (r *Reader) ReadBool() (bool, error) {
	b, err := r.ReadByte()
	return b != 0, err
}

// ReadByte reads a single byte. It is an alias for ReadUint8.
func (r *Reader) ReadByte() (byte, error) {
	return r.ReadUint8()
}

// ReadUint16 reads an unsigned 16-bit integer in the reader's byte order.
func (r *Reader) ReadUint16() (uint16, error) {
	_, b, err := r.ReadBytes(2)
	if err != nil {
		return 0, err
	}

	return r.order.Uint16(b), nil
}

// ReadUint32 reads an unsigned 32-bit integer in the reader's byte order.
func (r *Reader) ReadUint32() (uint32, error) {
	_, b, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}

	return r.order.Uint32(b), nil
}

// ReadUint64 reads an unsigned 64-bit integer in the reader's byte order.
func (r *Reader) ReadUint64() (uint64, error) {
	_, b, err := r.ReadBytes(8)
	if err != nil {
		return 0, err
	}

	return r.order.Uint64(b), nil
}

// ReadInt8 reads a signed 8-bit integer.
func (r *Reader) ReadInt8() (int8, error) {
	i, err := r.ReadUint8()
	return int8(i), err
}

// ReadInt16 reads a signed 16-bit integer in the reader's byte order.
func (r *Reader) ReadInt16() (int16, error) {
	i, err := r.ReadUint16()
	return int16(i), err
}

// ReadInt32 reads a signed 32-bit integer in the reader's byte order.
func (r *Reader) ReadInt32() (int32, error) {
	i, err := r.ReadUint32()
	return int32(i), err
}

// ReadInt64 reads a signed 64-bit integer in the reader's byte order.
func (r *Reader) ReadInt64() (int64, error) {
	i, err := r.ReadUint64()
	return int64(i), err
}

// ReadFloat32 reads a 32-bit IEEE 754 floating point number.
func (r *Reader) ReadFloat32() (float32, error) {
	b, err := r.ReadUint32()
	if err != nil {
		return 0, err
	}

	return math.Float32frombits(b), nil
}

// ReadFloat64 reads a 64-bit IEEE 754 floating point number.
func (r *Reader) ReadFloat64() (float64, error) {
	b, err := r.ReadUint64()
	if err != nil {
		return 0, err
	}

	return math.Float64frombits(b), nil
}

// Read implements the io.Reader interface.
func (r *Reader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

// Seek implements the io.Seeker interface.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

// Peek reads n bytes without advancing the reader position.
func (r *Reader) Peek(n int) ([]byte, error) {
	bytesRead, b, err := r.ReadBytes(n)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(-bytesRead), io.SeekCurrent)
	return b, err
}

// ReadFString reads an FString (a null-terminated string prefixed with its length) from the reader.
func (r *Reader) ReadFString() (string, error) {
	size, err := r.ReadUint32()
	if err != nil || size == 0 {
		return "", err
	}

	_, buf, err := r.ReadBytes(int(size))
	if err != nil {
		return "", err
	}
	if buf[len(buf)-1] != 0x0 {
		return "", errors.New("string is not null terminated")
	}
	return string(buf[:len(buf)-1]), nil // strip the null terminator
}

// ReadFStringArray reads an array of FStrings prefixed with the array length.
func (r *Reader) ReadFStringArray() (out []string, err error) {
	size, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}

	for i := uint32(0); i < size; i++ {
		fstr, err := r.ReadFString()
		if err != nil {
			return nil, err
		}
		out = append(out, fstr)
	}

	return
}

// ReadGUID reads a GUID stored as 4 uint32 segments in big endian byte order.
func (r *Reader) ReadGUID() (guid uuid.UUID, err error) {
	data := make([]uint32, 4)
	err = binary.Read(r, binary.BigEndian, &data)
	if err != nil {
		return uuid.Nil, err
	}
	for i, v := range data {
		binary.LittleEndian.PutUint32(guid[i*4:(i+1)*4], v)
	}
	return
}
