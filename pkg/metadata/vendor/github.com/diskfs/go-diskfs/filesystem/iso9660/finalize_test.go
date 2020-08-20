package iso9660_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"github.com/diskfs/go-diskfs/testhelper"
)

var (
	intImage = os.Getenv("TEST_IMAGE")
)

// test creating an iso with el torito boot
func TestFinalizeElTorito(t *testing.T) {
	blocksize := int64(2048)
	f, err := ioutil.TempFile("", "iso_finalize_test")
	defer os.Remove(f.Name())
	//fmt.Println(f.Name())
	if err != nil {
		t.Fatalf("Failed to create tmpfile: %v", err)
	}
	fs, err := iso9660.Create(f, 0, 0, blocksize)
	if err != nil {
		t.Fatalf("Failed to iso9660.Create: %v", err)
	}
	var isofile filesystem.File
	for _, filename := range []string{"/BOOT1.IMG", "/BOOT2.IMG"} {
		isofile, err = fs.OpenFile(filename, os.O_CREATE|os.O_RDWR)
		if err != nil {
			t.Fatalf("Failed to iso9660.OpenFile(%s): %v", filename, err)
		}
		// create some random data
		blen := 1024 * 1024
		for i := 0; i < 5; i++ {
			b := make([]byte, blen)
			_, err = rand.Read(b)
			if err != nil {
				t.Fatalf("%d: error getting random bytes for file %s: %v", i, filename, err)
			}
			if _, err = isofile.Write(b); err != nil {
				t.Fatalf("%d: error writing random bytes to tmpfile %s: %v", i, filename, err)
			}
		}
	}

	err = fs.Finalize(iso9660.FinalizeOptions{ElTorito: &iso9660.ElTorito{
		BootCatalog:     "/BOOT.CAT",
		HideBootCatalog: false,
		Platform:        iso9660.EFI,
		Entries: []*iso9660.ElToritoEntry{
			{Platform: iso9660.BIOS, Emulation: iso9660.NoEmulation, BootFile: "/BOOT1.IMG", HideBootFile: true, LoadSegment: 0, SystemType: mbr.Fat32LBA},
			{Platform: iso9660.EFI, Emulation: iso9660.NoEmulation, BootFile: "/BOOT2.IMG", HideBootFile: false, LoadSegment: 0, SystemType: mbr.Fat32LBA},
		},
	},
	})
	if err != nil {
		t.Fatal("Unexpected error fs.Finalize()", err)
	}
	if err != nil {
		t.Fatalf("Error trying to Stat() iso file: %v", err)
	}

	// now check the contents
	fs, err = iso9660.Read(f, 0, 0, 2048)
	if err != nil {
		t.Fatalf("error reading the tmpfile as iso: %v", err)
	}

	// we chose to hide the first one, so check the first one exists and not the second
	_, err = fs.OpenFile("/BOOT1.IMG", os.O_RDONLY)
	if err == nil {
		t.Errorf("Did not receive expected error opening file %s: %v", "/BOOT1.IMG", err)
	}
	_, err = fs.OpenFile("/BOOT2.IMG", os.O_RDONLY)
	if err != nil {
		t.Errorf("Error opening file %s: %v", "/BOOT2.IMG", err)
	}

	validateIso(t, f)

	validateElTorito(t, f)

	// close the file
	err = f.Close()
	if err != nil {
		t.Fatalf("Could not close iso file: %v", err)
	}
}

