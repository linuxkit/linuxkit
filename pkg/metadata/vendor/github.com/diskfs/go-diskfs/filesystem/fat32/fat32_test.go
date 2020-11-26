package fat32_test

/*
 These tests the exported functions
 We want to do full-in tests with files
*/

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/testhelper"
	"github.com/diskfs/go-diskfs/util"
)

var (
	intImage     = os.Getenv("TEST_IMAGE")
	keepTmpFiles = os.Getenv("KEEPTESTFILES")
)

func getOpenMode(mode int) string {
	modes := make([]string, 0, 0)
	if mode&os.O_CREATE == os.O_CREATE {
		modes = append(modes, "CREATE")
	}
	if mode&os.O_APPEND == os.O_APPEND {
		modes = append(modes, "APPEND")
	}
	if mode&os.O_RDWR == os.O_RDWR {
		modes = append(modes, "RDWR")
	} else {
		modes = append(modes, "RDONLY")
	}
	return strings.Join(modes, "|")
}

func tmpFat32(fill bool, embedPre, embedPost int64) (*os.File, error) {
	filename := "fat32_test"
	f, err := ioutil.TempFile("", filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to create tempfile %s :%v", filename, err)
	}

	// either copy the contents of the base file over, or make a file of similar size
	b, err := ioutil.ReadFile(fat32.Fat32File)
	if err != nil {
		return nil, fmt.Errorf("Failed to read contents of %s: %v", fat32.Fat32File, err)
	}
	if embedPre > 0 {
		empty := make([]byte, embedPre, embedPre)
		written, err := f.Write(empty)
		if err != nil {
			return nil, fmt.Errorf("Failed to write %d zeroes at beginning of %s: %v", embedPre, filename, err)
		}
		if written != len(empty) {
			return nil, fmt.Errorf("Wrote only %d zeroes at beginning of %s instead of %d", written, filename, len(empty))
		}
	}
	if fill {
		written, err := f.Write(b)
		if err != nil {
			return nil, fmt.Errorf("Failed to write contents of %s to %s: %v", fat32.Fat32File, filename, err)
		}
		if written != len(b) {
			return nil, fmt.Errorf("Wrote only %d bytes of %s to %s instead of %d", written, fat32.Fat32File, filename, len(b))
		}
	} else {
		size := int64(len(b))
		empty := make([]byte, size, size)
		written, err := f.Write(empty)
		if err != nil {
			return nil, fmt.Errorf("Failed to write %d zeroes as content of %s: %v", size, filename, err)
		}
		if written != len(empty) {
			return nil, fmt.Errorf("Wrote only %d zeroes as content of %s instead of %d", written, filename, len(empty))
		}
	}
	if embedPost > 0 {
		empty := make([]byte, embedPost, embedPost)
		written, err := f.Write(empty)
		if err != nil {
			return nil, fmt.Errorf("Failed to write %d zeroes at end of %s: %v", embedPost, filename, err)
		}
		if written != len(empty) {
			return nil, fmt.Errorf("Wrote only %d zeroes at end of %s instead of %d", written, filename, len(empty))
		}
	}

	return f, nil
}

func TestFat32Type(t *testing.T) {
	fs := &fat32.FileSystem{}
	fstype := fs.Type()
	expected := filesystem.TypeFat32
	if fstype != expected {
		t.Errorf("Type() returns %v instead of expected %v", fstype, expected)
	}
}

