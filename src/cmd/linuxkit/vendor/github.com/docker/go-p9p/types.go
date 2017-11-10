package p9p

import (
	"fmt"
	"time"
)

const (
	// DefaultMSize messages size used to establish a session.
	DefaultMSize = 64 << 10

	// DefaultVersion for this package. Currently, the only supported version.
	DefaultVersion = "9P2000"
)

// Mode constants for use Dir.Mode.
const (
	DMDIR    = 0x80000000 // mode bit for directories
	DMAPPEND = 0x40000000 // mode bit for append only files
	DMEXCL   = 0x20000000 // mode bit for exclusive use files
	DMMOUNT  = 0x10000000 // mode bit for mounted channel
	DMAUTH   = 0x08000000 // mode bit for authentication file
	DMTMP    = 0x04000000 // mode bit for non-backed-up files

	// 9p2000.u extensions

	DMSYMLINK   = 0x02000000
	DMDEVICE    = 0x00800000
	DMNAMEDPIPE = 0x00200000
	DMSOCKET    = 0x00100000
	DMSETUID    = 0x00080000
	DMSETGID    = 0x00040000

	DMREAD  = 0x4 // mode bit for read permission
	DMWRITE = 0x2 // mode bit for write permission
	DMEXEC  = 0x1 // mode bit for execute permission
)

// Flag defines the flag type for use with open and create
type Flag uint8

// Constants to use when opening files.
const (
	OREAD  Flag = 0x00 // open for read
	OWRITE Flag = 0x01 // write
	ORDWR  Flag = 0x02 // read and write
	OEXEC  Flag = 0x03 // execute, == read but check execute permission

	// PROPOSAL(stevvooe): Possible protocal extension to allow the create of
	// symlinks. Initially, the link is created with no value. Read and write
	// to read and set the link value.

	OSYMLINK Flag = 0x04

	OTRUNC  Flag = 0x10 // or'ed in (except for exec), truncate file first
	OCEXEC  Flag = 0x20 // or'ed in, close on exec
	ORCLOSE Flag = 0x40 // or'ed in, remove on close
)

// QType indicates the type of a resource within the Qid.
type QType uint8

// Constants for use in Qid to indicate resource type.
const (
	QTDIR    QType = 0x80 // type bit for directories
	QTAPPEND QType = 0x40 // type bit for append only files
	QTEXCL   QType = 0x20 // type bit for exclusive use files
	QTMOUNT  QType = 0x10 // type bit for mounted channel
	QTAUTH   QType = 0x08 // type bit for authentication file
	QTTMP    QType = 0x04 // type bit for not-backed-up file
	QTFILE   QType = 0x00 // plain file
)

func (qt QType) String() string {
	switch qt {
	case QTDIR:
		return "dir"
	case QTAPPEND:
		return "append"
	case QTEXCL:
		return "excl"
	case QTMOUNT:
		return "mount"
	case QTAUTH:
		return "auth"
	case QTTMP:
		return "tmp"
	case QTFILE:
		return "file"
	}

	return "unknown"
}

// Tag uniquely identifies an outstanding fcall in a 9p session.
type Tag uint16

// NOTAG is a reserved values for messages sent before establishing a session,
// such as Tversion.
const NOTAG Tag = ^Tag(0)

// Fid defines a type to hold Fid values.
type Fid uint32

// NOFID indicates the lack of an Fid.
const NOFID Fid = ^Fid(0)

// Qid indicates the type, path and version of the resource returned by a
// server. It is only valid for a session.
//
// Typically, a client maintains a mapping of Fid-Qid as Qids are returned by
// the server.
type Qid struct {
	Type    QType `9p:"type,1"`
	Version uint32
	Path    uint64
}

func (qid Qid) String() string {
	return fmt.Sprintf("qid(%v, v=%x, p=%x)",
		qid.Type, qid.Version, qid.Path)
}

// Dir defines the structure used for expressing resources in stat/wstat and
// when reading directories.
type Dir struct {
	Type uint16
	Dev  uint32
	Qid  Qid
	Mode uint32

	// BUG(stevvooe): The Year 2038 is coming soon. 9p wire protocol has these
	// as 4 byte epoch times. Some possibilities include time dilation fields
	// or atemporal files. We can also just not use them and set them to zero.

	AccessTime time.Time
	ModTime    time.Time

	Length uint64
	Name   string
	UID    string
	GID    string
	MUID   string
}

func (d Dir) String() string {
	return fmt.Sprintf("dir(%v mode=%v atime=%v mtime=%v length=%v name=%v uid=%v gid=%v muid=%v)",
		d.Qid, d.Mode, d.AccessTime, d.ModTime, d.Length, d.Name, d.UID, d.GID, d.MUID)
}
