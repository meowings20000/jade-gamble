package domain

import "crypto/rand"

// Rand is the randomness interface; production uses crypto, tests inject a
// deterministic source so EV tables and stone generation can be asserted.
type Rand interface {
	Float64() float64 // [0,1)
	Intn(n int) int   // [0,n)
}

type cryptoRand struct{}

func (cryptoRand) Float64() float64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	u := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	return float64(u>>11) / float64(1<<53)
}

func (cryptoRand) Intn(n int) int {
	if n <= 0 {
		panic("Intn n<=0")
	}
	return int(cryptoFloat() * float64(n))
}

func cryptoFloat() float64 { return RandSource.Float64() }

// RandSource is the process-wide randomness source.
var RandSource Rand = cryptoRand{}

// detRand is a deterministic PRNG (mulberry64) for tests.
type detRand struct{ state uint64 }

func NewDetRand(seed uint64) Rand { return &detRand{state: seed} }

func (d *detRand) next() uint64 {
	d.state += 0x9E3779B97F4A7C15
	z := d.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func (d *detRand) Float64() float64 { return float64(d.next()>>11) / float64(1<<53) }
func (d *detRand) Intn(n int) int   { return int(d.next() % uint64(n)) }

// RngIntn/intRange helpers on any Rand.
func Range(r Rand, lo, hi int) int          { return lo + r.Intn(hi-lo+1) }
func RangeF(r Rand, lo, hi float64) float64 { return lo + r.Float64()*(hi-lo) }