func TestFat32Mkdir(t *testing.T) {
	// only do this test if os.Getenv("TEST_IMAGE") contains a real image
	if intImage == "" {
		return
	}
	runTest := func(t *testing.T, post, pre int64, fatFunc func(util.File, int64, int64, int64) (*fat32.FileSystem, error)) {
		// create our directories
		tests := []string{
			"/",
			"/foo",
			"/foo/bar",
			"/a/b/c",
		}
		f, err := tmpFat32(true, pre, post)
		if err != nil {
			t.Fatal(err)
		}
		if keepTmpFiles == "" {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}
		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error getting file info for tmpfile %s: %v", f.Name(), err)
		}
		fs, err := fatFunc(f, fileInfo.Size()-pre-post, pre, 512)
		if err != nil {
			t.Fatalf("Error reading fat32 filesystem from %s: %v", f.Name(), err)
		}
		for _, p := range tests {
			err := fs.Mkdir(p)
			switch {
			case err != nil:
				t.Errorf("Mkdir(%s): error %v", p, err)
			default:
				// check that the directory actually was created
				output := new(bytes.Buffer)
				f.Seek(0, 0)
				err := testhelper.DockerRun(f, output, false, true, intImage, "mdir", "-i", fmt.Sprintf("%s@@%d", "/file.img", pre), fmt.Sprintf("::%s", p))
				if err != nil {
					t.Errorf("Mkdir(%s): Unexpected err: %v", p, err)
					t.Log(output.String())
				}
			}
		}
	}
	t.Run("Read to Mkdir", func(t *testing.T) {
		t.Run("entire image", func(t *testing.T) {
			runTest(t, 0, 0, fat32.Read)
		})
		t.Run("embedded filesystem", func(t *testing.T) {
			runTest(t, 500, 1000, fat32.Read)
		})
	})
	t.Run("Create to Mkdir", func(t *testing.T) {
		// This is to enable Create "fit" into the common testing logic
		createShim := func(file util.File, size int64, start int64, blocksize int64) (*fat32.FileSystem, error) {
			return fat32.Create(file, size, start, blocksize, "")
		}
		t.Run("entire image", func(t *testing.T) {
			runTest(t, 0, 0, createShim)
		})
		t.Run("embedded filesystem", func(t *testing.T) {
			runTest(t, 500, 1000, createShim)
		})
	})
}

func TestFat32Create(t *testing.T) {
	tests := []struct {
		blocksize int64
		filesize  int64
		fs        *fat32.FileSystem
		err       error
	}{
		{500, 6000, nil, fmt.Errorf("blocksize for FAT32 must be")},
		{513, 6000, nil, fmt.Errorf("blocksize for FAT32 must be")},
		{512, fat32.Fat32MaxSize + 100000, nil, fmt.Errorf("requested size is larger than maximum allowed FAT32")},
		{512, 0, nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32")},
		{512, 10000000, &fat32.FileSystem{}, nil},
	}
	runTest := func(t *testing.T, pre, post int64) {
		for _, tt := range tests {
			// get a temporary working file
			f, err := tmpFat32(false, pre, post)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			// create the filesystem
			fs, err := fat32.Create(f, tt.filesize-pre-post, pre, tt.blocksize, "")
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("Create(%s, %d, %d, %d): mismatched errors\nactual %v\nexpected %v", f.Name(), tt.filesize, 0, tt.blocksize, err, tt.err)
			case (fs == nil && tt.fs != nil) || (fs != nil && tt.fs == nil):
				t.Errorf("Create(%s, %d, %d, %d): mismatched fs\nactual %v\nexpected %v", f.Name(), tt.filesize, 0, tt.blocksize, fs, tt.fs)
			}
			// we do not match the filesystems here, only check functional accuracy
		}

	}

	t.Run("entire image", func(t *testing.T) {
		runTest(t, 0, 0)
	})
	t.Run("embedded filesystem", func(t *testing.T) {
		runTest(t, 500, 1000)
	})
}

func TestFat32Read(t *testing.T) {
	// test cases:
	// - invalid blocksize
	// - invalid file size (0 and too big)
	// - invalid FSISBootSector
	// - valid file
	tests := []struct {
		blocksize  int64
		filesize   int64
		bytechange int64
		fs         *fat32.FileSystem
		err        error
	}{
		{500, 6000, -1, nil, fmt.Errorf("blocksize for FAT32 must be")},
		{513, 6000, -1, nil, fmt.Errorf("blocksize for FAT32 must be")},
		{512, fat32.Fat32MaxSize + 10000, -1, nil, fmt.Errorf("requested size is larger than maximum allowed FAT32 size")},
		{512, 0, -1, nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32 size")},
		{512, 10000000, 512, nil, fmt.Errorf("Error reading FileSystem Information Sector")},
		{512, 10000000, -1, &fat32.FileSystem{}, nil},
	}
	runTest := func(t *testing.T, pre, post int64) {
		for _, tt := range tests {
			// get a temporary working file
			f, err := tmpFat32(true, pre, post)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			// make any changes needed to corrupt it
			corrupted := ""
			if tt.bytechange >= 0 {
				b := make([]byte, 1, 1)
				rand.Read(b)
				f.WriteAt(b, tt.bytechange+pre)
				corrupted = fmt.Sprintf("corrupted %d", tt.bytechange+pre)
			}
			// create the filesystem
			fs, err := fat32.Read(f, tt.filesize-pre-post, pre, tt.blocksize)
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("Read(%s, %d, %d, %d) %s: mismatched errors, actual %v expected %v", f.Name(), tt.filesize, 0, tt.blocksize, corrupted, err, tt.err)
			case (fs == nil && tt.fs != nil) || (fs != nil && tt.fs == nil):
				t.Errorf("Read(%s, %d, %d, %d) %s: mismatched fs, actual then expected", f.Name(), tt.filesize, 0, tt.blocksize, corrupted)
				t.Logf("%v", fs)
				t.Logf("%v", tt.fs)
			}
			// we do not match the filesystems here, only check functional accuracy
		}
	}
	t.Run("entire image", func(t *testing.T) {
		runTest(t, 0, 0)
	})
	t.Run("embedded filesystem", func(t *testing.T) {
		runTest(t, 500, 1000)
	})
}

