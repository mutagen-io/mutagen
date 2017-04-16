package rsync

import (
	"bytes"
	"math/rand"
	"testing"
)

// TestMinimumBlockSize verifies that optimalBlockSize returns a sane minimum
// block size.
func TestMinimumBlockSize(t *testing.T) {
	if s := optimalBlockSize(1); s != minimumBlockSize {
		t.Error("incorrect minimum block size:", s, "!=", minimumBlockSize)
	}
}

// TestMaximumBlockSize verifies that optimalBlockSize returns a sane maximum
// block size.
func TestMaximumBlockSize(t *testing.T) {
	if s := optimalBlockSize(maximumBlockSize*maximumBlockSize + 1000); s != maximumBlockSize {
		t.Error("incorrect maximum block size:", s, "!=", maximumBlockSize)
	}
}

// testDataGenerator generates repeatable random byte sequences with optional
// mutations and data prepending.
type testDataGenerator struct {
	length    int
	seed      int64
	mutations int
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
	for i := 0; i < g.mutations; i++ {
		result[random.Intn(g.length)] += 1
	}

	// Prepend data if necessary.
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
	numberOfOperations        int
	maximumDataOperations     int
	expectCoalescedOperations bool
}

// run executes the test case.
func (c engineTestCase) run(t *testing.T) {
	// Generate base and target data.
	base := c.base.generate()
	target := c.target.generate()

	// Create an engine.
	engine := NewEngine()

	// Compute the base signature.
	signature := engine.BytesSignature(base)

	// Compute a delta.
	delta := engine.DeltafyBytes(target, signature)

	// Validate the delta and verify its statistics.
	nDataOperations := 0
	haveCoalescedOperations := false
	for _, o := range delta {
		if err := o.ensureValid(); err != nil {
			t.Error("invalid operation:", err)
		} else if dataLength := len(o.Data); dataLength > 0 {
			if dataLength > maximumDataOperationSize {
				t.Error("data operation size greater than allowed:", dataLength)
			}
			nDataOperations += 1
		} else if o.Count > 1 {
			haveCoalescedOperations = true
		}
	}
	if c.numberOfOperations >= 0 && len(delta) != c.numberOfOperations {
		t.Error(
			"observed different number of operations than expected:",
			len(delta), "!=", c.numberOfOperations,
		)
	}
	if c.maximumDataOperations >= 0 && nDataOperations > c.maximumDataOperations {
		t.Error(
			"observed more data operations than expected:",
			nDataOperations, ">", c.maximumDataOperations,
		)
	}
	if c.expectCoalescedOperations && !haveCoalescedOperations {
		t.Error("expected coalesced operations but found none")
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
		base:                  testDataGenerator{0, 0, 0, nil},
		target:                testDataGenerator{0, 0, 0, nil},
		numberOfOperations:    0,
		maximumDataOperations: 0,
	}
	test.run(t)
}

// TestBaseEmptyNonMaxDataOperationMultiple verifies that data sent against an
// empty base will just be transmitted as data operations.
func TestBaseEmptyMaxDataOperationMultiple(t *testing.T) {
	test := engineTestCase{
		base:                  testDataGenerator{0, 0, 0, nil},
		target:                testDataGenerator{5 * maximumDataOperationSize, 473, 0, nil},
		numberOfOperations:    5,
		maximumDataOperations: 5,
	}
	test.run(t)
}

// TestBaseEmptyNonMaxDataOperationMultiple verifies that data sent against an
// empty base will just be transmitted as data operations, and verifies that
// this is done correctly in the case that one operation that is less than the
// maximum data operation size.
func TestBaseEmptyNonMaxDataOperationMultiple(t *testing.T) {
	test := engineTestCase{
		base:                  testDataGenerator{0, 0, 0, nil},
		target:                testDataGenerator{maximumDataOperationSize + 1, 473, 0, nil},
		numberOfOperations:    2,
		maximumDataOperations: 2,
	}
	test.run(t)
}

// TestTargetEmpty verifies that a completely empty target can be transmitted
// without any operations.
func TestTargetEmpty(t *testing.T) {
	test := engineTestCase{
		base:                  testDataGenerator{10 * 1024 * 1024, 473, 0, nil},
		target:                testDataGenerator{0, 0, 0, nil},
		numberOfOperations:    0,
		maximumDataOperations: 0,
	}
	test.run(t)
}

