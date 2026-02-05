package manifest

import (
	"encoding/binary"
	"io"

	"github.com/meszmate/manifest/binreader"
)

// FCustomFields holds arbitrary key-value metadata attached to a manifest.
type FCustomFields struct {
	DataSize    uint32
	DataVersion uint8
	Count       uint32
	Fields      map[string]string
}

// ReadCustomFields reads the custom fields section from f.
func ReadCustomFields(f io.ReadSeeker) (*FCustomFields, error) {
	reader := binreader.NewReader(f, binary.LittleEndian)
	var fields FCustomFields
	var err error

	fields.DataSize, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	fields.DataVersion, err = reader.ReadUint8()
	if err != nil {
		return nil, err
	}

	fields.Count, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	fields.Fields = map[string]string{}

	// store the keys for the second iteration
	keys := make([]string, fields.Count)
	for idx := range keys {
		keys[idx], err = reader.ReadFString()
		if err != nil {
			return nil, err
		}
	}

	// map indices to keys and use them to build the map
	for idx := range keys {
		fields.Fields[keys[idx]], err = reader.ReadFString()
		if err != nil {
			return nil, err
		}
	}

	return &fields, nil
}
