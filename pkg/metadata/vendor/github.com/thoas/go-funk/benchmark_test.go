package funk

import (
	"math/rand"
	"testing"
)

const (
	seed      = 918234565
	sliceSize = 3614562
)

func sliceGenerator(size uint, r *rand.Rand) (out []int64) {
	for i := uint(0); i < size; i++ {
		out = append(out, rand.Int63())
	}
	return
}

func BenchmarkSubtract(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	testData := sliceGenerator(sliceSize, r)
	what := sliceGenerator(sliceSize, r)

	b.Run("Subtract", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Subtract(testData, what)
		}
	})
}

func BenchmarkContains(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	testData := sliceGenerator(sliceSize, r)
	what := r.Int63()

	b.Run("ContainsInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			ContainsInt64(testData, what)
		}
	})

	b.Run("IndexOfInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			IndexOfInt64(testData, what)
		}
	})

	b.Run("Contains", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Contains(testData, what)
		}
	})
}

func BenchmarkUniq(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	testData := sliceGenerator(sliceSize, r)

	b.Run("UniqInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			UniqInt64(testData)
		}
	})

	b.Run("Uniq", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Uniq(testData)
		}
	})
}

func BenchmarkSum(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	testData := sliceGenerator(sliceSize, r)

	b.Run("SumInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			SumInt64(testData)
		}
	})

	b.Run("Sum", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Sum(testData)
		}
	})
}

func BenchmarkDrop(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	testData := sliceGenerator(sliceSize, r)

	b.Run("DropInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			DropInt64(testData, 1)
		}
	})

	b.Run("Drop", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Drop(testData, 1)
		}
	})
}

func BenchmarkJoin(b *testing.B) {
	r := rand.New(rand.NewSource(seed))
	fullArr := sliceGenerator(sliceSize, r)
	leftArr := fullArr[:sliceSize/3*2]
	rightArr := fullArr[sliceSize/3*1:]

	b.Run("InnerJoinInt64", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			JoinInt64(leftArr, rightArr, InnerJoinInt64)
		}
	})

	b.Run("InnerJoin", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			Join(leftArr, rightArr, InnerJoin)
		}
	})
}
