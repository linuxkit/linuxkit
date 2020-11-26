package fat32

import (
	"bytes"
	"io/ioutil"
	"sort"
	"testing"
)

const (
	eoc    = uint32(0x0fffffff) // {0xff, 0xff, 0xff, 0x0f})
	eocMin = uint32(0x0ffffff8) // {0xf8, 0xff, 0xff, 0x0f})
)

func getValidFat32Table() *table {
	sectorsPerFat := 158               // 158 sectors per FAT given in DOS20BPB
	sizeInBytes := sectorsPerFat * 512 // 512 bytes per sector,
	numClusters := sizeInBytes / 4
	// table as read from fat32.img using
	//    xxd -c 4 ./testdata/fat32.img
	// directory "\" is at cluster 2 (first data cluster) at byte 0x02b800 = 178176
	// directory "\foo" is at cluster 3 (second data cluster)
	return &table{
		fatID:     268435448, // 0x0ffffff8
		eocMarker: eoc,       // 0x0fffffff
		clusters: map[uint32]uint32{
			2:   eocMin,
			3:   60,
			4:   eoc,
			5:   6,
			6:   7,
			7:   8,
			8:   9,
			9:   10,
			10:  11,
			11:  12,
			12:  13,
			13:  14,
			14:  15,
			15:  16,
			16:  eoc,
			17:  eoc,
			18:  19,
			19:  20,
			20:  21,
			21:  22,
			22:  23,
			23:  24,
			24:  25,
			25:  26,
			26:  27,
			27:  28,
			28:  29,
			29:  30,
			30:  31,
			31:  eoc,
			32:  33,
			33:  34,
			34:  35,
			35:  36,
			36:  37,
			37:  38,
			38:  39,
			39:  40,
			40:  41,
			41:  42,
			42:  43,
			43:  44,
			44:  45,
			45:  eoc,
			46:  eoc,
			47:  eoc,
			48:  eoc,
			49:  eoc,
			50:  eoc,
			51:  eoc,
			52:  eoc,
			53:  eoc,
			54:  eoc,
			55:  eoc,
			56:  eoc,
			57:  eoc,
			58:  eoc,
			59:  eoc,
			60:  77,
			61:  eoc,
			62:  eoc,
			63:  eoc,
			64:  eoc,
			65:  eoc,
			66:  eoc,
			67:  eoc,
			68:  eoc,
			69:  eoc,
			70:  eoc,
			71:  eoc,
			72:  eoc,
			73:  eoc,
			74:  eoc,
			75:  eoc,
			76:  eoc,
			77:  94,
			78:  eoc,
			79:  eoc,
			80:  eoc,
			81:  eoc,
			82:  eoc,
			83:  eoc,
			84:  eoc,
			85:  eoc,
			86:  eoc,
			87:  eoc,
			88:  eoc,
			89:  eoc,
			90:  eoc,
			91:  eoc,
			92:  eoc,
			93:  eoc,
			94:  111,
			95:  eoc,
			96:  eoc,
			97:  eoc,
			98:  eoc,
			99:  eoc,
			100: eoc,
			101: eoc,
			102: eoc,
			103: eoc,
			104: eoc,
			105: eoc,
			106: eoc,
			107: eoc,
			108: eoc,
			109: eoc,
			110: eoc,
			111: eoc,
			112: eoc,
			113: eoc,
			114: eoc,
			115: eoc,
			116: eoc,
			117: eoc,
			118: eoc,
			119: eoc,
			120: eoc,
			121: eoc,
			122: eoc,
			123: eoc,
			124: eoc,
			125: eoc,
			126: eoc,
		},
		rootDirCluster: 2,
		size:           uint32(sizeInBytes),
		maxCluster:     uint32(numClusters),
	}
}

func TestFat32TableFromBytes(t *testing.T) {
	t.Run("valid FAT32 Table", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[16384 : 158*512+16384]
		table, err := tableFromBytes(b)
		if err != nil {
			t.Errorf("Return unexpected error: %v", err)
		}
		if table == nil {
			t.Fatalf("Returned FAT32 Table was nil unexpectedly")
		}
		valid := getValidFat32Table()
		if !table.equal(valid) {
			keys := make([]uint32, 0)
			for _, k := range valid.clusters {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			t.Log(keys)
			keys = make([]uint32, 0)
			for _, k := range table.clusters {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			t.Log(keys)
			t.Fatalf("Mismatched FAT32 Table")
		}
	})
}

func TestFat32TableToBytes(t *testing.T) {
	t.Run("valid FAT32 table", func(t *testing.T) {
		table := getValidFat32Table()
		b, err := table.bytes()
		if err != nil {
			t.Errorf("Error was not nil, instead %v", err)
		}
		if b == nil {
			t.Fatal("b was nil unexpectedly")
		}
		valid, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		validBytes := valid[16384 : 158*512+16384]
		if bytes.Compare(validBytes, b) != 0 {
			t.Error("Mismatched bytes")
		}
	})
}

func TestFat32TableIsEoc(t *testing.T) {
	tests := []struct {
		cluster uint32
		eoc     bool
	}{
		{0xa7, false},
		{0x00, false},
		{0xFFFFFF7, false},
		{0xFFFFFF8, true},
		{0xFFFFFF9, true},
		{0xFFFFFFa, true},
		{0xFFFFFFb, true},
		{0xFFFFFFc, true},
		{0xFFFFFFd, true},
		{0xFFFFFFe, true},
		{0xFFFFFFf, true},
		{0xaFFFFFFf, true},
		{0x2FFFFFF8, true},
	}
	tab := table{}
	for _, tt := range tests {
		eoc := tab.isEoc(tt.cluster)
		if eoc != tt.eoc {
			t.Errorf("isEoc(%x): actual %t instead of expected %t", tt.cluster, eoc, tt.eoc)
		}
	}
}
