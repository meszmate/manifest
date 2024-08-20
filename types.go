package manifest

//go:generate stringer -type=EFeatureLevel
type EFeatureLevel int32

const (
	// The original version.
	EFeatureLevelOriginal EFeatureLevel = iota + 0
	// Support for custom fields.
	EFeatureLevelCustomFields
	// Started storing the version number.
	EFeatureLevelStartStoringVersion
	// Made after data files where renamed to include the hash value, these chunks now go to ChunksV2.
	EFeatureLevelDataFileRenames
	// Manifest stores whether build was constructed with chunk or file data.
	EFeatureLevelStoresIfChunkOrFileData
	// Manifest stores group number for each chunk/file data for reference so that external readers don't need to know how to calculate them.
	EFeatureLevelStoresDataGroupNumbers
	// Added support for chunk compression, these chunks now go to ChunksV3. NB: Not File Data Compression yet.
	EFeatureLevelChunkCompressionSupport
	// Manifest stores product prerequisites info.
	EFeatureLevelStoresPrerequisitesInfo
	// Manifest stores chunk download sizes.
	EFeatureLevelStoresChunkFileSizes
	// Manifest can optionally be stored using UObject serialization and compressed.
	EFeatureLevelStoredAsCompressedUClass
	// These two features were removed and never used.
	EFeatureLevelUNUSED_0
	EFeatureLevelUNUSED_1
	// Manifest stores chunk data SHA1 hash to use in place of data compare, for faster generation.
	EFeatureLevelStoresChunkDataShaHashes
	// Manifest stores Prerequisite Ids.
	EFeatureLevelStoresPrerequisiteIds
	// The first minimal binary format was added. UObject classes will no longer be saved out when binary selected.
	EFeatureLevelStoredAsBinaryData
	// Temporary level where manifest can reference chunks with dynamic window size, but did not serialize them. Chunks from here onwards are stored in ChunksV4.
	EFeatureLevelVariableSizeChunksWithoutWindowSizeChunkInfo
	// Manifest can reference chunks with dynamic window size, and also serializes them.
	EFeatureLevelVariableSizeChunks
	// Manifest stores a unique build id for exact matching of build data.
	EFeatureLevelStoresUniqueBuildId
	// !! Always after the latest version entry, signifies the latest version plus 1 to allow the following Latest alias.
	EFeatureLevelLatestPlusOne
	// An alias for the actual latest version value.
	EFeatureLevelLatest = (EFeatureLevelLatestPlusOne - 1)
	// An alias to provide the latest version of a manifest supported by file data (nochunks).
	LatestNoChunks = EFeatureLevelStoresChunkFileSizes
	// An alias to provide the latest version of a manifest supported by a json serialized format.
	LatestJson = EFeatureLevelStoresPrerequisiteIds
	// An alias to provide the first available version of optimised delta manifest saving.
	FirstOptimisedDelta = EFeatureLevelStoresUniqueBuildId
	// JSON manifests were stored with a version of 255 during a certain CL range due to a bug.
	// We will treat this as being StoresChunkFileSizes in code.
	EFeatureLevelBrokenJsonVersion = 255
	// This is for UObject default, so that we always serialize it.
	EFeatureLevelInvalid = -1
)

// ChunkSubDir returns the chunk version sub directory
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
	StoredCompressed uint8 = 0x01
	StoredEncrypted  uint8 = 0x02
)
