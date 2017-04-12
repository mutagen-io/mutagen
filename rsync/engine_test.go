package rsync

import (
	"bytes"
	"math/rand"
	"testing"
)

// TODO: Add tests to cover edge cases of rsync handling, get coverage closer to
// 100%.

type testDataGenerator struct {
	length    int
	seed      int64
	mutations int
}

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

	// Done.
	return result
}

type engineTestCase struct {
	base       testDataGenerator
	target     testDataGenerator
	maxDataOps int
}

func (c engineTestCase) run(t *testing.T) {
	// Generate base and target data.
	base := c.base.generate()
	target := c.target.generate()

	// Create an engine.
	engine := NewDefaultEngine()

	// Compute the base signature.
	signature := engine.BytesSignature(base)

	// Compute a delta.
	delta := engine.DeltafyBytes(target, signature)

	// Ensure there are no more data operations than expected.
	nDataOperations := 0
	for _, o := range delta {
		if len(o.Data) > 0 {
			nDataOperations += 1
		}
	}
	if c.maxDataOps >= 0 && nDataOperations > c.maxDataOps {
		t.Error("observed more data operations than expected")
	}

	// Apply the delta.
	patched, err := engine.PatchBytes(base, delta)
	if err != nil {
		t.Fatal("unable to patch bytes:", err)
	}

	// Verify success.
	if !bytes.Equal(patched, target) {
		t.Error("patched data did not match expected")
	}
}

func TestBothEmpty(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{0, 0, 0},
		target:     testDataGenerator{0, 0, 0},
		maxDataOps: 0,
	}
	test.run(t)
}

func TestBaseEmpty(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{0, 0, 0},
		target:     testDataGenerator{10 * 1024 * 1024, 473, 0},
		maxDataOps: -1,
	}
	test.run(t)
}

func TestTargetEmpty(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{10 * 1024 * 1024, 473, 0},
		target:     testDataGenerator{0, 0, 0},
		maxDataOps: 0,
	}
	test.run(t)
}

func TestSame(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{10 * 1024 * 1024, 473, 0},
		target:     testDataGenerator{10 * 1024 * 1024, 473, 0},
		maxDataOps: 0,
	}
	test.run(t)
}

func TestSame1Mutation(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{10 * 1024 * 1024, 473, 0},
		target:     testDataGenerator{10 * 1024 * 1024, 473, 1},
		maxDataOps: 1,
	}
	test.run(t)
}

func TestSame2Mutation(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{10 * 1024 * 1024, 473, 0},
		target:     testDataGenerator{10 * 1024 * 1024, 473, 2},
		maxDataOps: 2,
	}
	test.run(t)
}

func TestSameDataShorterTarget(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{9892814, 473, 0},
		target:     testDataGenerator{5 * 1024 * 1024, 473, 0},
		maxDataOps: 0,
	}
	test.run(t)
}

func TestSameDataLongerTarget(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{985498, 473, 0},
		target:     testDataGenerator{15414553, 473, 0},
		maxDataOps: -1,
	}
	test.run(t)
}

func TestDifferentDataSameLength(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{10 * 1024 * 1024, 473, 0},
		target:     testDataGenerator{10 * 1024 * 1024, 182, 0},
		maxDataOps: -1,
	}
	test.run(t)
}

func TestDifferent(t *testing.T) {
	test := engineTestCase{
		base:       testDataGenerator{459879, 473, 0},
		target:     testDataGenerator{21345, 182, 0},
		maxDataOps: -1,
	}
	test.run(t)
}

func TestBlockLength(t *testing.T) {
	// Check invariants required by this test.
	if defaultMaxOpSize < defaultBlockSize {
		t.Fatal("test requires max op size > block size")
	}

	// Create and run the test.
	test := engineTestCase{
		base:       testDataGenerator{0, 0, 0},
		target:     testDataGenerator{defaultBlockSize, 421, 0},
		maxDataOps: 1,
	}
	test.run(t)
}

func TestLessThanBlockLength(t *testing.T) {
	// Create and run the test.
	test := engineTestCase{
		base:       testDataGenerator{0, 0, 0},
		target:     testDataGenerator{defaultBlockSize - 1, 421, 0},
		maxDataOps: 1,
	}
	test.run(t)
}
