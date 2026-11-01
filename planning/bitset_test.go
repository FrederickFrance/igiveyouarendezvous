package planning

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

// The bitset works by words; a set of booleans tells what it should do.
func TestBitset(t *testing.T) {
	t.Parallel()

	rnd := rand.New(rand.NewPCG(1, 2))

	for _, n := range []int{1, 63, 64, 65, 200} {
		b := newBitset(n)
		want := make([]bool, n)

		for range 2000 {
			from := rnd.IntN(n)
			to := from + rnd.IntN(n-from+1)

			if rnd.IntN(3) == 0 {
				on := rnd.IntN(2) == 0
				b.set(from, to, on)

				for i := from; i < to; i++ {
					want[i] = on
				}
			}

			free := true
			for i := from; i < to; i++ {
				free = free && !want[i]
			}

			require.Equal(t, free, b.free(from, to), "n=%d [%d, %d)", n, from, to)

			first, last := -1, -1
			for i, on := range want {
				if on {
					if first < 0 {
						first = i
					}

					last = i
				}
			}

			f, l := b.bounds()
			require.Equal(t, []int{first, last}, []int{f, l})
		}
	}
}