// TestSame verifies that completely equivalent data will be sent in a single
// coalesced block operation. It requires that the data length be at least two
// blocks in length (although one may be a short block) so that coalescing can
// occur.
func TestSame(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{10 * 1024 * 1024, 473, 0, nil},
		target:                    testDataGenerator{10 * 1024 * 1024, 473, 0, nil},
		numberOfOperations:        1,
		maximumDataOperations:     0,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestSame1Mutation verifies that data which is identical except for a single
// mutations will be transmitted as some number of operations that we can't know
// (due to the randomness of the mutation location) but will be restricted to
// (at most) one data operation (this requies that the maximum data operation
// size be longer than the block size) and include coalesced block operations
// (this requires that the total data length be at least four block lengths so
// that there will be at least two consecutive unmodified blocks).
func TestSame1Mutation(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{maximumDataOperationSize, 473, 0, nil},
		target:                    testDataGenerator{maximumDataOperationSize, 473, 1, nil},
		numberOfOperations:        -1,
		maximumDataOperations:     1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestSame2Mutations verifies that data which is identical except for (up to)
// two mutations will be transmitted as some number of operations that we can't
// know (due to the randomness of the mutation locations) but will be restricted
// to (at most) two data operations (this requies that the maximum data
// operation size be longer than the block size) and include coalesced block
// operations (this requires that the total data length be at least five block
// lengths so that there will be at least two consecutive unmodified blocks).
func TestSame2Mutations(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{maximumDataOperationSize, 473, 0, nil},
		target:                    testDataGenerator{maximumDataOperationSize, 473, 2, nil},
		numberOfOperations:        -1,
		maximumDataOperations:     2,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestTruncateOnBlockBoundary verifies that truncation on a block boundary will
// send only a single coalesced block operation when data is truncated on the
// block boundary. This function unfortunately requires careful coordination
// with the definition of optimalBlockSize.
func TestTruncateOnBlockBoundary(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{240000, 473, 0, nil},
		target:                    testDataGenerator{4800, 473, 0, nil},
		numberOfOperations:        1,
		maximumDataOperations:     0,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestTruncateOffBlockBoundary verifies that truncation that's not on a block
// boundary will send only a single coalesced block operation and a single data
// operation when data is truncated within one maximum data operation size of a
// block boundary. This function unfortunately requires careful coordination
// with the definition of optimalBlockSize.
func TestTruncateOffBlockBoundary(t *testing.T) {
	test := engineTestCase{
		// This will yield a block size of 2400.
		base:                      testDataGenerator{240000, 473, 0, nil},
		target:                    testDataGenerator{4800 + maximumDataOperationSize, 473, 0, nil},
		numberOfOperations:        2,
		maximumDataOperations:     1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestPrepend verifies that data which has been prepended with data shorter
// than the maximum data operation size can be transmitted in a single data
// operation and a single coalesced block operation. We choose a base length
// that ensures we have a short block to match to verify that short block
// matching works. This function unfortunately requires careful coordination
// with the definition of optimalBlockSize. It also requires that base has a
// length longer than two blocks so that coalescing can occur.
func TestPrepend(t *testing.T) {
	test := engineTestCase{
		base:                      testDataGenerator{10 * 1024 * 1024, 473, 0, nil},
		target:                    testDataGenerator{10 * 1024 * 1024, 473, 0, []byte{1, 2, 3}},
		numberOfOperations:        2,
		maximumDataOperations:     1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestAppend verifies that data which has been appended with data shorter than
// the maximum data operation size can be transmitted in a single coalesced
// block operation and a data operation. Because the rsync algorithm can't match
// short blocks that aren't at the end of the target, we have to ensure that the
// append happens on a full block size and that it can fit into a single data
// operation (or, alternatively, that the short block and appended data can fit
// into a single data operation, but this is harder, but also perhaps more
// desirable). This function unfortunately requires careful coordination with
// the definition of optimalBlockSize. It also requires that base has a length
// longer than two blocks so that coalescing can occur.
func TestAppend(t *testing.T) {
	test := engineTestCase{
		// This will yield a block size of 24 * minimumBlockSize.
		base:                      testDataGenerator{24 * minimumBlockSize * minimumBlockSize, 473, 0, nil},
		target:                    testDataGenerator{24*minimumBlockSize*minimumBlockSize + (maximumDataOperationSize / 2), 473, 0, nil},
		numberOfOperations:        2,
		maximumDataOperations:     1,
		expectCoalescedOperations: true,
	}
	test.run(t)
}

// TestDifferentDataSameLength verifies that different data with no matching
// blocks but the same length won't be influenced by the matching length and
// will just send the new data.
func TestDifferentDataSameLength(t *testing.T) {
	test := engineTestCase{
		base:                  testDataGenerator{10*maximumDataOperationSize + 1, 473, 0, nil},
		target:                testDataGenerator{10*maximumDataOperationSize + 1, 182, 0, nil},
		numberOfOperations:    11,
		maximumDataOperations: 11,
	}
	test.run(t)
}

// TestDifferentDataDifferentLength verifies that different data with no
// matching blocks and different total length will just send the new data.
func TestDifferentDataDifferentLength(t *testing.T) {
	test := engineTestCase{
		base:                  testDataGenerator{678345, 473, 0, nil},
		target:                testDataGenerator{10*maximumDataOperationSize + 1, 182, 0, nil},
		numberOfOperations:    11,
		maximumDataOperations: 11,
	}
	test.run(t)
}
