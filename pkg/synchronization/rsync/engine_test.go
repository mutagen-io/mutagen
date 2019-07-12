package rsync

import (
	"bytes"
	"math/rand"
	"testing"
)

// TestBlockHashNilInvalid verifies that a nil block hash is treated as invalid.
func TestBlockHashNilInvalid(t *testing.T) {
	var hash *BlockHash
	if hash.EnsureValid() == nil {
		t.Error("nil block hash considered valid")
	}
}

// TestBlockHashNilStrongHashInvalid verifies that a block has with a nil strong
// hash is treated as invalid.
func TestBlockHashNilStrongHashInvalid(t *testing.T) {
	hash := &BlockHash{Weak: 5}
	if hash.EnsureValid() == nil {
		t.Error("block hash with nil strong hash considered valid")
	}
}

// TestBlockHashEmptyStrongHashInvalid verifies that a block has with an empty
// strong hash is treated as invalid.
func TestBlockHashEmptyStrongHashInvalid(t *testing.T) {
	hash := &BlockHash{Weak: 5, Strong: make([]byte, 0)}
	if hash.EnsureValid() == nil {
		t.Error("block hash with empty strong hash considered valid")
	}
}

// TestSignatureNilInvalid verifies that a nil signature is treated as invalid.
func TestSignatureNilInvalid(t *testing.T) {
	var signature *Signature
	if signature.EnsureValid() == nil {
		t.Error("nil signature considered valid")
	}
}

// TestSignatureZeroBlockSizeNonZeroLastBlockSizeInvalid verifies the
// EnsureValid behavior of Signature when the block size is zero and the last
// block size is non-zero.
func TestSignatureZeroBlockSizeNonZeroLastBlockSizeInvalid(t *testing.T) {
	signature := &Signature{LastBlockSize: 8192}
	if signature.EnsureValid() == nil {
		t.Error("zero block size with non-zero last block size considered valid")
	}
}

// TestSignatureZeroBlockSizeWithHashesInvalid verifies the EnsureValid behavior
// of Signature when the block size is zero and hashes are present.
func TestSignatureZeroBlockSizeWithHashesInvalid(t *testing.T) {
	signature := &Signature{Hashes: []*BlockHash{{Weak: 5, Strong: []byte{0x0}}}}
	if signature.EnsureValid() == nil {
		t.Error("zero block size with hashes considered valid")
	}
}

// TestSignatureZeroLastBlockSizeInvalid verifies the EnsureValid behavior of
// Signature when the last block size is 0.
func TestSignatureZeroLastBlockSizeInvalid(t *testing.T) {
	signature := &Signature{BlockSize: 8192}
	if signature.EnsureValid() == nil {
		t.Error("zero last block size considered valid")
	}
}

// TestSignatureLastBlockSizeTooBigInvalid verifies the EnsureValid behavior of
// Signature when the last block size is too big.
func TestSignatureLastBlockSizeTooBigInvalid(t *testing.T) {
	signature := &Signature{BlockSize: 8192, LastBlockSize: 8193}
	if signature.EnsureValid() == nil {
		t.Error("overly large last block size considered valid")
	}
}

// TestSignatureNoHashesInvalid verifies the EnsureValid behavior of Signature
// when no hashes are present.
func TestSignatureNoHashesInvalid(t *testing.T) {
	signature := &Signature{BlockSize: 8192, LastBlockSize: 8192}
	if signature.EnsureValid() == nil {
		t.Error("signature with no hashes considered valid")
	}
}

// TestSignatureInvalidHashesInvalid verifies the EnsureValid behavior of
// Signature when invalid hashes are present.
func TestSignatureInvalidHashesInvalid(t *testing.T) {
	signature := &Signature{
		BlockSize:     8192,
		LastBlockSize: 8192,
		Hashes:        []*BlockHash{nil},
	}
	if signature.EnsureValid() == nil {
		t.Error("signature with no hashes considered valid")
	}
}

// TestSignatureValid verifies the EnsureValid behavior of Signature for a valid
// signature.
func TestSignatureValid(t *testing.T) {
	signature := &Signature{
		BlockSize:     8192,
		LastBlockSize: 8192,
		Hashes:        []*BlockHash{{Weak: 1, Strong: []byte{0x0}}},
	}
	if err := signature.EnsureValid(); err != nil {
		t.Error("valid signature failed validation:", err)
	}
}

