package moby

import (
	"archive/tar"
	"bytes"
)

// apkTarWriter apk-aware tar writer that consolidates installed database, so that
// it can be called multiple times and will do the union of all such databases,
// rather than overwriting the previous one.
// Useful only for things that write to the base filesystem, i.e. init, since everything
// else is inside containers.
const apkInstalledPath = "lib/apk/db/installed"

type apkTarWriter struct {
	*tar.Writer
	dbs      [][]byte
	current  *bytes.Buffer
	location string
}

func NewAPKTarWriter(w *tar.Writer, location string) *apkTarWriter {
	return &apkTarWriter{
		Writer:   w,
		location: location,
	}
}

func (a *apkTarWriter) WriteHeader(hdr *tar.Header) error {
	if a.current != nil {
		a.dbs = append(a.dbs, a.current.Bytes())
		a.current = nil
	}
	if hdr.Name == apkInstalledPath {
		a.current = new(bytes.Buffer)
	}
	return a.Writer.WriteHeader(hdr)
}
func (a *apkTarWriter) Write(b []byte) (int, error) {
	if a.current != nil {
		a.current.Write(b)
	}
	return a.Writer.Write(b)
}

func (a *apkTarWriter) Close() error {
	// before closing, write out the union of all the databases
	if a.current != nil {
		a.dbs = append(a.dbs, a.current.Bytes())
		a.current = nil
	}
	if err := a.WriteAPKDB(); err != nil {
		return err
	}
	return a.Writer.Close()
}

func (a *apkTarWriter) WriteAPKDB() error {
	if len(a.dbs) > 1 {
		// consolidate the databases
		// calculate the size of the new database
		var size int
		for _, db := range a.dbs {
			size += len(db)
			size += 2 // 2 trailing newlines for each db
		}
		hdr := &tar.Header{
			Name:     apkInstalledPath,
			Mode:     0o644,
			Uid:      0,
			Gid:      0,
			Typeflag: tar.TypeReg,
			Size:     int64(size),
			PAXRecords: map[string]string{
				PaxRecordLinuxkitSource:   "LINUXKIT.apkinit",
				PaxRecordLinuxkitLocation: a.location,
			},
		}
		if err := a.Writer.WriteHeader(hdr); err != nil {
			return err
		}
		for _, db := range a.dbs {
			if _, err := a.Writer.Write(db); err != nil {
				return err
			}
			if _, err := a.Writer.Write([]byte{'\n', '\n'}); err != nil {
				return err
			}
		}
	}
	// once complete, clear the databases
	a.dbs = nil
	return nil
}
