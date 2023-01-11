package core

// emptyByteLookupMap implements byteLookupMap for empty caches.
type emptyByteLookupMap struct{}

// length returns the length of the map.
func (m *emptyByteLookupMap) length() int {
	return 0
}

// insert adds a key-value pair to the map.
func (m *emptyByteLookupMap) insert(_ []byte, _ string) {}

// find looks for a key in the map, returning the associated value (defaulting
// to an empty string if the key was not present) and whether or not the key was
// found.
func (m *emptyByteLookupMap) find(_ []byte) (string, bool) {
	return "", false
}

// byteLookupMap20 implements byteLookupMap for 20-byte digests.
type byteLookupMap20 map[[20]byte]string

// length returns the length of the map.
func (m byteLookupMap20) length() int {
	return len(m)
}

// insert adds a key-value pair to the map.
func (m byteLookupMap20) insert(k []byte, v string) {
	var key [20]byte
	copy(key[:], k)
	m[key] = v
}

// find looks for a key in the map, returning the associated value (defaulting
// to an empty string if the key was not present) and whether or not the key was
// found.
func (m byteLookupMap20) find(k []byte) (string, bool) {
	var key [20]byte
	copy(key[:], k)
	result, ok := m[key]
	return result, ok
}

// byteLookupMap32 implements byteLookupMap for 32-byte digests.
type byteLookupMap32 map[[32]byte]string

// length returns the length of the map.
func (m byteLookupMap32) length() int {
	return len(m)
}

// insert adds a key-value pair to the map.
func (m byteLookupMap32) insert(k []byte, v string) {
	var key [32]byte
	copy(key[:], k)
	m[key] = v
}

// find looks for a key in the map, returning the associated value (defaulting
// to an empty string if the key was not present) and whether or not the key was
// found.
func (m byteLookupMap32) find(k []byte) (string, bool) {
	var key [32]byte
	copy(key[:], k)
	result, ok := m[key]
	return result, ok
}

// byteLookupMap16 implements byteLookupMap for 16-byte digests.
type byteLookupMap16 map[[16]byte]string

// length returns the length of the map.
func (m byteLookupMap16) length() int {
	return len(m)
}

// insert adds a key-value pair to the map.
func (m byteLookupMap16) insert(k []byte, v string) {
	var key [16]byte
	copy(key[:], k)
	m[key] = v
}

// find looks for a key in the map, returning the associated value (defaulting
// to an empty string if the key was not present) and whether or not the key was
// found.
func (m byteLookupMap16) find(k []byte) (string, bool) {
	var key [16]byte
	copy(key[:], k)
	result, ok := m[key]
	return result, ok
}