func TestFat32ReadDir(t *testing.T) {
	runTest := func(t *testing.T, pre, post int64) {
		// get a temporary working file
		f, err := tmpFat32(true, pre, post)
		if err != nil {
			t.Fatal(err)
		}
		if keepTmpFiles == "" {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}
		tests := []struct {
			path  string
			count int
			name  string
			isDir bool
			err   error
		}{
			// should have 4 entries
			//   foo
			//   TERCER~1
			//   CORTO1.TXT
			//   UNARCH~1.DAT
			{"/", 4, "foo", true, nil},
			// should have 80 entries:
			//  dir0-75 = 76 entries
			//  dir     =  1 entry
			//  bar     =  1 entry
			//    .     =  1 entry
			//   ..     =  1 entry
			// total = 80 entries
			{"/foo", 80, ".", true, nil},
			// 0 entries because the directory does not exist
			{"/a/b/c", 0, "", false, fmt.Errorf("Error reading directory /a/b/c")},
		}
		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error getting file info for tmpfile %s: %v", f.Name(), err)
		}
		fs, err := fat32.Read(f, fileInfo.Size()-pre-post, pre, 512)
		if err != nil {
			t.Fatalf("Error reading fat32 filesystem from %s: %v", f.Name(), err)
		}
		for _, tt := range tests {
			output, err := fs.ReadDir(tt.path)
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("ReadDir(%s): mismatched errors, actual: %v , expected: %v", tt.path, err, tt.err)
			case output == nil && tt.err == nil:
				t.Errorf("ReadDir(%s): Unexpected nil output", tt.path)
			case len(output) != tt.count:
				t.Errorf("ReadDir(%s): output gave %d entries instead of expected %d", tt.path, len(output), tt.count)
			case output != nil && len(output) > 0 && output[0].IsDir() != tt.isDir:
				t.Errorf("ReadDir(%s): output gave directory %t expected %t", tt.path, output[0].IsDir(), tt.isDir)
			case output != nil && len(output) > 0 && output[0].Name() != tt.name:
				t.Errorf("ReadDir(%s): output gave name %s expected %s", tt.path, output[0].Name(), tt.name)
			}
		}
	}
	t.Run("entire image", func(t *testing.T) {
		runTest(t, 0, 0)
	})
	t.Run("embedded filesystem", func(t *testing.T) {
		runTest(t, 500, 1000)
	})
}

