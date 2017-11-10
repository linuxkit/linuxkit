package datakit

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"context"
)

type Version int

var InitialVersion = Version(0)

// Record is a typed view on top of a database branch
type Record struct {
	client    *Client
	path      []string // directory inside the store
	version   Version
	schemaF   *IntField
	fields    []*StringRefField // registered fields, for schema upgrades
	lookupB   []string          // priority ordered list of branches to look up values in
	defaultsB string            // name of the branch containing built-in defaults
	stateB    string            // name of the branch containing run-time state
	event     chan (interface{})
	onUpdate  [](func([]*Snapshot, Version))
}

func NewRecord(ctx context.Context, client *Client, lookupB []string, defaultsB string, stateB string, path []string) (*Record, error) {
	event := make(chan (interface{}), 0)
	for _, b := range append(lookupB, stateB) {
		// Create the branch if it doesn't exist
		t, err := NewTransaction(ctx, client, b)
		if err != nil {
			log.Fatalf("Failed to open a new transaction: %#v", err)
		}
		if err = t.Write(ctx, []string{"branch-created"}, ""); err != nil {
			log.Fatalf("Failed to write branch-created: %#v", err)
		}
		if err = t.Commit(ctx, "Creating branch"); err != nil {
			log.Fatalf("Failed to commit transaction: %#v", err)
		}
	}
	for _, b := range lookupB {
		if err := client.Mkdir(ctx, "branch", b); err != nil {
			return nil, err
		}
		w, err := NewWatch(ctx, client, b, path)
		if err != nil {
			return nil, err
		}
		go func() {
			for {
				_, err := w.Next(ctx)
				if err != nil {
					return
				}
				log.Printf("Snapshot has changed\n")
				event <- 0
			}
		}()
	}
	onUpdate := make([](func([]*Snapshot, Version)), 0)
	fields := make([]*StringRefField, 0)
	r := &Record{
		client:    client,
		path:      path,
		version:   InitialVersion,
		fields:    fields,
		lookupB:   lookupB,
		defaultsB: defaultsB,
		stateB:    stateB,
		event:     event,
		onUpdate:  onUpdate,
	}
	r.schemaF = r.IntField("schema-version", 1)
	return r, nil
}

func (r *Record) updateAll(ctx context.Context) error {
	snapshots := make([]*Snapshot, 0)
	for _, b := range r.lookupB {
		head, err := Head(ctx, r.client, b)
		if err != nil {
			return err
		}
		snap := NewSnapshot(ctx, r.client, COMMIT, head)
		snapshots = append(snapshots, snap)
	}
	for _, fn := range r.onUpdate {
		fn(snapshots, r.version)
	}
	return nil
}

func (r *Record) Seal(ctx context.Context) error {
	return r.updateAll(ctx)
}

func (r *Record) Wait(ctx context.Context) error {
	<-r.event
	r.version = r.version + 1
	return r.updateAll(ctx)
}

func (r *Record) Upgrade(ctx context.Context, schemaVersion int) error {
	currentVersion, _ := r.schemaF.Get()
	if schemaVersion <= currentVersion {
		log.Printf("No schema upgrade necessary because new version (%d) <= current version (%d)\n", schemaVersion, currentVersion)
		return nil
	}
	r.schemaF.defaultInt = schemaVersion
	defaultString := fmt.Sprintf("%d", schemaVersion)
	r.schemaF.raw.defaultValue = &defaultString
	// Create defaults branch
	log.Printf("Performing schema upgrade to version %d\n", schemaVersion)
	t, err := NewTransaction(ctx, r.client, r.defaultsB)
	if err != nil {
		return err
	}
	// For each known field, write default value to branch
	for _, f := range r.fields {
		p := append(r.path, f.path...)
		if f.defaultValue == nil {
			err = t.Remove(ctx, p)
		} else {
			err = t.Write(ctx, p, *f.defaultValue)
		}
		if err != nil {
			return err
		}
	}

	// Merge to the defaults branch
	err = t.Commit(ctx, fmt.Sprintf("Upgrade to schema version %d", schemaVersion))
	if err != nil {
		return err
	}
	return r.Wait(ctx)
}