// full test - create some files, finalize, check the output
func TestFinalize9660(t *testing.T) {
	blocksize := int64(2048)
	t.Run("deep dir", func(t *testing.T) {
		f, err := ioutil.TempFile("", "iso_finalize_test")
		defer os.Remove(f.Name())
		//fmt.Println(f.Name())
		if err != nil {
			t.Fatalf("Failed to create tmpfile: %v", err)
		}
		fs, err := iso9660.Create(f, 0, 0, blocksize)
		if err != nil {
			t.Fatalf("Failed to iso9660.Create: %v", err)
		}
		for _, dir := range []string{"/A/B/C/D/E/F/G/H/I/J/K"} {
			err = fs.Mkdir(dir)
			if err != nil {
				t.Fatalf("Failed to iso9660.Mkdir(%s): %v", dir, err)
			}
		}

		err = fs.Finalize(iso9660.FinalizeOptions{})
		if err == nil {
			t.Fatal("Unexpected lack of error fs.Finalize()", err)
		}
	})
	t.Run("valid", func(t *testing.T) {
		f, err := ioutil.TempFile("", "iso_finalize_test")
		defer os.Remove(f.Name())
		//fmt.Println(f.Name())
		if err != nil {
			t.Fatalf("Failed to create tmpfile: %v", err)
		}
		fs, err := iso9660.Create(f, 0, 0, blocksize)
		if err != nil {
			t.Fatalf("Failed to iso9660.Create: %v", err)
		}
		for _, dir := range []string{"/", "/FOO", "/BAR", "/ABC"} {
			err = fs.Mkdir(dir)
			if err != nil {
				t.Fatalf("Failed to iso9660.Mkdir(%s): %v", dir, err)
			}
		}
		var isofile filesystem.File
		for _, filename := range []string{"/BAR/LARGEFILE", "/ABC/LARGEFILE"} {
			isofile, err = fs.OpenFile(filename, os.O_CREATE|os.O_RDWR)
			if err != nil {
				t.Fatalf("Failed to iso9660.OpenFile(%s): %v", filename, err)
			}
			// create some random data
			blen := 1024 * 1024
			for i := 0; i < 5; i++ {
				b := make([]byte, blen)
				_, err = rand.Read(b)
				if err != nil {
					t.Fatalf("%d: error getting random bytes for file %s: %v", i, filename, err)
				}
				if _, err = isofile.Write(b); err != nil {
					t.Fatalf("%d: error writing random bytes to tmpfile %s: %v", i, filename, err)
				}
			}
		}

		isofile, err = fs.OpenFile("README.MD", os.O_CREATE|os.O_RDWR)
		if err != nil {
			t.Fatalf("Failed to iso9660.OpenFile(%s): %v", "README.MD", err)
		}
		b := []byte("readme\n")
		if _, err = isofile.Write(b); err != nil {
			t.Fatalf("error writing %s to tmpfile %s: %v", string(b), "README.MD", err)
		}

		fooCount := 75
		for i := 0; i <= fooCount; i++ {
			filename := fmt.Sprintf("/FOO/FILENAME_%d", i)
			contents := []byte(fmt.Sprintf("filename_%d\n", i))
			isofile, err = fs.OpenFile(filename, os.O_CREATE|os.O_RDWR)
			if err != nil {
				t.Fatalf("Failed to iso9660.OpenFile(%s): %v", filename, err)
			}
			if _, err = isofile.Write(contents); err != nil {
				t.Fatalf("%d: error writing bytes to tmpfile %s: %v", i, filename, err)
			}
		}

		err = fs.Finalize(iso9660.FinalizeOptions{})
		if err != nil {
			t.Fatal("Unexpected error fs.Finalize()", err)
		}
		// now need to check contents
		fi, err := f.Stat()
		if err != nil {
			t.Fatalf("Error trying to Stat() iso file: %v", err)
		}
		// we made two 5MB files, so should be at least 10MB
		if fi.Size() < 10*1024*1024 {
			t.Fatalf("Resultant file too small after finalizing %d", fi.Size())
		}

		// now check the contents
		fs, err = iso9660.Read(f, 0, 0, 2048)
		if err != nil {
			t.Fatalf("error reading the tmpfile as iso: %v", err)
		}

		dirFi, err := fs.ReadDir("/")
		if err != nil {
			t.Errorf("error reading the root directory from iso: %v", err)
		}
		// we expect to have 3 entries: ABC BAR and FOO
		expected := map[string]bool{
			"ABC": false, "BAR": false, "FOO": false, "README.MD": false,
		}
		for _, e := range dirFi {
			delete(expected, e.Name())
		}
		if len(expected) > 0 {
			keys := make([]string, 0)
			for k := range expected {
				keys = append(keys, k)
			}
			t.Errorf("Some entries not found in root: %v", keys)
		}

		// get a few files I expect
		fileContents := map[string]string{
			"/README.MD":       "readme\n",
			"/FOO/FILENAME_50": "filename_50\n",
			"/FOO/FILENAME_2":  "filename_2\n",
		}

		for k, v := range fileContents {
			var (
				f    filesystem.File
				read int
			)

			f, err = fs.OpenFile(k, os.O_RDONLY)
			if err != nil {
				t.Errorf("Error opening file %s: %v", k, err)
				continue
			}
			// check the contents
			b := make([]byte, 50, 50)
			read, err = f.Read(b)
			if err != nil && err != io.EOF {
				t.Errorf("Error reading from file %s: %v", k, err)
			}
			actual := string(b[:read])
			if actual != v {
				t.Errorf("Mismatched content, actual '%s' expected '%s'", actual, v)
			}
		}

		validateIso(t, f)

		// close the file
		err = f.Close()
		if err != nil {
			t.Fatalf("Could not close iso file: %v", err)
		}
	})
}

