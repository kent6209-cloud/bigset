package datasource

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

const pmHeaderSize = 127

type pmHeader struct {
	Magic          [7]byte
	Version        uint8
	RootOffset     uint64
	RootLength     uint64
	MetaOffset     uint64
	MetaLength     uint64
	LeafOffset     uint64
	LeafLength     uint64
	TileDataOffset uint64
	TileDataLength uint64
	NumTiles       uint64
	NumEntries     uint64
	NumContents    uint64
	Clustered      uint8
	InternalComp   uint8
	TileComp       uint8
	TileType       uint8
	MinZoom        uint8
	MaxZoom        uint8
	MinLon         int32
	MinLat         int32
	MaxLon         int32
	MaxLat         int32
	CenterZoom     uint8
	CenterLon      int32
	CenterLat      int32
}

type pmDirEntry struct {
	TileID uint64
	Offset uint64
	Length uint32
}

type PMTilesSource struct {
	name   string
	crs    string
	path   string
	data   []byte
	hdr    pmHeader
	root   []pmDirEntry
	leaves []pmDirEntry
}

func NewPMTilesSource(name, path, crs string) (*PMTilesSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("PMTiles 讀取失敗: %w", err)
	}
	if len(data) < pmHeaderSize {
		return nil, fmt.Errorf("不是有效的 PMTiles 檔案 (大小不足)")
	}

	s := &PMTilesSource{name: name, crs: crs, path: path, data: data}
	if err := s.parseHeader(); err != nil {
		return nil, err
	}
	if err := s.parseRootDir(); err != nil {
		return nil, err
	}
	if s.hdr.LeafLength > 0 {
		if err := s.parseLeafDirs(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *PMTilesSource) parseHeader() error {
	d := s.data[:pmHeaderSize]
	copy(s.hdr.Magic[:], d[0:7])
	s.hdr.Version = d[7]
	if string(s.hdr.Magic[:]) != "PMTiles" {
		return fmt.Errorf("不是 PMTiles 檔案 (magic 錯誤)")
	}
	s.hdr.RootOffset = binary.LittleEndian.Uint64(d[8:16])
	s.hdr.RootLength = binary.LittleEndian.Uint64(d[16:24])
	s.hdr.MetaOffset = binary.LittleEndian.Uint64(d[24:32])
	s.hdr.MetaLength = binary.LittleEndian.Uint64(d[32:40])
	s.hdr.LeafOffset = binary.LittleEndian.Uint64(d[40:48])
	s.hdr.LeafLength = binary.LittleEndian.Uint64(d[48:56])
	s.hdr.TileDataOffset = binary.LittleEndian.Uint64(d[56:64])
	s.hdr.TileDataLength = binary.LittleEndian.Uint64(d[64:72])
	s.hdr.NumTiles = binary.LittleEndian.Uint64(d[72:80])
	s.hdr.NumEntries = binary.LittleEndian.Uint64(d[80:88])
	s.hdr.NumContents = binary.LittleEndian.Uint64(d[88:96])
	s.hdr.Clustered = d[96]
	s.hdr.InternalComp = d[97]
	s.hdr.TileComp = d[98]
	s.hdr.TileType = d[99]
	s.hdr.MinZoom = d[100]
	s.hdr.MaxZoom = d[101]
	s.hdr.MinLon = int32(binary.LittleEndian.Uint32(d[102:106]))
	s.hdr.MinLat = int32(binary.LittleEndian.Uint32(d[106:110]))
	s.hdr.MaxLon = int32(binary.LittleEndian.Uint32(d[110:114]))
	s.hdr.MaxLat = int32(binary.LittleEndian.Uint32(d[114:118]))
	s.hdr.CenterZoom = d[118]
	s.hdr.CenterLon = int32(binary.LittleEndian.Uint32(d[119:123]))
	s.hdr.CenterLat = int32(binary.LittleEndian.Uint32(d[123:127]))
	return nil
}

func (s *PMTilesSource) parseDir(offset, length uint64) ([]pmDirEntry, error) {
	if offset+length > uint64(len(s.data)) {
		return nil, fmt.Errorf("目錄區塊超出檔案範圍")
	}
	raw := s.data[offset : offset+length]
	num := len(raw) / 17
	entries := make([]pmDirEntry, num)
	for i := 0; i < num; i++ {
		base := i * 17
		entries[i].TileID = binary.LittleEndian.Uint64(raw[base : base+8])
		entries[i].Offset = binary.LittleEndian.Uint64(raw[base+8 : base+16])
		entries[i].Length = binary.LittleEndian.Uint32(raw[base+16 : base+20])
	}
	return entries, nil
}

func (s *PMTilesSource) parseRootDir() error {
	var err error
	s.root, err = s.parseDir(s.hdr.RootOffset, s.hdr.RootLength)
	return err
}

func (s *PMTilesSource) parseLeafDirs() error {
	var err error
	s.leaves, err = s.parseDir(s.hdr.LeafOffset, s.hdr.LeafLength)
	return err
}

func tileID(z, x, y int) uint64 {
	acc := uint64(0)
	for i := 0; i < z; i++ {
		bitX := (x >> i) & 1
		bitY := (y >> i) & 1
		acc |= uint64(bitX) << (2*uint(i) + 1)
		acc |= uint64(bitY) << (2 * uint(i))
	}
	return acc
}

func (s *PMTilesSource) searchDir(entries []pmDirEntry, id uint64) *pmDirEntry {
	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i].TileID > id
	})
	if idx == 0 {
		return nil
	}
	// Previous entry has TileID <= id
	prev := &entries[idx-1]
	if prev.Offset == 0 && prev.Length == 0 {
		return nil
	}

	// If prev.Offset points to leaf directory range, recurse into it
	if s.hdr.LeafOffset > 0 && prev.Offset >= s.hdr.LeafOffset && prev.Offset+uint64(prev.Length) <= s.hdr.LeafOffset+s.hdr.LeafLength {
		leafOff := prev.Offset
		leafLen := uint64(prev.Length)
		leafEntries, err := s.parseDir(leafOff, leafLen)
		if err != nil {
			return nil
		}
		leafIdx := sort.Search(len(leafEntries), func(i int) bool {
			return leafEntries[i].TileID > id
		})
		if leafIdx == 0 {
			return nil
		}
		leafPrev := &leafEntries[leafIdx-1]
		if leafPrev.Offset == 0 && leafPrev.Length == 0 {
			return nil
		}
		return leafPrev
	}

	return prev
}

