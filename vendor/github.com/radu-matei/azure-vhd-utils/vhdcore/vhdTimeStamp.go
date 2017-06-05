package vhdcore

import (
	"log"
	"time"
)

// TimeStamp represents the the creation time of a hard disk image. This is the number
// of seconds since vhd base time which is January 1, 2000 12:00:00 AM in UTC/GMT.
// The disk creation time is stored in the vhd footer in big-endian format.
//
type TimeStamp struct {
	TotalSeconds uint32
}

// NewVhdTimeStamp creates new VhdTimeStamp with creation time as dateTime. This function
// will panic if the given datetime is before the vhd base time.
//
func NewVhdTimeStamp(dateTime *time.Time) *TimeStamp {
	vhdBaseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if !dateTime.After(vhdBaseTime) {
		log.Panicf("DateTime must be after Base Vhd Time: %v", dateTime)
	}

	return &TimeStamp{
		TotalSeconds: uint32(dateTime.Sub(vhdBaseTime).Seconds()),
	}
}

// NewVhdTimeStampFromSeconds creates new VhdTimeStamp, creation time is calculated by adding
// given total seconds with the vhd base time.
//
func NewVhdTimeStampFromSeconds(totalSeconds uint32) *TimeStamp {
	return &TimeStamp{TotalSeconds: totalSeconds}
}

// ToDateTime returns the time.Time representation of this instance.
//
func (v *TimeStamp) ToDateTime() time.Time {
	vhdBaseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return vhdBaseTime.Add(time.Duration(v.TotalSeconds) * time.Second)
}
