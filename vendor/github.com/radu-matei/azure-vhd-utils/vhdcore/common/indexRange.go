package common

import (
	"fmt"
	"sort"
)

// IndexRange represents sequence of integral numbers in a specified range, where range starts
// at Start and ends at End, inclusive
//
type IndexRange struct {
	Start int64
	End   int64
}

// NewIndexRange creates a new range with start as value of the first integer in the sequence
// and end as value of last integer in the sequence.
//
func NewIndexRange(start, end int64) *IndexRange {
	return &IndexRange{Start: start, End: end}
}

// NewIndexRangeFromLength creates a new range starting from start and ends at start + length - 1.
//
func NewIndexRangeFromLength(start, length int64) *IndexRange {
	return NewIndexRange(start, start+length-1)
}

// TotalRangeLength returns the total length of a given slice of ranges.
//
func TotalRangeLength(ranges []*IndexRange) int64 {
	var length = int64(0)
	for _, r := range ranges {
		length += r.Length()
	}
	return length
}

// SubtractRanges produces a set of ranges, each subset of ranges in this set is produced by
// subtracting subtrahends from each range in minuends.
//
func SubtractRanges(minuends, subtrahends []*IndexRange) []*IndexRange {
	var result = make([]*IndexRange, 0)
	for _, minuend := range minuends {
		result = minuend.SubtractRanges(subtrahends, false, result)
	}

	return sortAndDedup(result)
}

// ChunkRangesBySize produces a set of ranges by partitioning the ranges in the given ranges by
// the given partition-size.
// Each each range in the given ranges X will be partitioned by the given partition-size to produce
// a range set A. If the last range in A is not of partition-size and if it is adjacent to the
// next range in the X then we calculate the bytes required to reach partition-size and
//    1. if next range has more bytes than required, then we borrow the required bytes from next
//       range and advances the next range start
//    2. if next range has less or equal to the required bytes, then we borrow available and skip
//       next range
//
func ChunkRangesBySize(ranges []*IndexRange, chunkSizeInBytes int64) []*IndexRange {
	var chunks = make([]*IndexRange, 0)
	length := len(ranges)
	var remaining *IndexRange

	for i, current := range ranges {
		if remaining != nil {
			if remaining.Adjacent(current) {
				requiredBytes := chunkSizeInBytes - remaining.Length()
				availableBytes := current.Length()
				if requiredBytes < availableBytes {
					remaining.End += requiredBytes
					current.Start += requiredBytes
					chunks = append(chunks, remaining)
					remaining = nil
				} else {
					remaining.End += availableBytes
					current = nil
				}
			} else {
				chunks = append(chunks, remaining)
				remaining = nil
			}
		}

		if current != nil {
			chunksSet := current.PartitionBy(chunkSizeInBytes)

			lastChunkIndex := len(chunksSet) - 1
			lastChunk := chunksSet[lastChunkIndex]
			if (lastChunk.Length() != chunkSizeInBytes) && (i+1 < length) && lastChunk.Adjacent(ranges[i+1]) {
				remaining = lastChunk
				chunks = append(chunks, chunksSet[:lastChunkIndex]...)
			} else {
				chunks = append(chunks, chunksSet...)
			}
		}
	}

	if remaining != nil {
		chunks = append(chunks, remaining)
	}

	return chunks
}

// Length returns number of sequential integers in the range.
//
func (ir *IndexRange) Length() int64 {
	return ir.End - ir.Start + 1
}

// Equals returns true if this and given range represents the same sequence, two sequences
// are same if both have the same start and end.
//
func (ir *IndexRange) Equals(other *IndexRange) bool {
	return other != nil && ir.Start == other.Start && ir.End == other.End
}

// CompareTo indicates whether the this range precedes, follows, or occurs in the same
// position in the sort order as the other
// A return value
//   Less than zero:    This range precedes the other in the sort order, range A precedes
//                      range B if A start before B or both has the same start and A ends
//                      before B.
//   Zero:              This range occurs in the same position as other in sort order, two
//                      ranges are in the same sort position if both has the same start
//                      and end
//   Greater than zero: This range follows the other in the sort order, a range A follows
//                      range B, if A start after B or both has the same start and A ends
//                      after B
//
func (ir *IndexRange) CompareTo(other *IndexRange) int64 {
	r := ir.Start - other.Start
	if r != 0 {
		return r
	}

	return ir.End - other.End
}

// Intersects checks this and other range intersects, two ranges A and B intersects if either
// of them starts or ends within the range of other, inclusive.
//
func (ir *IndexRange) Intersects(other *IndexRange) bool {
	start := ir.Start
	if start < other.Start {
		start = other.Start
	}

	end := ir.End
	if end > other.End {
		end = other.End
	}

	return start <= end
}

// Intersection computes the range representing the intersection of two ranges, a return
// value nil indicates the ranges does not intersects.
//
func (ir *IndexRange) Intersection(other *IndexRange) *IndexRange {
	start := ir.Start
	if start < other.Start {
		start = other.Start
	}

	end := ir.End
	if end > other.End {
		end = other.End
	}

	if start > end {
		return nil
	}

	return NewIndexRange(start, end)
}

// Includes checks this range includes the other range, a range A includes range B if B starts
// and ends within A, inclusive. In other words a range A includes range B if their intersection
// produces B
//
func (ir *IndexRange) Includes(other *IndexRange) bool {
	if other.Start < ir.Start {
		return false
	}

	if other.End > ir.End {
		return false
	}

	return true
}

