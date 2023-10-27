// Package obfuscate obfuscates a byte array.
package obfuscate

// key is the key used to obfuscate.
var key = []byte{0x0a, 0xba, 0x0d, 0x1d, 0xea}

// Apply applies the obfuscation key to the byte sequence using XOR.
func Apply(input []byte) (output []byte) {
	for idx, entry := range input {
		output = append(output, entry^key[idx%len(key)])
	}
	return
}