// fillInDefault updates the default branch to contain the new value.
func (r *Record) fillInDefault(path []string, valueref *string) error {
	ctx := context.Background()
	t, err := NewTransaction(ctx, r.client, r.defaultsB)
	if err != nil {
		return err
	}
	p := append(r.path, path...)
	if valueref == nil {
		log.Printf("Removing default value at %s", strings.Join(p, "/"))

		if err = t.Remove(ctx, p); err != nil {
			log.Printf("Failed to remove key at %s", strings.Join(p, "/"))
			return err
		}
	} else {
		log.Printf("Updating default value at %s to %s", strings.Join(p, "/"), *valueref)
		if err = t.Write(ctx, p, *valueref); err != nil {
			log.Printf("Failed to write key %s = %s", strings.Join(p, "/"), *valueref)
			return err
		}
	}
	return t.Commit(ctx, fmt.Sprintf("fill-in default for %s", p))
}

func (r *Record) SetMultiple(description string, fields []*StringField, values []string) error {
	if len(fields) != len(values) {
		return fmt.Errorf("Length of fields and values is not equal")
	}
	ctx := context.Background()
	t, err := NewTransaction(ctx, r.client, r.lookupB[0])
	if err != nil {
		return err
	}
	for i, k := range fields {
		p := append(r.path, k.raw.path...)
		v := values[i]
		log.Printf("Setting value in store: %#v=%s\n", p, v)
		err = t.Write(ctx, p, v)
		if err != nil {
			return err
		}
	}
	return t.Commit(ctx, description)
}

type StringRefField struct {
	path         []string
	value        *string
	defaultValue *string
	version      Version // version of last change
	record       *Record
}

// Set unconditionally sets the value of the key
func (f *StringRefField) Set(description string, value *string) error {
	// TODO: maybe this should return Version, too?
	ctx := context.Background()
	p := append(f.record.path, f.path...)
	log.Printf("Setting value in store: %#v=%#v\n", p, value)
	t, err := NewTransaction(ctx, f.record.client, f.record.lookupB[0])
	if err != nil {
		return err
	}
	if value == nil {
		err = t.Remove(ctx, p)
	} else {
		err = t.Write(ctx, p, *value)
	}
	if err != nil {
		return err
	}
	return t.Commit(ctx, fmt.Sprintf("Unconditionally set %s: %s", f.path, description))
}

// Get retrieves the current value of the key
func (f *StringRefField) Get() (*string, Version) {
	if f.value == nil {
		return nil, f.version
	}
	raw := strings.TrimSpace(*f.value)
	return &raw, f.version
}

// HasChanged returns true if the key has changed since the given version
func (f *StringRefField) HasChanged(version Version) bool {
	return version < f.version
}

// StringRefField defines a string option which can be nil with a specified
// key and default value
func (f *Record) StringRefField(key string, value *string) *StringRefField {
	path := strings.Split(key, "/")
	field := &StringRefField{path: path, value: value, defaultValue: value, version: InitialVersion, record: f}
	// If the value is not in the database, write the default Value.
	err := f.fillInDefault(path, value)
	if err != nil {
		log.Println("Failed to write default value", key, "=", value)
	}
	fn := func(snaps []*Snapshot, version Version) {
		ctx := context.Background()
		var newValue *string
		for _, snap := range snaps {
			v, err := snap.Read(ctx, append(f.path, path...))
			if err != nil {
				if err != enoent {
					log.Println("Failed to read key", key, "from directory snapshot", snap)
					return
				}
				// if enoent then newValue == nil
			} else {
				newValue = &v
				break
			}
		}
		if (field.value == nil && newValue != nil) || (field.value != nil && newValue == nil) || (field.value != nil && newValue != nil && *field.value != *newValue) {
			field.value = newValue
			field.version = version
		}

		// Update the value in memory and in the state branch
		t, err := NewTransaction(ctx, f.client, f.stateB)
		if err != nil {
			log.Fatalf("Failed to create transaction for updating state branch: %#v", err)
		}
		if newValue != nil {
			if err = t.Write(ctx, append(f.path, path...), *newValue); err != nil {
				log.Fatalf("Failed to write state %#v = %s: %#v", path, *newValue, err)
			}
		} else {
			if err = t.Remove(ctx, append(f.path, path...)); err != nil {
				log.Fatalf("Failed to remove state %#v: %#v", path, err)
			}
		}
		if err = t.Commit(ctx, "Updating state branch"); err != nil {
			log.Fatalf("Failed to commit transaction: %#v", err)
		}

	}
	f.onUpdate = append(f.onUpdate, fn)
	//fn(f.version)
	f.fields = append(f.fields, field)
	return field
}

