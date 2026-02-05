package manifest

// EFeatureLevel represents the feature level (version) of a manifest file.
// Higher levels indicate newer formats with additional capabilities.
//
//go:generate stringer -type=EFeatureLevel
type EFeatureLevel int32

const (
	// EFeatureLevelOriginal is the original version.
	EFeatureLevelOriginal EFeatureLevel = iota + 0
	// EFeatureLevelCustomFields adds support for custom fields.
	EFeatureLevelCustomFields
	// EFeatureLevelStartStoringVersion started storing the version number.
	EFeatureLevelStartStoringVersion
	// EFeatureLevelDataFileRenames was made after data files were renamed to include the hash value; these chunks now go to ChunksV2.
	EFeatureLevelDataFileRenames
	// EFeatureLevelStoresIfChunkOrFileData stores whether build was constructed with chunk or file data.
	EFeatureLevelStoresIfChunkOrFileData
	// EFeatureLevelStoresDataGroupNumbers stores group number for each chunk/file data for reference.
	EFeatureLevelStoresDataGroupNumbers
	// EFeatureLevelChunkCompressionSupport adds support for chunk compression; these chunks now go to ChunksV3.
	EFeatureLevelChunkCompressionSupport
	// EFeatureLevelStoresPrerequisitesInfo stores product prerequisites info.
	EFeatureLevelStoresPrerequisitesInfo
	// EFeatureLevelStoresChunkFileSizes stores chunk download sizes.
	EFeatureLevelStoresChunkFileSizes
	// EFeatureLevelStoredAsCompressedUClass allows manifests to optionally be stored using UObject serialization and compressed.
	EFeatureLevelStoredAsCompressedUClass
	// EFeatureLevelUNUSED_0 was removed and never used.
	EFeatureLevelUNUSED_0
	// EFeatureLevelUNUSED_1 was removed and never used.
	EFeatureLevelUNUSED_1
	// EFeatureLevelStoresChunkDataShaHashes stores chunk data SHA1 hash to use in place of data compare.
	EFeatureLevelStoresChunkDataShaHashes
	// EFeatureLevelStoresPrerequisiteIds stores prerequisite IDs.
	EFeatureLevelStoresPrerequisiteIds
	// EFeatureLevelStoredAsBinaryData is the first minimal binary format. UObject classes will no longer be saved out when binary selected.
	EFeatureLevelStoredAsBinaryData
	// EFeatureLevelVariableSizeChunksWithoutWindowSizeChunkInfo is a temporary level where manifest can reference chunks with dynamic window size. Chunks from here onwards are stored in ChunksV4.
	EFeatureLevelVariableSizeChunksWithoutWindowSizeChunkInfo
	// EFeatureLevelVariableSizeChunks allows manifests to reference chunks with dynamic window size and serialize them.
	EFeatureLevelVariableSizeChunks
	// EFeatureLevelStoresUniqueBuildId stores a unique build id for exact matching of build data.
	EFeatureLevelStoresUniqueBuildId
	// EFeatureLevelLatestPlusOne is always after the latest version entry.
	EFeatureLevelLatestPlusOne
	// EFeatureLevelLatest is an alias for the actual latest version value.
	EFeatureLevelLatest = (EFeatureLevelLatestPlusOne - 1)
	// LatestNoChunks is the latest version of a manifest supported by file data (nochunks).
	LatestNoChunks = EFeatureLevelStoresChunkFileSizes
	// LatestJson is the latest version of a manifest supported by a JSON serialized format.
	LatestJson = EFeatureLevelStoresPrerequisiteIds
	// FirstOptimisedDelta is the first available version of optimised delta manifest saving.
	FirstOptimisedDelta = EFeatureLevelStoresUniqueBuildId
	// EFeatureLevelBrokenJsonVersion is for JSON manifests stored with a version of 255 due to a bug.
	EFeatureLevelBrokenJsonVersion = 255
	// EFeatureLevelInvalid is for UObject default so that we always serialize it.
	EFeatureLevelInvalid = -1
)

// ChunkSubDir returns the chunk version sub directory based on the feature level.
//
// source: https://github.com/EpicGames/UnrealEngine/blob/d9d435c9c280b99a6c679b517adedd3f4b02cfd7/Engine/Source/Runtime/Online/BuildPatchServices/Private/Data/ManifestData.cpp#L77
func (e EFeatureLevel) ChunkSubDir() string {
	if e < EFeatureLevelDataFileRenames {
		return "Chunks"
	} else if e < EFeatureLevelChunkCompressionSupport {
		return "ChunksV2"
	} else if e < EFeatureLevelVariableSizeChunksWithoutWindowSizeChunkInfo {
		return "ChunksV3"
	}

	return "ChunksV4"
}

const (
	// StoredCompressed indicates the manifest body is zlib-compressed.
	StoredCompressed uint8 = 0x01
	// StoredEncrypted indicates the manifest body is encrypted.
	StoredEncrypted uint8 = 0x02
)
