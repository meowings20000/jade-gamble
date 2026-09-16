package domain

// Perm returns a permutation of n ints (Fisher-Yates on the Rand source).
func (r *detRand) Perm(n int) []int {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := int(r.next() % uint64(i+1))
		p[i], p[j] = p[j], p[i]
	}
	return p
}

// interface version for crypto rand
type randWithPerm interface {
	Perm(n int) []int
}

// PermAny calls Perm when the source supports it, else Fisher-Yates manually.
func PermAny(r Rand, n int) []int {
	if d, ok := r.(*detRand); ok {
		return d.Perm(n)
	}
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	return p
}
