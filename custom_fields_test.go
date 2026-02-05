package manifest

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadCustomFields(t *testing.T) {
	var body bytes.Buffer

	// DataVersion
	body.WriteByte(0)

	// Count = 2
	binary.Write(&body, binary.LittleEndian, uint32(2))

	// Keys
	writeFString(&body, "key_a")
	writeFString(&body, "key_b")

	// Values
	writeFString(&body, "val_a")
	writeFString(&body, "val_b")

	// DataSize
	dataSize := uint32(4 + 1 + body.Len())
	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())

	fields, err := ReadCustomFields(bytes.NewReader(section.Bytes()))
	if err != nil {
		t.Fatalf("ReadCustomFields failed: %v", err)
	}

	if fields.Count != 2 {
		t.Errorf("Count = %d, want 2", fields.Count)
	}
	if fields.Fields["key_a"] != "val_a" {
		t.Errorf("key_a = %q, want %q", fields.Fields["key_a"], "val_a")
	}
	if fields.Fields["key_b"] != "val_b" {
		t.Errorf("key_b = %q, want %q", fields.Fields["key_b"], "val_b")
	}
}

func TestReadCustomFields_Empty(t *testing.T) {
	var body bytes.Buffer
	body.WriteByte(0)                                    // DataVersion
	binary.Write(&body, binary.LittleEndian, uint32(0)) // Count

	dataSize := uint32(4 + 1 + body.Len())
	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())

	fields, err := ReadCustomFields(bytes.NewReader(section.Bytes()))
	if err != nil {
		t.Fatalf("ReadCustomFields failed: %v", err)
	}
	if len(fields.Fields) != 0 {
		t.Errorf("expected empty fields, got %d", len(fields.Fields))
	}
}