type StringField struct {
	raw           *StringRefField
	defaultString string
}

// Get retrieves the current value of the key
func (f *StringField) Get() (string, Version) {
	if f.raw.value == nil {
		log.Printf("Failed to find string in database at %s, defaulting to %s", strings.Join(f.raw.path, "/"), f.defaultString)
		return f.defaultString, f.raw.version
	}
	return *f.raw.value, f.raw.version
}

// Set unconditionally sets the value of the key
func (f *StringField) Set(description string, value string) error {
	return f.raw.Set(description, &value)
}

// HasChanged returns true if the key has changed since the given version
func (f *StringField) HasChanged(version Version) bool {
	return version < f.raw.version
}

// StringField defines a string
func (f *Record) StringField(key string, value string) *StringField {
	raw := f.StringRefField(key, &value)
	return &StringField{raw: raw, defaultString: value}
}

type IntField struct {
	raw        *StringRefField
	defaultInt int
}

// Get retrieves the current value of the key
func (f *IntField) Get() (int, Version) {
	if f.raw.value == nil {
		log.Printf("Key %s missing in database, defaulting value to %t", strings.Join(f.raw.path, "/"), f.defaultInt)
		return f.defaultInt, f.raw.version
	}
	value64, err := strconv.ParseInt(strings.TrimSpace(*f.raw.value), 10, 0)
	if err != nil {
		// revert to default if we can't parse the result
		log.Printf("Failed to parse int in database: '%s', defaulting to %d", f.raw.value, f.defaultInt)
		return f.defaultInt, f.raw.version
	}
	return int(value64), f.raw.version
}

// HasChanged returns true if the key has changed since the given version
func (f *IntField) HasChanged(version Version) bool {
	return version < f.raw.version
}

// IntField defines an boolean option with a specified key and default value
func (f *Record) IntField(key string, value int) *IntField {
	stringValue := fmt.Sprintf("%d", value)
	raw := f.StringRefField(key, &stringValue)
	return &IntField{raw: raw, defaultInt: value}
}

type BoolField struct {
	raw         *StringRefField
	defaultBool bool
}

// Get retrieves the current value of the key
func (f *BoolField) Get() (bool, Version) {
	if f.raw.value == nil {
		log.Printf("Key %s missing in database, defaulting value to %t", strings.Join(f.raw.path, "/"), f.defaultBool)
		return f.defaultBool, f.raw.version
	}
	value, err := strconv.ParseBool(strings.TrimSpace(*f.raw.value))
	if err != nil {
		// revert to default if we can't parse the result
		log.Printf("Failed to parse boolean in database: '%s', defaulting to %t", f.raw.value, f.defaultBool)
		return f.defaultBool, f.raw.version
	}
	return value, f.raw.version
}

// HasChanged returns true if the key has changed since the given version
func (f *BoolField) HasChanged(version Version) bool {
	return version < f.raw.version
}

// BoolField defines an boolean option with a specified key and default value
func (f *Record) BoolField(key string, value bool) *BoolField {
	stringValue := fmt.Sprintf("%t", value)
	raw := f.StringRefField(key, &stringValue)
	return &BoolField{raw: raw, defaultBool: value}
}