// TestOperationNilInvalid verifies that a nil operation is treated as invalid.
func TestOperationNilInvalid(t *testing.T) {
	var operation *Operation
	if operation.EnsureValid() == nil {
		t.Error("nil operation considered valid")
	}
}

// TestOperationDataAndStartInvalid verifies the EnsureValid behavior of
// Operation when data and a block start index are provided.
func TestOperationDataAndStartInvalid(t *testing.T) {
	operation := &Operation{Data: []byte{0}, Start: 4}
	if operation.EnsureValid() == nil {
		t.Error("operation with data and start considered valid")
	}
}

// TestOperationDataAndCountInvalid verifies the EnsureValid behavior of
// Operation when data and a block count are provided.
func TestOperationDataAndCountInvalid(t *testing.T) {
	operation := &Operation{Data: []byte{0}, Count: 4}
	if operation.EnsureValid() == nil {
		t.Error("operation with data and count considered valid")
	}
}

// TestOperationZeroCountInvalid verifies the EnsureValid behavior of Operation
// when the block count is zero.
func TestOperationZeroCountInvalid(t *testing.T) {
	operation := &Operation{Start: 40}
	if operation.EnsureValid() == nil {
		t.Error("operation with zero count considered valid")
	}
}

// TestOperationDataValid verifies the EnsureValid behavior of Operation in the
// case of a valid data operation.
func TestOperationDataValid(t *testing.T) {
	operation := &Operation{Data: []byte{0}}
	if err := operation.EnsureValid(); err != nil {
		t.Error("valid data operation considered invalid")
	}
}

// TestOperationBlocksValid verifies the EnsureValid behavior of Operation in the
// case of a valid block operation.
func TestOperationBlocksValid(t *testing.T) {
	operation := &Operation{Start: 10, Count: 50}
	if err := operation.EnsureValid(); err != nil {
		t.Error("valid block operation considered invalid")
	}
}

// TestMinimumBlockSize verifies that OptimalBlockSizeForBaseLength returns a
// sane minimum block size.
func TestMinimumBlockSize(t *testing.T) {
	if s := OptimalBlockSizeForBaseLength(1); s != minimumOptimalBlockSize {
		t.Error("incorrect minimum block size:", s, "!=", minimumOptimalBlockSize)
	}
}

// TestMaximumBlockSize verifies that OptimalBlockSizeForBaseLength returns a
// sane maximum block size.
func TestMaximumBlockSize(t *testing.T) {
	if s := OptimalBlockSizeForBaseLength(maximumOptimalBlockSize * maximumOptimalBlockSize); s != maximumOptimalBlockSize {
		t.Error("incorrect maximum block size:", s, "!=", maximumOptimalBlockSize)
	}
}

// TestOptimalBlockSizeForBase verifies the behavior of OptimalBlockSizeForBase.
func TestOptimalBlockSizeForBase(t *testing.T) {
	// Create a base.
	baseLength := uint64(1234567)
	base := bytes.NewReader(make([]byte, baseLength))

	// Compute the optimal block size using OptimalBlockSizeForBase.
	optimalBlockSize, err := OptimalBlockSizeForBase(base)
	if err != nil {
		t.Fatal("unable to compute optimal block size for base")
	}

	// Compate it with what we'd expect by computing manually.
	expectedOptimalBlockSize := OptimalBlockSizeForBaseLength(baseLength)
	if optimalBlockSize != expectedOptimalBlockSize {
		t.Error(
			"mismatch between optimal block size computations:",
			optimalBlockSize, "!=", expectedOptimalBlockSize,
		)
	}

	// Ensure that the reader was reset to the beginning.
	if uint64(base.Len()) != baseLength {
		t.Error("base was not reset to beginning")
	}
}

// testDataGenerator generates repeatable random byte sequences with optional
// mutations and data prepending.
type testDataGenerator struct {
	length    int
	seed      int64
	mutations []int
	prepend   []byte
}

// generate creates a byte sequence based on the generator's parameters.
func (g testDataGenerator) generate() []byte {
	// Create a random number generator.
	random := rand.New(rand.NewSource(g.seed))

	// Create a buffer and fill it. The read is guaranteed to succeed.
	result := make([]byte, g.length)
	random.Read(result)

	// Mutate.
	for _, index := range g.mutations {
		result[index] += 1
	}

	// Prepend data if necessary. This isn't super-efficient, but it's fine for
	// testing.
	if len(g.prepend) > 0 {
		result = append(g.prepend, result...)
	}

	// Done.
	return result
}