func (s *PMTilesSource) Name() string { return s.name }
func (s *PMTilesSource) Type() string { return "pmtiles" }
func (s *PMTilesSource) CRS() string  { return s.crs }

func (s *PMTilesSource) GetTile(z, x, y int) ([]byte, string, error) {
	id := tileID(z, x, y)
	entry := s.searchDir(s.root, id)
	if entry == nil {
		return nil, "", fmt.Errorf("切片 (%d/%d/%d) 不存在", z, x, y)
	}
	if entry.Offset+uint64(entry.Length) > uint64(len(s.data)) {
		return nil, "", fmt.Errorf("切片資料超出範圍")
	}
	data := s.data[entry.Offset : entry.Offset+uint64(entry.Length)]

	format := "image/png"
	if len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8 {
		format = "image/jpeg"
	} else if len(data) > 2 && data[0] == 0x89 && data[1] == 0x50 {
		format = "image/png"
	} else if len(data) > 1 && data[0] == 0x47 && data[1] == 0x49 {
		format = "image/gif"
	} else if len(data) > 1 && data[0] == 0x42 && data[1] == 0x4D {
		format = "image/bmp"
	} else if len(data) > 3 && data[0] == 0x66 && data[1] == 0x6C && data[2] == 0x79 && data[3] == 0x49 {
		format = "image/webp"
	}

	return data, format, nil
}

func (s *PMTilesSource) GetFeatures(bbox BBox, targetCRS string) ([]Feature, error) {
	return nil, fmt.Errorf("PMTiles 不支援向量查詢")
}
