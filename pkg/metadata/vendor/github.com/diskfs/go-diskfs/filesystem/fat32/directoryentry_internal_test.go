package fat32

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

var (
	timeDateTimeTests = []struct {
		date uint16
		time uint16
		rfc  string
	}{
		// see reference at https://en.wikipedia.org/wiki/Design_of_the_FAT_file_system#DIR_OFS_10h and https://en.wikipedia.org/wiki/Design_of_the_FAT_file_system#DIR_OFS_0Eh
		{0x0022, 0x7472, "1980-01-02T14:35:36Z"}, // date: 0b0000000 0001 00010 / 0x0022 ; time: 0b01110 100011 10010 / 0x7472
		{0x1f79, 0x0203, "1995-11-25T00:16:07Z"}, // date: 0b0001111 1011 11001 / 0x1f79 ; time: 0b00000 010000 00011 / 0x0203
		{0xf2de, 0x6000, "2101-06-30T12:00:00Z"}, // date: 0b1111001 0110 11110 / 0xf2de ; time: 0b01100 000000 00000 / 0x6000
	}

	unarcBytes = []byte{
		0x43, 0x6f, 0x00, 0x2e, 0x00, 0x64, 0x00, 0x61, 0x00, 0x74, 0x00, 0x0f, 0x00, 0xb3, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
		0x02, 0x6e, 0x00, 0x20, 0x00, 0x6e, 0x00, 0x6f, 0x00, 0x6d, 0x00, 0x0f, 0x00, 0xb3, 0x62, 0x00, 0x72, 0x00, 0x65, 0x00, 0x20, 0x00, 0x6c, 0x00, 0x61, 0x00, 0x00, 0x00, 0x72, 0x00, 0x67, 0x00,
		0x01, 0x55, 0x00, 0x6e, 0x00, 0x20, 0x00, 0x61, 0x00, 0x72, 0x00, 0x0f, 0x00, 0xb3, 0x63, 0x00, 0x68, 0x00, 0x69, 0x00, 0x76, 0x00, 0x6f, 0x00, 0x20, 0x00, 0x00, 0x00, 0x63, 0x00, 0x6f, 0x00,
	}

	lfnBytesTests = []struct {
		lfn string
		err error
		b   []byte
	}{
		// first 2 are too short and too long - rest are normal
		{"", fmt.Errorf("longFilenameEntryFromBytes only can parse byte of length 32"), []byte{0x43, 0x6f, 0x00, 0x2e, 0x00, 0x64, 0x00, 0x61, 0x00, 0x74, 0x00, 0x0f, 0x00, 0xb3, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff}},
		{"", fmt.Errorf("longFilenameEntryFromBytes only can parse byte of length 32"), []byte{0x43, 0x6f, 0x00, 0x2e, 0x00, 0x64, 0x00, 0x61, 0x00, 0x74, 0x00, 0x0f, 0x00, 0xb3, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00}},
		// normal are taken from ./testdata/README.md
		{"o.dat", nil, unarcBytes[0:32]},
		{"n nombre larg", nil, unarcBytes[32:64]},
		{"Un archivo co", nil, unarcBytes[64:96]},
		{"o", nil, []byte{0x42, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0f, 0x00, 0x59, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}},
		{"tercer_archiv", nil, []byte{0x01, 0x74, 0x00, 0x65, 0x00, 0x72, 0x00, 0x63, 0x00, 0x65, 0x00, 0x0f, 0x00, 0x59, 0x72, 0x00, 0x5f, 0x00, 0x61, 0x00, 0x72, 0x00, 0x63, 0x00, 0x68, 0x00, 0x00, 0x00, 0x69, 0x00, 0x76, 0x00}},
		// this one adds some unicode
		{"edded_nameא", nil, []byte{0x42, 0x65, 0x00, 0x64, 0x00, 0x64, 0x00, 0x65, 0x00, 0x64, 0x00, 0x0f, 0x00, 0x60, 0x5f, 0x00, 0x6e, 0x00, 0x61, 0x00, 0x6d, 0x00, 0x65, 0x00, 0xd0, 0x05, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}},
		{"some_long_emb", nil, []byte{0x01, 0x73, 0x00, 0x6f, 0x00, 0x6d, 0x00, 0x65, 0x00, 0x5f, 0x00, 0x0f, 0x00, 0x60, 0x6c, 0x00, 0x6f, 0x00, 0x6e, 0x00, 0x67, 0x00, 0x5f, 0x00, 0x65, 0x00, 0x00, 0x00, 0x6d, 0x00, 0x62, 0x00}},
	}

	sfnBytesTests = []struct {
		shortName string
		extension string
		lfn       string
		b         []byte
		err       error
	}{
		// first several tests use invalid shortname char or too long
		{"foo", "TXT", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Invalid shortname character in filename")},
		{"א", "TXT", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Invalid shortname character in filename")},
		{"abcdefghuk", "TXT", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Invalid shortname character in filename")},
		{"FOO", "א", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Invalid shortname character in extension")},
		{"FOO", "TXT234", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Extension for file is longer")},
		{"FOO", "txt", "very long filename indeed", nil, fmt.Errorf("Could not calculate checksum for 8.3 filename: Invalid shortname character in extension")},
		// rest are valid
		{"UNARCH~1", "DAT", "Un archivo con nombre largo.dat", unarcBytes, nil},
	}
)

func compareDirectoryEntriesIgnoreDates(a, b *directoryEntry) bool {
	now := time.Now()
	// copy values so we do not mess up the originals
	c := &directoryEntry{}
	d := &directoryEntry{}
	*c = *a
	*d = *b

	// unify fields we let be equal
	c.createTime = now
	d.createTime = now
	c.accessTime = now
	d.accessTime = now
	c.modifyTime = now
	d.modifyTime = now

	return *c == *d
}

func getValidDirectoryEntries() ([]*directoryEntry, [][]byte, error) {
	// these are taken from the file ./testdata/fat32.img, see ./testdata/README.md
	t1, _ := time.Parse(time.RFC3339, "2017-11-26T07:53:16Z")
	t10, _ := time.Parse(time.RFC3339, "2017-11-26T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2017-11-26T08:01:02Z")
	t20, _ := time.Parse(time.RFC3339, "2017-11-26T00:00:00Z")
	t3, _ := time.Parse(time.RFC3339, "2017-11-26T08:01:38Z")
	t30, _ := time.Parse(time.RFC3339, "2017-11-26T00:00:00Z")
	t4, _ := time.Parse(time.RFC3339, "2017-11-26T08:01:44Z")
	t40, _ := time.Parse(time.RFC3339, "2017-11-26T00:00:00Z")
	entries := []*directoryEntry{
		&directoryEntry{
			filenameShort:      "FOO",
			fileExtension:      "",
			filenameLong:       "",
			isReadOnly:         false,
			isHidden:           false,
			isSystem:           false,
			isVolumeLabel:      false,
			isSubdirectory:     true,
			isArchiveDirty:     false,
			isDevice:           false,
			lowercaseShortname: true,
			createTime:         t1,
			modifyTime:         t1,
			accessTime:         t10,
			acccessRights:      accessRightsUnlimited,
			clusterLocation:    3,
			fileSize:           0,
			//	start:             uint32,
			longFilenameSlots: 0,
			isNew:             false,
		},
		&directoryEntry{
			filenameShort:   "TERCER~1",
			fileExtension:   "",
			filenameLong:    "tercer_archivo",
			isReadOnly:      false,
			isHidden:        false,
			isSystem:        false,
			isVolumeLabel:   false,
			isSubdirectory:  false,
			isArchiveDirty:  true,
			isDevice:        false,
			createTime:      t2,
			modifyTime:      t2,
			accessTime:      t20,
			acccessRights:   accessRightsUnlimited,
			clusterLocation: 5,
			fileSize:        6144,
			//	start:             uint32,
			longFilenameSlots: 2,
			isNew:             false,
		},
		&directoryEntry{
			filenameShort:   "CORTO1",
			fileExtension:   "TXT",
			filenameLong:    "",
			isReadOnly:      false,
			isHidden:        false,
			isSystem:        false,
			isVolumeLabel:   false,
			isSubdirectory:  false,
			isArchiveDirty:  true,
			isDevice:        false,
			createTime:      t3,
			modifyTime:      t3,
			accessTime:      t30,
			acccessRights:   accessRightsUnlimited,
			clusterLocation: 17,
			fileSize:        25,
			//	start:             uint32,
			longFilenameSlots: 0,
			isNew:             false,
		},
		&directoryEntry{
			filenameShort:   "UNARCH~1",
			fileExtension:   "DAT",
			filenameLong:    "Un archivo con nombre largo.dat",
			isReadOnly:      false,
			isHidden:        false,
			isSystem:        false,
			isVolumeLabel:   false,
			isSubdirectory:  false,
			isArchiveDirty:  true,
			isDevice:        false,
			createTime:      t4,
			modifyTime:      t4,
			accessTime:      t40,
			acccessRights:   accessRightsUnlimited,
			clusterLocation: 18,
			fileSize:        7168,
			//	start:             uint32,
			longFilenameSlots: 3,
			isNew:             false,
		},
	}

	// read correct bytes off of disk
	input, err := ioutil.ReadFile(Fat32File)
	if err != nil {
		return nil, nil, fmt.Errorf("Error reading data from fat32 test fixture %s: %v", Fat32File, err)
	}

	// start of root directory in fat32.img - sector 348
	// sector 0 - boot sector ; indicates: 32 reserved sectors (0-31); 158 sectors per fat
	// sector 1 - FS Information Sector
	// sector 2-31 - more reserved sectors
	// sector 32-189 -  FAT1
	// sector 190-347 - FAT2
	// sector 348 - start of root directory
	//    348*512 = 178176
	start := 178176 // 0x0002b800 - start of root directory in fat32.img

	// we only have 9 actual 32-byte entries, of which 4 are real and 3 are VFAT extensionBytes
	//   the rest are all 0s (as they should be), so we will include to exercise it
	b := make([][]byte, 8, 8)
	//
	b[0] = input[start : start+32]
	b[1] = input[start+32 : start+4*32]
	b[2] = input[start+4*32 : start+5*32]
	b[3] = input[start+5*32 : start+9*32]
	b[4] = input[start+9*32 : start+10*32]
	b[5] = input[start+10*32 : start+11*32]
	b[6] = input[start+11*32 : start+12*32]
	// how many zeroes will come from cluster?
	remainder := 2048 - (12 * 32)
	b[7] = make([]byte, remainder, remainder)
	return entries, b, nil
}

func getValidDirectoryEntriesExtended() ([]*directoryEntry, [][]byte, error) {
	// these are taken from the file ./testdata/fat32.img, see ./testdata/README.md
	t1, _ := time.Parse(time.RFC3339, "2017-11-26T07:53:16Z")
	t10, _ := time.Parse(time.RFC3339, "2017-11-26T00:00:00Z")
	entries := []*directoryEntry{
		// FilenameShort, FileExtension,FilenameLong,IsReadOnly,IsHidden,IsSystem,IsVolumeLabel,IsSubdirectory,IsArchiveDirty,IsDevice,LowercaseShortname,LowercaseExtension,CreateTime,ModifyTime,AccessTime,AcccessRights,clusterLocation,FileSize,filesystem,start,longFilenameSlots,isNew,
		{".", "", "", false, false, false, false, true, false, false, false, false, t1, t1, t10, accessRightsUnlimited, 3, 0, nil, 0, false},
		{"..", "", "", false, false, false, false, true, false, false, false, false, t1, t1, t10, accessRightsUnlimited, 0, 0, nil, 0, false},
		{"BAR", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 4, 0, nil, 0, false},
		{"DIR", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x2e, 0, nil, 0, false},

		{"DIR0", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x2f, 0, nil, 0, false},
		{"DIR1", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x30, 0, nil, 0, false},
		{"DIR2", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x31, 0, nil, 0, false},
		{"DIR3", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x32, 0, nil, 0, false},
		{"DIR4", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x33, 0, nil, 0, false},
		{"DIR5", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x34, 0, nil, 0, false},
		{"DIR6", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x35, 0, nil, 0, false},
		{"DIR7", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x36, 0, nil, 0, false},
		{"DIR8", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x37, 0, nil, 0, false},
		{"DIR9", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x38, 0, nil, 0, false},
		{"DIR10", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x39, 0, nil, 0, false},
		{"DIR11", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x3a, 0, nil, 0, false},

		{"DIR12", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x3b, 0, nil, 0, false},
		{"DIR13", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x3d, 0, nil, 0, false},
		{"DIR14", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x3e, 0, nil, 0, false},
		{"DIR15", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x3f, 0, nil, 0, false},
		{"DIR16", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x40, 0, nil, 0, false},
		{"DIR17", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x41, 0, nil, 0, false},
		{"DIR18", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x42, 0, nil, 0, false},
		{"DIR19", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x43, 0, nil, 0, false},
		{"DIR20", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x44, 0, nil, 0, false},
		{"DIR21", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x45, 0, nil, 0, false},
		{"DIR22", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x46, 0, nil, 0, false},
		{"DIR23", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x47, 0, nil, 0, false},
		{"DIR24", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x48, 0, nil, 0, false},
		{"DIR25", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x49, 0, nil, 0, false},
		{"DIR26", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x4a, 0, nil, 0, false},
		{"DIR27", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x4b, 0, nil, 0, false},

		{"DIR28", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x4c, 0, nil, 0, false},
		{"DIR29", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x4e, 0, nil, 0, false},
		{"DIR30", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x4f, 0, nil, 0, false},
		{"DIR31", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x50, 0, nil, 0, false},
		{"DIR32", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x51, 0, nil, 0, false},
		{"DIR33", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x52, 0, nil, 0, false},
		{"DIR34", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x53, 0, nil, 0, false},
		{"DIR35", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x54, 0, nil, 0, false},
		{"DIR36", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x55, 0, nil, 0, false},
		{"DIR37", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x56, 0, nil, 0, false},
		{"DIR38", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x57, 0, nil, 0, false},
		{"DIR39", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x58, 0, nil, 0, false},
		{"DIR40", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x59, 0, nil, 0, false},
		{"DIR41", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x5a, 0, nil, 0, false},
		{"DIR42", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x5b, 0, nil, 0, false},
		{"DIR43", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x5c, 0, nil, 0, false},

		{"DIR44", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x5d, 0, nil, 0, false},
		{"DIR45", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x5f, 0, nil, 0, false},
		{"DIR46", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x60, 0, nil, 0, false},
		{"DIR47", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x61, 0, nil, 0, false},
		{"DIR48", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x62, 0, nil, 0, false},
		{"DIR49", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x63, 0, nil, 0, false},
		{"DIR50", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x64, 0, nil, 0, false},
		{"DIR51", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x65, 0, nil, 0, false},
		{"DIR52", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x66, 0, nil, 0, false},
		{"DIR53", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x67, 0, nil, 0, false},
		{"DIR54", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x68, 0, nil, 0, false},
		{"DIR55", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x69, 0, nil, 0, false},
		{"DIR56", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x6a, 0, nil, 0, false},
		{"DIR57", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x6b, 0, nil, 0, false},
		{"DIR58", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x6c, 0, nil, 0, false},
		{"DIR59", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x6d, 0, nil, 0, false},

		{"DIR60", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x6e, 0, nil, 0, false},
		{"DIR61", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x70, 0, nil, 0, false},
		{"DIR62", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x71, 0, nil, 0, false},
		{"DIR63", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x72, 0, nil, 0, false},
		{"DIR64", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x73, 0, nil, 0, false},
		{"DIR65", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x74, 0, nil, 0, false},
		{"DIR66", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x75, 0, nil, 0, false},
		{"DIR67", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x76, 0, nil, 0, false},
		{"DIR68", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x77, 0, nil, 0, false},
		{"DIR69", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x78, 0, nil, 0, false},
		{"DIR70", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x79, 0, nil, 0, false},
		{"DIR71", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x7a, 0, nil, 0, false},
		{"DIR72", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x7b, 0, nil, 0, false},
		{"DIR73", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x7c, 0, nil, 0, false},
		{"DIR74", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x7d, 0, nil, 0, false},
		{"DIR75", "", "", false, false, false, false, true, false, false, true, false, t1, t1, t10, accessRightsUnlimited, 0x7e, 0, nil, 0, false},
	}

	// read correct bytes off of disk
	input, err := ioutil.ReadFile(Fat32File)
	if err != nil {
		return nil, nil, fmt.Errorf("Error reading data from fat32 test fixture %s: %v", Fat32File, err)
	}

	// start of foo directory in fat32.img - cluster 3 = sector 349 = bytes 349*512 = 178688 = 0x0002ba00
	start := 178688

	b := make([][]byte, len(entries), len(entries))
	for i := 0; i < len(entries); i++ {
		b[i] = input[start+i*32 : start+i*32+32]
	}
	return entries, b, nil
}

func TestDirectoryEntryLongFilenameBytes(t *testing.T) {
	for _, tt := range sfnBytesTests {
		output, err := longFilenameBytes(tt.lfn, tt.shortName, tt.extension)
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Log(err)
			t.Log(tt.err)
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
		if bytes.Compare(output, tt.b) != 0 {
			t.Errorf("longFilenameBytes(%s, %s, %s) bytes mismatch", tt.lfn, tt.shortName, tt.extension)
			t.Log(fmt.Sprintf("actual  : % x", output))
			t.Log(fmt.Sprintf("expected: % x", tt.b))
		}
	}

}

func TestDirectoryEntryLongFilenameEntryFromBytes(t *testing.T) {
	for i, tt := range lfnBytesTests {
		output, err := longFilenameEntryFromBytes(tt.b)
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
		if output != tt.lfn {
			t.Errorf("%d: longFilenameEntryFromBytes() returned %s instead of %s from %v", i, output, tt.lfn, tt.b)
		}
	}
}

func TestDateTimeToTime(t *testing.T) {
	for _, tt := range timeDateTimeTests {
		output := dateTimeToTime(tt.date, tt.time)
		expected, err := time.Parse(time.RFC3339, tt.rfc)
		if err != nil {
			t.Fatalf("Error parsing expected date: %v", err)
		}
		// handle odd error case
		if expected.Second()%2 != 0 {
			expected = expected.Add(-1 * time.Second)
		}
		if expected != output {
			t.Errorf("dateTimeToTime(%d, %d) expected output %v, actual %v", tt.date, tt.time, expected, output)
		}
	}
}

func TestTimeToDateTime(t *testing.T) {
	for _, tt := range timeDateTimeTests {
		input, err := time.Parse(time.RFC3339, tt.rfc)
		if err != nil {
			t.Fatalf("Error parsing input date: %v", err)
		}
		outDate, outTime := timeToDateTime(input)
		if outDate != tt.date || outTime != tt.time {
			t.Errorf("timeToDateTime(%v) expected output %d %d, actual %d %d", tt.rfc, tt.date, tt.time, outDate, outTime)
		}
	}

}

func TestDirectoryEntryLfnChecksum(t *testing.T) {
	/*
		the values for the hashes are taken from testdata/calcsfn_checksum.c, which is based on the
		formula given at https://en.wikipedia.org/wiki/Design_of_the_FAT_file_system#VFAT_long_file_names
	*/
	tests := []struct {
		name      string
		extension string
		output    byte
		err       error
	}{
		// first all of the error cases
		{"abc\u2378", "F", 0x00, fmt.Errorf("Invalid shortname character in filename")},
		{"abc", "F", 0x00, fmt.Errorf("Invalid shortname character in filename")},
		{"ABC", "F\u2378", 0x00, fmt.Errorf("Invalid shortname character in extension")},
		{"ABC", "f", 0x00, fmt.Errorf("Invalid shortname character in extension")},
		{"ABCDEFGHIJ", "F", 0x00, fmt.Errorf("Short name for file is longer than")},
		{"ABCD", "FUUYY", 0x00, fmt.Errorf("Extension for file is longer than")},
		// valid exact length of each
		{"ABCDEFGH", "TXT", 0xf6, nil},
		// shortened each
		{"ABCDEFG", "TXT", 0x51, nil},
		{"ABCDEFGH", "TX", 0xc2, nil},
		{"ABCDEF", "T", 0xcf, nil},
	}
	for _, tt := range tests {
		output, err := lfnChecksum(tt.name, tt.extension)
		if output != tt.output {
			t.Errorf("lfnChecksum(%s,%s) expected output %v, actual %v", tt.name, tt.extension, tt.output, output)
		}
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
	}
}

func TestDirectoryEntryStringToASCIIBytes(t *testing.T) {
	tests := []struct {
		input  string
		output []byte
		err    error
	}{
		{"abc", []byte{0x61, 0x62, 0x63}, nil},
		{"abcdefg", []byte{0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67}, nil},
		{"abcdef\u2318", nil, fmt.Errorf("Non-ASCII character in name: %s", "abcdef\u2318")},
	}
	for _, tt := range tests {
		output, err := stringToASCIIBytes(tt.input)
		if bytes.Compare(output, tt.output) != 0 {
			t.Errorf("stringToASCIIBytes(%s) expected output %v, actual %v", tt.input, tt.output, output)
		}
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
	}

}

func TestDirectoryEntryCalculateSlots(t *testing.T) {
	// holds 13 chars per slot, so test x<13, x==13, 13<x<26, x==26, 26< x
	tests := []struct {
		input string
		slots int
	}{
		{"abc", 1},
		{"abcdefghijklm", 1},
		{"abcdefghijklmn", 2},
		{"abcdefghijklmnopqrstuvwxyz", 2},
		{"abcdefghijklmnopqrstuvwxyz1", 3},
	}
	for _, tt := range tests {
		slots := calculateSlots(tt.input)
		if slots != tt.slots {
			t.Errorf("calculateSlots(%s) expected %d , actual %d", tt.input, tt.slots, slots)
		}
	}

}

func TestDirectoryEntryConvertLfnSfn(t *testing.T) {
	tests := []struct {
		input       string
		sfn         string
		extension   string
		isLfn       bool
		isTruncated bool
	}{
		{"ABC", "ABC", "", false, false},
		{"ABC.TXT", "ABC", "TXT", false, false},
		{"abc", "ABC", "", true, false},
		{"ABC.TXTTT", "ABC", "TXT", true, false},
		{"ABC.txt", "ABC", "TXT", true, false},
		{"aBC.q", "ABC", "Q", true, false},
		{"ABC.q.rt", "ABCQ", "RT", true, false},
		{"VeryLongName.ft", "VERYLO~1", "FT", true, true},
	}
	for _, tt := range tests {
		sfn, extension, isLfn, isTruncated := convertLfnSfn(tt.input)
		if sfn != tt.sfn || extension != tt.extension || isLfn != tt.isLfn || isTruncated != tt.isTruncated {
			t.Errorf("convertLfnSfn(%s) expected %s / %s / %t / %t ; actual %s / %s / %t / %t", tt.input, tt.sfn, tt.extension, tt.isLfn, tt.isTruncated, sfn, extension, isLfn, isTruncated)
		}
	}

}

func TestDirectoryEntryUCaseValid(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"abc", "ABC"},
		{"ABC", "ABC"},
		{"aBC", "ABC"},
		{"a15D", "A15D"},
		{"A BC", "ABC"},
		{"A..-a*)82y12112bb", "A_A__82Y12112BB"},
	}
	for _, tt := range tests {
		output := uCaseValid(tt.input)
		if output != tt.output {
			t.Errorf("uCaseValid(%s) expected %s actual %s", tt.input, tt.output, output)
		}
	}
}

func TestDirectoryEntryParseDirEntries(t *testing.T) {
	validDe, validBytes, err := getValidDirectoryEntries()
	// validBytes is ordered [][]byte - just string them all together
	b := make([]byte, 0)
	for _, b2 := range validBytes {
		b = append(b, b2...)
	}
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		de  []*directoryEntry
		b   []byte
		err error
	}{
		{validDe, b, nil},
	}

	for _, tt := range tests {
		output, err := parseDirEntries(tt.b, nil)
		switch {
		case (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Log(err)
			t.Log(tt.err)
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		case (output == nil && tt.de != nil) || (tt.de == nil && output != nil):
			t.Errorf("parseDirEntries() DirectoryEntry mismatched nil actual, expected %v %v", output, tt.de)
		case len(output) != len(tt.de):
			t.Errorf("parseDirEntries() DirectoryEntry mismatched length actual, expected %d %d", len(output), len(tt.de))
		default:
			for i, de := range output {
				if *de != *tt.de[i] {
					t.Errorf("%d: parseDirEntries() DirectoryEntry mismatch, actual then valid:", i)
					t.Log(de)
					t.Log(tt.de[i])
				}
			}
		}
	}

}

func TestDirectoryEntryToBytes(t *testing.T) {
	validDe, validBytes, err := getValidDirectoryEntries()
	if err != nil {
		t.Fatal(err)
	}
	for i, de := range validDe {
		b, err := de.toBytes()
		if err != nil {
			t.Errorf("Error converting directory entry to bytes: %v", err)
			t.Logf("%v", de)
		} else {
			if bytes.Compare(b, validBytes[i]) != 0 {
				t.Errorf("Mismatched bytes %s, actual vs expected", de.filenameShort)
				t.Log(b)
				t.Log(validBytes[i])
			}
		}
	}
}