// engineTestCase performs an rsync cycle with a specified base and target and
// verifies certain behavior/parameters of the cycle.
type engineTestCase struct {
	base                      testDataGenerator
	target                    testDataGenerator
	blockSize                 uint64
	maxDataOpSize             uint64
	numberOfOperations        uint
	numberOfDataOperations    uint
	expectCoalescedOperations bool
}

// run executes the test case.
func (c engineTestCase) run(t *testing.T) {
	// Mark this as a helper function.
	t.Helper()

	// Generate base and target data.
	base := c.base.generate()
	target := c.target.generate()

	// Create an engine.
	engine := NewEngine()

	// Compute the base signature. Verify that it's sane and that it used the
	// correct block size.
	signature := engine.BytesSignature(base, c.blockSize)
	if err := signature.EnsureValid(); err != nil {
		t.Fatal("generated signature was invalid:", err)
	} else if len(signature.Hashes) != 0 {
		if c.blockSize != 0 && signature.BlockSize != c.blockSize {
			t.Error(
				"generated signature did not have correct block size:",
				signature.BlockSize, "!=", c.blockSize,
			)
		}
	}

	// Compute a delta.
	delta := engine.DeltafyBytes(target, signature, c.maxDataOpSize)

	// Determine what we should expect for the maximumd data operation size.
	expectedMaxDataOpSize := c.maxDataOpSize
	if expectedMaxDataOpSize == 0 {
		expectedMaxDataOpSize = DefaultMaximumDataOperationSize
	}

	// Validate the delta and verify its statistics.
	nDataOperations := uint(0)
	haveCoalescedOperations := false
	for _, o := range delta {
		if err := o.EnsureValid(); err != nil {
			t.Error("invalid operation:", err)
		} else if dataLength := uint64(len(o.Data)); dataLength > 0 {
			if dataLength > expectedMaxDataOpSize {
				t.Error(
					"data operation size greater than allowed:",
					dataLength, ">", expectedMaxDataOpSize,
				)
			}
			nDataOperations += 1
		} else if o.Count > 1 {
			haveCoalescedOperations = true
		}
	}
	if uint(len(delta)) != c.numberOfOperations {
		t.Error(
			"observed different number of operations than expected:",
			len(delta), "!=", c.numberOfOperations,
		)
	}
	if nDataOperations != c.numberOfDataOperations {
		t.Error(
			"observed different number of data operations than expected:",
			nDataOperations, ">", c.numberOfDataOperations,
		)
	}
	if haveCoalescedOperations != c.expectCoalescedOperations {
		t.Error(
			"expectations about coalescing not met:",
			haveCoalescedOperations, "!=", c.expectCoalescedOperations,
		)
	}

	// Apply the delta.
	patched, err := engine.PatchBytes(base, signature, delta)
	if err != nil {
		t.Fatal("unable to patch bytes:", err)
	}

	// Verify success.
	if !bytes.Equal(patched, target) {
		t.Error("patched data did not match expected")
	}
}

// TestBothEmpty verifies that no operations are exchanged in the case that both
// base and target are empty.
func TestBothEmpty(t *testing.T) {
	test := engineTestCase{
		base:   testDataGenerator{},
		target: testDataGenerator{},
	}
	test.run(t)
}

// TestBaseEmptyMaxDataOperationMultiple verifies that data sent against an
// empty base will just be transmitted as data operations, and verifies that
// this is done correctly in the case that the data length is a multiple of the
// maximum data operation size.
func TestBaseEmptyMaxDataOperationMultiple(t *testing.T) {
	test := engineTestCase{
		base:                   testDataGenerator{},
		target:                 testDataGenerator{10240, 473, nil, nil},
		maxDataOpSize:          1024,
		numberOfOperations:     10,
		numberOfDataOperations: 10,
	}
	test.run(t)
}

// TestBaseEmptyNonMaxDataOperationMultiple verifies that data sent against an
// empty base will just be transmitted as data operations, and verifies that
// this is done correctly in the case that one operation that is less than the
// maximum data operation size.
func TestBaseEmptyNonMaxDataOperationMultiple(t *testing.T) {
	test := engineTestCase{
		base:                   testDataGenerator{},
		target:                 testDataGenerator{10241, 473, nil, nil},
		maxDataOpSize:          1024,
		numberOfOperations:     11,
		numberOfDataOperations: 11,
	}
	test.run(t)
}

// TestTargetEmpty verifies that a completely empty target can be transmitted
// without any operations.
func TestTargetEmpty(t *testing.T) {
	test := engineTestCase{
		base:   testDataGenerator{12345, 473, nil, nil},
		target: testDataGenerator{},
	}
	test.run(t)
}