func TestFat32OpenFile(t *testing.T) {
	// opening directories and files for reading
	t.Run("Read", func(t *testing.T) {
		runTest := func(t *testing.T, pre, post int64) {
			// get a temporary working file
			f, err := tmpFat32(true, pre, post)
			if err != nil {
				t.Fatal(err)
			}
			if keepTmpFiles == "" {
				defer os.Remove(f.Name())
			} else {
				fmt.Println(f.Name())
			}
			tests := []struct {
				path     string
				mode     int
				expected string
				err      error
			}{
				// error opening a directory
				{"/", os.O_RDONLY, "", fmt.Errorf("Cannot open directory %s as file", "/")},
				{"/", os.O_RDWR, "", fmt.Errorf("Cannot open directory %s as file", "/")},
				{"/", os.O_CREATE, "", fmt.Errorf("Cannot open directory %s as file", "/")},
				// open non-existent file for read or read write
				{"/abcdefg", os.O_RDONLY, "", fmt.Errorf("Target file %s does not exist", "/abcdefg")},
				{"/abcdefg", os.O_RDWR, "", fmt.Errorf("Target file %s does not exist", "/abcdefg")},
				{"/abcdefg", os.O_APPEND, "", fmt.Errorf("Target file %s does not exist", "/abcdefg")},
				// open file for read or read write and check contents
				{"/CORTO1.TXT", os.O_RDONLY, "Tenemos un archivo corto\n", nil},
				{"/CORTO1.TXT", os.O_RDWR, "Tenemos un archivo corto\n", nil},
				// open file for create that already exists
				//{"/CORTO1.TXT", os.O_CREATE | os.O_RDWR, "Tenemos un archivo corto\n", nil},
				//{"/CORTO1.TXT", os.O_CREATE | os.O_RDONLY, "Tenemos un archivo corto\n", nil},
			}
			fileInfo, err := f.Stat()
			if err != nil {
				t.Fatalf("Error getting file info for tmpfile %s: %v", f.Name(), err)
			}
			fs, err := fat32.Read(f, fileInfo.Size()-pre-post, pre, 512)
			if err != nil {
				t.Fatalf("Error reading fat32 filesystem from %s: %v", f.Name(), err)
			}
			for _, tt := range tests {
				header := fmt.Sprintf("OpenFile(%s, %s)", tt.path, getOpenMode(tt.mode))
				reader, err := fs.OpenFile(tt.path, tt.mode)
				switch {
				case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
					t.Errorf("%s: mismatched errors, actual: %v , expected: %v", header, err, tt.err)
				case reader == nil && (tt.err == nil || tt.expected != ""):
					t.Errorf("%s: Unexpected nil output", header)
				case reader != nil:
					b, err := ioutil.ReadAll(reader)
					if err != nil {
						t.Errorf("%s: ioutil.ReadAll(reader) unexpected error: %v", header, err)
					}
					if string(b) != tt.expected {
						t.Errorf("%s: mismatched contents, actual then expected", header)
						t.Log(string(b))
						t.Log(tt.expected)
					}
				}
			}
		}
		t.Run("entire image", func(t *testing.T) {
			runTest(t, 0, 0)
		})
		t.Run("embedded filesystem", func(t *testing.T) {
			runTest(t, 500, 1000)
		})
	})

	// write / create-and-write files and check contents
	// *** Write - writes right after last write or read
	// *** Read - reads right after last write or read
	// ** WriteAt - writes at specific location in file
	// ** ReadAt - reads at specific location in file
	t.Run("Write", func(t *testing.T) {
		runTest := func(t *testing.T, pre, post int64) {
			tests := []struct {
				path      string
				mode      int
				beginning bool // true = "Seek() to beginning of file before writing"; false = "read entire file then write"
				contents  string
				expected  string
				err       error
			}{
				//  - open for create file that does not exist (write contents, check that written)
				{"/abcdefg", os.O_RDWR | os.O_CREATE, false, "This is a test", "This is a test", nil},
				//  - open for readwrite file that does exist (write contents, check that overwritten)
				{"/CORTO1.TXT", os.O_RDWR, true, "This is a very long replacement string", "This is a very long replacement string", nil},
				{"/CORTO1.TXT", os.O_RDWR, true, "Two", "Twoemos un archivo corto\n", nil},
				{"/CORTO1.TXT", os.O_RDWR, false, "This is a very long replacement string", "Tenemos un archivo corto\nThis is a very long replacement string", nil},
				{"/CORTO1.TXT", os.O_RDWR, false, "Two", "Tenemos un archivo corto\nTwo", nil},
				//  - open for append file that does exist (write contents, check that appended)
				{"/CORTO1.TXT", os.O_APPEND, false, "More", "", fmt.Errorf("Cannot write to file opened read-only")},
				{"/CORTO1.TXT", os.O_APPEND | os.O_RDWR, false, "More", "Tenemos un archivo corto\nMore", nil},
				{"/CORTO1.TXT", os.O_APPEND, true, "More", "", fmt.Errorf("Cannot write to file opened read-only")},
				{"/CORTO1.TXT", os.O_APPEND | os.O_RDWR, true, "More", "Moremos un archivo corto\n", nil},
			}
			for _, tt := range tests {
				header := fmt.Sprintf("OpenFile(%s, %s, %t)", tt.path, getOpenMode(tt.mode), tt.beginning)
				// get a temporary working file
				f, err := tmpFat32(true, pre, post)
				if err != nil {
					t.Fatal(err)
				}
				if keepTmpFiles == "" {
					defer os.Remove(f.Name())
				} else {
					fmt.Println(f.Name())
				}
				fileInfo, err := f.Stat()
				if err != nil {
					t.Fatalf("Error getting file info for tmpfile %s: %v", f.Name(), err)
				}
				fs, err := fat32.Read(f, fileInfo.Size()-pre-post, pre, 512)
				if err != nil {
					t.Fatalf("Error reading fat32 filesystem from %s: %v", f.Name(), err)
				}
				readWriter, err := fs.OpenFile(tt.path, tt.mode)
				switch {
				case err != nil:
					t.Errorf("%s: unexpected error: %v", header, err)
				case readWriter == nil:
					t.Errorf("%s: Unexpected nil output", header)
				default:
					// write and then read
					bWrite := []byte(tt.contents)
					if tt.beginning {
						offset, err := readWriter.Seek(0, 0)
						if err != nil {
							t.Errorf("%s: Seek(0,0) unexpected error: %v", header, err)
							continue
						}
						if offset != 0 {
							t.Errorf("%s: Seek(0,0) reset to %d instead of %d", header, offset, 0)
							continue
						}
					} else {
						b := make([]byte, 512, 512)
						_, err := readWriter.Read(b)
						if err != nil && err != io.EOF {
							t.Errorf("%s: ioutil.ReadAll(readWriter) unexpected error: %v", header, err)
							continue
						}
					}
					written, writeErr := readWriter.Write(bWrite)
					readWriter.Seek(0, 0)
					bRead, readErr := ioutil.ReadAll(readWriter)

					switch {
					case readErr != nil:
						t.Errorf("%s: ioutil.ReadAll() unexpected error: %v", header, readErr)
					case (writeErr == nil && tt.err != nil) || (writeErr != nil && tt.err == nil) || (writeErr != nil && tt.err != nil && !strings.HasPrefix(writeErr.Error(), tt.err.Error())):
						t.Errorf("%s: readWriter.Write(b) mismatched errors, actual: %v , expected: %v", header, writeErr, tt.err)
					case written != len(bWrite) && tt.err == nil:
						t.Errorf("%s: readWriter.Write(b) wrote %d bytes instead of expected %d", header, written, len(bWrite))
					case string(bRead) != tt.expected && tt.err == nil:
						t.Errorf("%s: mismatched contents, actual then expected", header)
						t.Log(string(bRead))
						t.Log(tt.expected)
					}
				}
			}
		}
		t.Run("entire image", func(t *testing.T) {
			runTest(t, 0, 0)
		})
		t.Run("embedded filesystem", func(t *testing.T) {
			runTest(t, 500, 1000)
		})
	})

	// large file should cross multiple clusters
	// out cluster size is 512 bytes, so make it 10+ clusters
	t.Run("Large File", func(t *testing.T) {
		runTest := func(t *testing.T, pre, post int64) {
			// get a temporary working file
			f, err := tmpFat32(true, pre, post)
			if err != nil {
				t.Fatal(err)
			}
			if keepTmpFiles == "" {
				defer os.Remove(f.Name())
			} else {
				fmt.Println(f.Name())
			}
			fileInfo, err := f.Stat()
			if err != nil {
				t.Fatalf("Error getting file info for tmpfile %s: %v", f.Name(), err)
			}
			fs, err := fat32.Read(f, fileInfo.Size()-pre-post, pre, 512)
			if err != nil {
				t.Fatalf("Error reading fat32 filesystem from %s: %v", f.Name(), err)
			}
			path := "/abcdefghi"
			mode := os.O_RDWR | os.O_CREATE
			// each cluster is 512 bytes, so use 10 clusters and a bit of another
			size := 10*512 + 22
			bWrite := make([]byte, size, size)
			header := fmt.Sprintf("OpenFile(%s, %s)", path, getOpenMode(mode))
			readWriter, err := fs.OpenFile(path, mode)
			switch {
			case err != nil:
				t.Errorf("%s: unexpected error: %v", header, err)
			case readWriter == nil:
				t.Errorf("%s: Unexpected nil output", header)
			default:
				// write and then read
				rand.Read(bWrite)
				written, writeErr := readWriter.Write(bWrite)
				readWriter.Seek(0, 0)
				bRead, readErr := ioutil.ReadAll(readWriter)

				switch {
				case readErr != nil:
					t.Errorf("%s: ioutil.ReadAll() unexpected error: %v", header, readErr)
				case writeErr != nil:
					t.Errorf("%s: readWriter.Write(b) unexpected error: %v", header, writeErr)
				case written != len(bWrite):
					t.Errorf("%s: readWriter.Write(b) wrote %d bytes instead of expected %d", header, written, len(bWrite))
				case bytes.Compare(bWrite, bRead) != 0:
					t.Errorf("%s: mismatched contents, read %d expected %d, actual data then expected:", header, len(bRead), len(bWrite))
					//t.Log(bRead)
					//t.Log(bWrite)
				}
			}
		}
		t.Run("entire image", func(t *testing.T) {
			runTest(t, 0, 0)
		})
		t.Run("embedded filesystem", func(t *testing.T) {
			runTest(t, 500, 1000)
		})
	})

}