func TestFinalizeRockRidge(t *testing.T) {
	blocksize := int64(2048)
	t.Run("valid", func(t *testing.T) {
		f, err := ioutil.TempFile("", "iso_finalize_test")
		defer os.Remove(f.Name())
		//fmt.Println(f.Name())
		if err != nil {
			t.Fatalf("Failed to create tmpfile: %v", err)
		}
		fs, err := iso9660.Create(f, 0, 0, blocksize)
		if err != nil {
			t.Fatalf("Failed to iso9660.Create: %v", err)
		}
		for _, dir := range []string{"/", "/foo", "/bar", "/abc"} {
			err = fs.Mkdir(dir)
			if err != nil {
				t.Fatalf("Failed to iso9660.Mkdir(%s): %v", dir, err)
			}
		}
		// make a deep directory
		dir := "/deep/a/b/c/d/e/f/g/h/i/j"
		err = fs.Mkdir(dir)
		if err != nil {
			t.Fatalf("Failed to iso9660.Mkdir(%s): %v", dir, err)
		}
		var isofile filesystem.File
		for _, filename := range []string{"/bar/largefile", "/abc/largefile"} {
			isofile, err = fs.OpenFile(filename, os.O_CREATE|os.O_RDWR)
			if err != nil {
				t.Fatalf("Failed to iso9660.OpenFile(%s): %v", filename, err)
			}
			// create some random data
			blen := 1024 * 1024
			for i := 0; i < 5; i++ {
				b := make([]byte, blen)
				_, err = rand.Read(b)
				if err != nil {
					t.Fatalf("%d: error getting random bytes for file %s: %v", i, filename, err)
				}
				if _, err = isofile.Write(b); err != nil {
					t.Fatalf("%d: error writing random bytes to tmpfile %s: %v", i, filename, err)
				}
			}
		}

		isofile, err = fs.OpenFile("README.md", os.O_CREATE|os.O_RDWR)
		if err != nil {
			t.Fatalf("Failed to iso9660.OpenFile(%s): %v", "README.md", err)
		}
		b := []byte("readme\n")
		if _, err = isofile.Write(b); err != nil {
			t.Fatalf("error writing %s to tmpfile %s: %v", string(b), "README.md", err)
		}

		fooCount := 75
		for i := 0; i <= fooCount; i++ {
			filename := fmt.Sprintf("/foo/filename_%d", i)
			contents := []byte(fmt.Sprintf("filename_%d\n", i))
			isofile, err = fs.OpenFile(filename, os.O_CREATE|os.O_RDWR)
			if err != nil {
				t.Fatalf("Failed to iso9660.OpenFile(%s): %v", filename, err)
			}
			if _, err = isofile.Write(contents); err != nil {
				t.Fatalf("%d: error writing bytes to tmpfile %s: %v", i, filename, err)
			}
		}

		err = fs.Finalize(iso9660.FinalizeOptions{RockRidge: true})
		if err != nil {
			t.Fatal("Unexpected error fs.Finalize({RockRidge: true})", err)
		}
		// now need to check contents
		fi, err := f.Stat()
		if err != nil {
			t.Fatalf("Error trying to Stat() iso file: %v", err)
		}
		// we made two 5MB files, so should be at least 10MB
		if fi.Size() < 10*1024*1024 {
			t.Fatalf("Resultant file too small after finalizing %d", fi.Size())
		}

		// now check the contents
		fs, err = iso9660.Read(f, 0, 0, 2048)
		if err != nil {
			t.Fatalf("error reading the tmpfile as iso: %v", err)
		}

		dirFi, err := fs.ReadDir("/")
		if err != nil {
			t.Errorf("error reading the root directory from iso: %v", err)
		}
		// we expect to have 3 entries: ABC BAR and FOO
		expected := map[string]bool{
			"abc": false, "bar": false, "foo": false, "README.md": false,
		}
		for _, e := range dirFi {
			delete(expected, e.Name())
		}
		if len(expected) > 0 {
			keys := make([]string, 0)
			for k := range expected {
				keys = append(keys, k)
			}
			t.Errorf("Some entries not found in root: %v", keys)
		}

		// get a few files I expect
		fileContents := map[string]string{
			"/README.md":       "readme\n",
			"/foo/filename_50": "filename_50\n",
			"/foo/filename_2":  "filename_2\n",
		}

		for k, v := range fileContents {
			var (
				f    filesystem.File
				read int
			)

			f, err = fs.OpenFile(k, os.O_RDONLY)
			if err != nil {
				t.Errorf("Error opening file %s: %v", k, err)
				continue
			}
			// check the contents
			b := make([]byte, 50, 50)
			read, err = f.Read(b)
			if err != nil && err != io.EOF {
				t.Errorf("Error reading from file %s: %v", k, err)
			}
			actual := string(b[:read])
			if actual != v {
				t.Errorf("Mismatched content, actual '%s' expected '%s'", actual, v)
			}
		}

		validateIso(t, f)

		// close the file
		err = f.Close()
		if err != nil {
			t.Fatalf("Could not close iso file: %v", err)
		}
	})
}

