package grf

type FileEntry struct {
	CompressedSizeAligned uint32
	CompressedSize        uint32
	UncompressedSize      uint32
	Flags                 uint8
	Offset                uint32
	FileName              string
	DisplayName           string
}