// Gap compute the range representing the gap between this and the other range, a return value
// nil indicates there is no gap because either the ranges intersects or they are adjacent.
//
func (ir *IndexRange) Gap(other *IndexRange) *IndexRange {
	if ir.Intersects(other) {
		return nil
	}

	r := ir.CompareTo(other)
	if r < 0 {
		g := NewIndexRange(ir.End+1, other.Start-1)
		if g.Length() <= 0 {
			return nil
		}
		return g
	}

	g := NewIndexRange(other.End+1, ir.Start-1)
	if g.Length() <= 0 {
		return nil
	}
	return g
}

// Adjacent checks this range starts immediately starts after the other range or vice-versa,
// a return value nil indicates the ranges intersects or there is a gap between the ranges.
//
func (ir *IndexRange) Adjacent(other *IndexRange) bool {
	return !ir.Intersects(other) && ir.Gap(other) == nil
}

// Subtract subtracts other range from this range and appends the ranges representing the
// differences to result slice.
//
// Given two ranges A and B, A - B produces
// 1. No result
//      a. If they are equal or
//      b. B includes A i.e 'A n B' = A
//  OR
// 2. A, if they don't intersects
//  OR
// 3. [(A n B).End + 1, A.End],     if A and 'A n B' has same start
//  OR
// 4. [A.Start, (A n B).Start - 1], if A and 'A n B' has same end
//  OR
// 5. { [A.Start, (A n B).Start - 1], [(A n B).End + 1, A.End] }, otherwise
//
func (ir *IndexRange) Subtract(other *IndexRange, result []*IndexRange) []*IndexRange {
	if ir.Equals(other) {
		return result
	}

	if !ir.Intersects(other) {
		result = append(result, NewIndexRange(ir.Start, ir.End))
		return result
	}

	in := ir.Intersection(other)
	if ir.Equals(in) {
		return result
	}

	if in.Start == ir.Start {
		result = append(result, NewIndexRange(in.End+1, ir.End))
		return result
	}

	if in.End == ir.End {
		result = append(result, NewIndexRange(ir.Start, in.Start-1))
		return result
	}

	result = append(result, NewIndexRange(ir.Start, in.Start-1))
	result = append(result, NewIndexRange(in.End+1, ir.End))
	return result
}

// SubtractRanges subtracts a set of ranges from this range and appends the ranges representing
// the differences to result slice. The result slice will be sorted and de-duped if sortandDedup
// is true.
//
func (ir *IndexRange) SubtractRanges(ranges []*IndexRange, sortandDedup bool, result []*IndexRange) []*IndexRange {
	intersectAny := false
	for _, o := range ranges {
		if ir.Intersects(o) {
			result = ir.Subtract(o, result)
			intersectAny = true
		}
	}

	if !intersectAny {
		result = append(result, NewIndexRange(ir.Start, ir.End))
	}

	if !sortandDedup {
		return result
	}

	return sortAndDedup(result)
}

// Merge produces a range by merging this and other range if they are adjacent. Trying to merge
// non-adjacent ranges are panic.
//
func (ir *IndexRange) Merge(other *IndexRange) *IndexRange {
	if !ir.Adjacent(other) {
		// TODO: error
	}

	if ir.CompareTo(other) < 0 {
		return NewIndexRange(ir.Start, other.End)
	}

	return NewIndexRange(other.Start, ir.End)
}

// PartitionBy produces a slice of adjacent ranges of same size, first range in the slice starts
// where this range starts and last range ends where this range ends. The length of last range will
// be less than size if length of this range is not multiple of size.
//
func (ir *IndexRange) PartitionBy(size int64) []*IndexRange {
	length := ir.Length()
	if length <= size {
		return []*IndexRange{NewIndexRange(ir.Start, ir.End)}
	}

	blocks := length / size
	r := make([]*IndexRange, blocks+1)
	for i := int64(0); i < blocks; i++ {
		r[i] = NewIndexRangeFromLength(ir.Start+i*size, size)
	}

	reminder := length % size
	if reminder != 0 {
		r[blocks] = NewIndexRangeFromLength(ir.Start+blocks*size, reminder)
		return r
	}

	return r[:blocks]
}

// String returns the string representation of this range, this satisfies stringer interface.
//
func (ir *IndexRange) String() string {
	return fmt.Sprintf("{%d, %d}", ir.Start, ir.End)
}

// sortAndDedup sorts the given range slice in place, remove the duplicates from the sorted slice
// and returns the updated slice.
//
func sortAndDedup(indexRanges []*IndexRange) []*IndexRange {
	if len(indexRanges) == 0 {
		return indexRanges
	}
	sort.Sort(indexRangeSorter(indexRanges))
	i := 0
	for j := 1; j < len(indexRanges); j++ {
		if !indexRanges[i].Equals(indexRanges[j]) {
			i++
			indexRanges[i] = indexRanges[j]
		}
	}
	return indexRanges[:i+1]
}

// indexRangeSorter is a type that satisfies sort.Interface interface for supporting sorting of
// a IndexRange collection.
//
type indexRangeSorter []*IndexRange

// Len is the number of elements in the range collection.
//
func (s indexRangeSorter) Len() int {
	return len(s)
}

// Less reports whether range at i-th position precedes the range at j-th position in sort order.
// range A precedes range B if A start before B or both has the same start and A ends before B.
//
func (s indexRangeSorter) Less(i, j int) bool {
	return s[i].CompareTo(s[j]) < 0
}

// Swap swaps the elements with indexes i and j.
//
func (s indexRangeSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
