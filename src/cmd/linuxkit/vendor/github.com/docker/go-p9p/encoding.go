package p9p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"
)

// Codec defines the interface for encoding and decoding of 9p types.
// Unsupported types will throw an error.
type Codec interface {
	// Unmarshal from data into the value pointed to by v.
	Unmarshal(data []byte, v interface{}) error

	// Marshal the value v into a byte slice.
	Marshal(v interface{}) ([]byte, error)

	// Size returns the encoded size for the target of v.
	Size(v interface{}) int
}

// NewCodec returns a new, standard 9P2000 codec, ready for use.
func NewCodec() Codec {
	return codec9p{}
}

type codec9p struct{}

func (c codec9p) Unmarshal(data []byte, v interface{}) error {
	dec := &decoder{bytes.NewReader(data)}
	return dec.decode(v)
}

func (c codec9p) Marshal(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	enc := &encoder{&b}

	if err := enc.encode(v); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (c codec9p) Size(v interface{}) int {
	return int(size9p(v))
}

// DecodeDir decodes a directory entry from rd using the provided codec.
func DecodeDir(codec Codec, rd io.Reader, d *Dir) error {
	var ll uint16

	// pull the size off the wire
	if err := binary.Read(rd, binary.LittleEndian, &ll); err != nil {
		return err
	}

	p := make([]byte, ll+2)
	binary.LittleEndian.PutUint16(p, ll) // must have size at start

	// read out the rest of the record
	if _, err := io.ReadFull(rd, p[2:]); err != nil {
		return err
	}

	return codec.Unmarshal(p, d)
}

// EncodeDir writes the directory to wr.
func EncodeDir(codec Codec, wr io.Writer, d *Dir) error {
	p, err := codec.Marshal(d)
	if err != nil {
		return err
	}

	_, err = wr.Write(p)
	return err
}

type encoder struct {
	wr io.Writer
}

func (e *encoder) encode(vs ...interface{}) error {
	for _, v := range vs {
		switch v := v.(type) {
		case uint8, uint16, uint32, uint64, FcallType, Tag, QType, Fid, Flag,
			*uint8, *uint16, *uint32, *uint64, *FcallType, *Tag, *QType, *Fid, *Flag:
			if err := binary.Write(e.wr, binary.LittleEndian, v); err != nil {
				return err
			}
		case []byte:
			if err := e.encode(uint32(len(v))); err != nil {
				return err
			}

			if err := binary.Write(e.wr, binary.LittleEndian, v); err != nil {
				return err
			}

		case *[]byte:
			if err := e.encode(*v); err != nil {
				return err
			}
		case string:
			if err := binary.Write(e.wr, binary.LittleEndian, uint16(len(v))); err != nil {
				return err
			}

			_, err := io.WriteString(e.wr, v)
			if err != nil {
				return err
			}
		case *string:
			if err := e.encode(*v); err != nil {
				return err
			}

		case []string:
			if err := e.encode(uint16(len(v))); err != nil {
				return err
			}

			for _, m := range v {
				if err := e.encode(m); err != nil {
					return err
				}
			}
		case *[]string:
			if err := e.encode(*v); err != nil {
				return err
			}
		case time.Time:
			if err := e.encode(uint32(v.Unix())); err != nil {
				return err
			}
		case *time.Time:
			if err := e.encode(*v); err != nil {
				return err
			}
		case Qid:
			if err := e.encode(v.Type, v.Version, v.Path); err != nil {
				return err
			}
		case *Qid:
			if err := e.encode(*v); err != nil {
				return err
			}
		case []Qid:
			if err := e.encode(uint16(len(v))); err != nil {
				return err
			}

			elements := make([]interface{}, len(v))
			for i := range v {
				elements[i] = &v[i]
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *[]Qid:
			if err := e.encode(*v); err != nil {
				return err
			}
		case Dir:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			if err := e.encode(uint16(size9p(elements...))); err != nil {
				return err
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *Dir:
			if err := e.encode(*v); err != nil {
				return err
			}
		case []Dir:
			elements := make([]interface{}, len(v))
			for i := range v {
				elements[i] = &v[i]
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		case *[]Dir:
			if err := e.encode(*v); err != nil {
				return err
			}
		case Fcall:
			if err := e.encode(v.Type, v.Tag, v.Message); err != nil {
				return err
			}
		case *Fcall:
			if err := e.encode(*v); err != nil {
				return err
			}
		case Message:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			switch v.(type) {
			case MessageRstat, *MessageRstat:
				// NOTE(stevvooe): Prepend size preceeding Dir. See bugs in
				// http://man.cat-v.org/plan_9/5/stat to make sense of this.
				// The field has been included here but we need to make sure
				// to double emit it for Rstat.
				if err := e.encode(uint16(size9p(elements...))); err != nil {
					return err
				}
			}

			if err := e.encode(elements...); err != nil {
				return err
			}
		}
	}

	return nil
}

type decoder struct {
	rd io.Reader
}

// read9p extracts values from rd and unmarshals them to the targets of vs.
func (d *decoder) decode(vs ...interface{}) error {
	for _, v := range vs {
		switch v := v.(type) {
		case *uint8, *uint16, *uint32, *uint64, *FcallType, *Tag, *QType, *Fid, *Flag:
			if err := binary.Read(d.rd, binary.LittleEndian, v); err != nil {
				return err
			}
		case *[]byte:
			var ll uint32

			if err := d.decode(&ll); err != nil {
				return err
			}

			if ll > 0 {
				*v = make([]byte, int(ll))
			}

			if err := binary.Read(d.rd, binary.LittleEndian, v); err != nil {
				return err
			}
		case *string:
			var ll uint16

			// implement string[s] encoding
			if err := d.decode(&ll); err != nil {
				return err
			}

			b := make([]byte, ll)

			n, err := io.ReadFull(d.rd, b)
			if err != nil {
				return err
			}

			if n != int(ll) {
				return fmt.Errorf("unexpected string length")
			}

			*v = string(b)
		case *[]string:
			var ll uint16

			if err := d.decode(&ll); err != nil {
				return err
			}

			elements := make([]interface{}, int(ll))
			*v = make([]string, int(ll))
			for i := range elements {
				elements[i] = &(*v)[i]
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		case *time.Time:
			var epoch uint32
			if err := d.decode(&epoch); err != nil {
				return err
			}

			*v = time.Unix(int64(epoch), 0).UTC()
		case *Qid:
			if err := d.decode(&v.Type, &v.Version, &v.Path); err != nil {
				return err
			}
		case *[]Qid:
			var ll uint16

			if err := d.decode(&ll); err != nil {
				return err
			}

			elements := make([]interface{}, int(ll))
			*v = make([]Qid, int(ll))
			for i := range elements {
				elements[i] = &(*v)[i]
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		case *Dir:
			var ll uint16

			if err := d.decode(&ll); err != nil {
				return err
			}

			b := make([]byte, ll)
			// must consume entire dir entry.
			n, err := io.ReadFull(d.rd, b)
			if err != nil {
				log.Println("dir readfull failed:", err, ll, n)
				return err
			}

			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			dec := &decoder{bytes.NewReader(b)}

			if err := dec.decode(elements...); err != nil {
				return err
			}
		case *[]Dir:
			*v = make([]Dir, 0)
			for {
				element := Dir{}
				if err := d.decode(&element); err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				*v = append(*v, element)
			}
		case *Fcall:
			if err := d.decode(&v.Type, &v.Tag); err != nil {
				return err
			}

			message, err := newMessage(v.Type)
			if err != nil {
				return err
			}

			// NOTE(stevvooe): We do a little pointer dance to allocate the
			// new type, write to it, then assign it back to the interface as
			// a concrete type, avoiding a pointer (the interface) to a
			// pointer.
			rv := reflect.New(reflect.TypeOf(message))
			if err := d.decode(rv.Interface()); err != nil {
				return err
			}

			v.Message = rv.Elem().Interface().(Message)
		case Message:
			elements, err := fields9p(v)
			if err != nil {
				return err
			}

			switch v.(type) {
			case *MessageRstat, MessageRstat:
				// NOTE(stevvooe): Consume extra size preceeding Dir. See bugs
				// in http://man.cat-v.org/plan_9/5/stat to make sense of
				// this. The field has been included here but we need to make
				// sure to double emit it for Rstat. decode extra size header
				// for stat structure.
				var ll uint16
				if err := d.decode(&ll); err != nil {
					return err
				}
			}

			if err := d.decode(elements...); err != nil {
				return err
			}
		}
	}

	return nil
}

// size9p calculates the projected size of the values in vs when encoded into
// 9p binary protocol. If an element or elements are not valid for 9p encoded,
// the value 0 will be used for the size. The error will be detected when
// encoding.
func size9p(vs ...interface{}) uint32 {
	var s uint32
	for _, v := range vs {
		if v == nil {
			continue
		}

		switch v := v.(type) {
		case uint8, uint16, uint32, uint64, FcallType, Tag, QType, Fid, Flag,
			*uint8, *uint16, *uint32, *uint64, *FcallType, *Tag, *QType, *Fid, *Flag:
			s += uint32(binary.Size(v))
		case []byte:
			s += uint32(binary.Size(uint32(0)) + len(v))
		case *[]byte:
			s += size9p(uint32(0), *v)
		case string:
			s += uint32(binary.Size(uint16(0)) + len(v))
		case *string:
			s += size9p(*v)
		case []string:
			s += size9p(uint16(0))

			for _, sv := range v {
				s += size9p(sv)
			}
		case *[]string:
			s += size9p(*v)
		case time.Time, *time.Time:
			// BUG(stevvooe): Y2038 is coming.
			s += size9p(uint32(0))
		case Qid:
			s += size9p(v.Type, v.Version, v.Path)
		case *Qid:
			s += size9p(*v)
		case []Qid:
			s += size9p(uint16(0))
			elements := make([]interface{}, len(v))
			for i := range elements {
				elements[i] = &v[i]
			}
			s += size9p(elements...)
		case *[]Qid:
			s += size9p(*v)

		case Dir:
			// walk the fields of the message to get the total size. we just
			// use the field order from the message struct. We may add tag
			// ignoring if needed.
			elements, err := fields9p(v)
			if err != nil {
				// BUG(stevvooe): The options here are to return 0, panic or
				// make this return an error. Ideally, we make it safe to
				// return 0 and have the rest of the package do the right
				// thing. For now, we do this, but may want to panic until
				// things are stable.
				panic(err)
			}

			s += size9p(elements...) + size9p(uint16(0))
		case *Dir:
			s += size9p(*v)
		case []Dir:
			elements := make([]interface{}, len(v))
			for i := range elements {
				elements[i] = &v[i]
			}
			s += size9p(elements...)
		case *[]Dir:
			s += size9p(*v)
		case Fcall:
			s += size9p(v.Type, v.Tag, v.Message)
		case *Fcall:
			s += size9p(*v)
		case Message:
			// special case twstat and rstat for size fields. See bugs in
			// http://man.cat-v.org/plan_9/5/stat to make sense of this.
			switch v.(type) {
			case *MessageRstat, MessageRstat:
				s += size9p(uint16(0)) // for extra size field before dir
			}

			// walk the fields of the message to get the total size. we just
			// use the field order from the message struct. We may add tag
			// ignoring if needed.
			elements, err := fields9p(v)
			if err != nil {
				// BUG(stevvooe): The options here are to return 0, panic or
				// make this return an error. Ideally, we make it safe to
				// return 0 and have the rest of the package do the right
				// thing. For now, we do this, but may want to panic until
				// things are stable.
				panic(err)
			}

			s += size9p(elements...)
		}
	}

	return s
}

// fields9p lists the settable fields from a struct type for reading and
// writing. We are using a lot of reflection here for fairly static
// serialization but we can replace this in the future with generated code if
// performance is an issue.
func fields9p(v interface{}) ([]interface{}, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot extract fields from non-struct: %v", rv)
	}

	var elements []interface{}
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)

		if !f.CanInterface() {
			// unexported field, skip it.
			continue
		}

		if f.CanAddr() {
			f = f.Addr()
		}

		elements = append(elements, f.Interface())
	}

	return elements, nil
}

func string9p(v interface{}) string {
	if v == nil {
		return "nil"
	}

	rv := reflect.Indirect(reflect.ValueOf(v))

	if rv.Kind() != reflect.Struct {
		panic("not a struct")
	}

	var s string

	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)

		s += fmt.Sprintf(" %v=%v", strings.ToLower(rv.Type().Field(i).Name), f.Interface())
	}

	return s
}