// TestSame verifies that completely equivalent data will be sent in a single
// coalesced block operation. It requires that the data length be at least two
// blocks in length (although one may be a short block) so that coalescing can
// occur.
func TestSame(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{1234567, 473, nil, nil},
		target:                    testDataGenerator{1234567, 473, nil, nil},
		numberOfOperations:        1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestSame1Mutation verifies that data which is identical except for a single
// mutation in the second block (of ten blocks) will be transmitted as two
// block operations (one of which is coalesced) and a single data operation. It
// sets the maximum data operation size to ensure that the mutated block can be
// sent in a single data operation.
func TestSame1Mutation(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{10240, 473, nil, nil},
		target:                    testDataGenerator{10240, 473, []int{1300}, nil},
		blockSize:                 1024,
		maxDataOpSize:             1024,
		numberOfOperations:        3,
		numberOfDataOperations:    1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestSame2Mutations verifies that data which is identical except for mutations
// in the second and fourth blocks (of five blocks, the last of which is short)
// will be transmitted as three block operations (none of which are coalesced)
// and two data operations. It sets the maximum data operation size to ensure
// that the mutated blocks can be sent in single data operations.
func TestSame2Mutations(t *testing.T) {
	test := engineTestCase{
		base:                   testDataGenerator{10220, 473, nil, nil},
		target:                 testDataGenerator{10220, 473, []int{2073, 7000}, nil},
		blockSize:              2048,
		maxDataOpSize:          2048,
		numberOfOperations:     5,
		numberOfDataOperations: 2,
	}
	test.run(t)
}

// TestTruncateOnBlockBoundary verifies that truncation on a block boundary will
// send only a single coalesced block operation when data is truncated on the
// block boundary.
func TestTruncateOnBlockBoundary(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{999, 212, nil, nil},
		target:                    testDataGenerator{666, 212, nil, nil},
		blockSize:                 333,
		numberOfOperations:        1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestTruncateOffBlockBoundary verifies that truncation that's not on a block
// boundary will send only a single coalesced block operation and a single data
// operation when data is truncated within one maximum data operation size of a
// block boundary.
func TestTruncateOffBlockBoundary(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{888, 912, nil, nil},
		target:                    testDataGenerator{790, 912, nil, nil},
		blockSize:                 111,
		maxDataOpSize:             1024,
		numberOfOperations:        2,
		numberOfDataOperations:    1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestPrepend verifies that data which has been prepended with data shorter
// than the maximum data operation size can be transmitted in a single data
// operation and a single coalesced block operation. It also tests short block
// matching.
func TestPrepend(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{9880, 11, nil, nil},
		target:                    testDataGenerator{9880, 11, nil, []byte{1, 2, 3}},
		blockSize:                 1234,
		maxDataOpSize:             5,
		numberOfOperations:        2,
		numberOfDataOperations:    1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestAppend verifies that data which has been appended with data shorter than
// the maximum data operation size can be transmitted in a single coalesced
// block operation and a data operation. Because the rsync algorithm can't match
// short blocks that aren't at the end of the target, we have to ensure that the
// short block and the appended data can fit into a single data operation.
func TestAppend(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{45271, 473, nil, nil},
		target:                    testDataGenerator{45271 + 876, 473, nil, nil},
		blockSize:                 6453,
		maxDataOpSize:             1024,
		numberOfOperations:        2,
		numberOfDataOperations:    1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestDifferentDataSameLength verifies that different data with no matching
// blocks but the same length won't be influenced by the matching length and
// will just send the new data.
func TestDifferentDataSameLength(t *testing.T) {
	test := engineTestCase{
		base:                   testDataGenerator{10473, 473, nil, nil},
		target:                 testDataGenerator{10473, 182, nil, nil},
		maxDataOpSize:          1024,
		numberOfOperations:     11,
		numberOfDataOperations: 11,
	}
	test.run(t)
}

// TestDifferentDataDifferentLength verifies that different data with no
// matching blocks and different total length will just send the new data.
func TestDifferentDataDifferentLength(t *testing.T) {
	test := engineTestCase{
		base:                   testDataGenerator{678345, 473, nil, nil},
		target:                 testDataGenerator{473711, 182, nil, nil},
		maxDataOpSize:          12304,
		numberOfOperations:     39,
		numberOfDataOperations: 39,
	}
	test.run(t)
}