func validateIso(t *testing.T, f *os.File) {
	// only do this test if os.Getenv("TEST_IMAGE") contains a real image for integration testing
	if intImage == "" {
		return
	}
	output := new(bytes.Buffer)
	f.Seek(0, 0)
	/* to check file contents
	7z l -ba file.iso
	*/
	err := testhelper.DockerRun(f, output, false, true, intImage, "7z", "l", "-ba", "/file.img")
	outString := output.String()
	if err != nil {
		t.Errorf("Unexpected err: %v", err)
		t.Log(outString)
	}
}

func validateElTorito(t *testing.T, f *os.File) {
	// only do this test if os.Getenv("TEST_IMAGE") contains a real image for integration testing
	if intImage == "" {
		return
	}
	output := new(bytes.Buffer)
	f.Seek(0, 0)
	err := testhelper.DockerRun(f, output, false, true, intImage, "isoinfo", "-d", "-i", "/file.img")
	outString := output.String()
	if err != nil {
		t.Errorf("Unexpected err: %v", err)
		t.Log(outString)
	}
	// look for El Torito line
	re := regexp.MustCompile(`El Torito VD version 1 found, boot catalog is in sector (\d+)\n`)
	matches := re.FindStringSubmatch(outString)
	if matches == nil || len(matches) < 1 {
		t.Fatalf("Unable to match El Torito information")
	}
	// what sector should it be in?
}
