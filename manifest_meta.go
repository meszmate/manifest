package manifest

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/meszmate/manifest/binreader"
)

type FManifestMeta struct {
	DataSize    uint32
	DataVersion uint8

	FeatureLevel  EFeatureLevel
	IsFileData    bool
	AppID         int32
	AppName       string
	BuildVersion  string
	LaunchExe     string
	LaunchCommand string
	PrereqIds     []string
	PrereqName    string
	PrereqPath    string
	PrereqArgs    string

	// if DataVersion >= 1
	BuildId string
}

func (m FManifestMeta) String() string {
	out := fmt.Sprintf(`Data size in file: %d bytes
Data version: %d
Feature Level: %s
Is file data: %v
App ID: %d
App Name: %s
Build Version: %s
Launch Exe: %s
Launch Command: %s
Prerequisite IDs: %v
Prerequisite Name: %s
Prerequisite Path: %s
Prerequisite Args: %s`, m.DataSize, m.DataVersion, m.FeatureLevel.String(),
		m.IsFileData, m.AppID, m.AppName, m.BuildVersion, m.LaunchExe,
		m.LaunchCommand, m.PrereqIds, m.PrereqName, m.PrereqPath, m.PrereqArgs)

	if m.DataVersion >= 1 {
		out += fmt.Sprintf(`Build ID: %s`, m.BuildId)
	}
	return out
}

func ReadFManifestMeta(f io.ReadSeeker) (*FManifestMeta, error) {
	reader := binreader.NewReader(f, binary.LittleEndian)
	var meta FManifestMeta
	var err error

	meta.DataSize, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	meta.DataVersion, err = reader.ReadUint8()
	if err != nil {
		return nil, err
	}

	featureLevel, err := reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	meta.FeatureLevel = EFeatureLevel(featureLevel)

	meta.IsFileData, err = reader.ReadBool()
	if err != nil {
		return nil, err
	}

	meta.AppID, err = reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	meta.AppName, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.BuildVersion, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.LaunchExe, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.LaunchCommand, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.PrereqIds, err = reader.ReadFStringArray()
	if err != nil {
		return nil, err
	}

	meta.PrereqName, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.PrereqPath, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	meta.PrereqArgs, err = reader.ReadFString()
	if err != nil {
		return nil, err
	}

	if meta.DataVersion >= 1 {
		meta.BuildId, err = reader.ReadFString()
		if err != nil {
			return nil, err
		}
	}

	return &meta, nil
}
