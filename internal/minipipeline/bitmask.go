package minipipeline

// Bitmask is a bitmask using a big integer.
type Bitmask struct {
	V []uint8
}

// Set sets the bit in position pos.
func (b *Bitmask) Set(pos int) {
	idx := pos / 8
	for idx >= len(b.V) {
		b.V = append(b.V, 0)
	}
	off := pos % 8
	b.V[idx] |= uint8(1 << off)
}

// Clear clears the bit in position pos.
func (b *Bitmask) Clear(pos int) {
	idx := pos / 8
	if idx >= len(b.V) {
		return
	}
	off := pos % 8
	b.V[idx] &= ^uint8(1 << off)
}

// Get gets the bit in position pos.
func (b *Bitmask) Get(pos int) bool {
	idx := pos / 8
	if idx >= len(b.V) {
		return false
	}
	off := pos % 8
	return (b.V[idx] >> off) == 1
}
