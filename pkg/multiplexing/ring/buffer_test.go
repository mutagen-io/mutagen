package ring

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// operation represents an operation on a Buffer.
type operation interface {
	// perform performs the operation, validating that results and error
	// conditions behave as expected. It should only return an error if the
	// operation's behavior doesn't match what's expected.
	perform(buffer *Buffer) error
}

// write encodes a call to Buffer.Write.
type write struct {
	// data is the Write argument.
	data []byte
	// result is the expected count return value.
	result int
	// err is the expected error return value.
	err error
}

// perform implements operation.perform.
func (w *write) perform(buffer *Buffer) error {
	if result, err := buffer.Write(w.data); err != w.err {
		if err != nil {
			return errors.New("unexpectedly nil error")
		}
		return err
	} else if result != w.result {
		return errors.New("Write returned unexpected count")
	}
	return nil
}

// writeByte encodes a call to Buffer.WriteByte.
type writeByte struct {
	// value is the WriteByte argument.
	value byte
	// err is the expected error return value.
	err error
}

// perform implements operation.perform.
func (w *writeByte) perform(buffer *Buffer) error {
	if err := buffer.WriteByte(w.value); err != w.err {
		if err != nil {
			return errors.New("unexpectedly nil error")
		}
		return err
	}
	return nil
}

// readNFrom encodes a call to Buffer.ReadNFrom.
type readNFrom struct {
	// data is the data used to populate a bytes.Reader passed to ReadNFrom.
	data []byte
	// n is the requested number of bytes to read.
	n int
	// result is the expected count return value.
	result int
	// err is the expected error return value.
	err error
}

// perform implements operation.perform.
func (r *readNFrom) perform(buffer *Buffer) error {
	source := bytes.NewReader(r.data)
	if result, err := buffer.ReadNFrom(source, r.n); err != r.err {
		if err != nil {
			return errors.New("unexpectedly nil error")
		}
		return err
	} else if result != r.result {
		return errors.New("ReadNFrom returned unexpected count")
	} else if result+source.Len() != len(r.data) {
		return errors.New("ReadNFrom reported incorrect number of bytes")
	}
	return nil
}

// sameCallEOFReader is an io.Reader that fills a buffer and returns io.EOF on
// the same call.
type sameCallEOFReader struct{}

// Read implements io.Reader.Read.
func (r *sameCallEOFReader) Read(buffer []byte) (int, error) {
	return len(buffer), io.EOF
}

// readNFromEOFSameCall encodes a call to Buffer.ReadNFrom with a single-element
// reader that returns io.EOF on the same call from which it returns data.
type readNFromEOFSameCall struct {
	// n is the requested number of bytes to read.
	n int
}

// perform implements operation.perform.
func (r *readNFromEOFSameCall) perform(buffer *Buffer) error {
	if result, err := buffer.ReadNFrom(&sameCallEOFReader{}, r.n); err != nil {
		return err
	} else if result != r.n {
		return errors.New("unexpected result count")
	}
	return nil
}

// read encodes a call to Buffer.Read.
type read struct {
	// buffer is the Read argument.
	buffer []byte
	// expected is the expected resulting value of buffer after the read.
	expected []byte
	// result is the expected count return value.
	result int
	// err is the expected error return value.
	err error
}

// perform implements operation.perform.
func (r *read) perform(buffer *Buffer) error {
	if len(r.buffer) != len(r.expected) {
		return errors.New("invalid read operation specification")
	} else if result, err := buffer.Read(r.buffer); err != r.err {
		if err != nil {
			return errors.New("unexpectedly nil error")
		}
		return err
	} else if result != r.result {
		return errors.New("Read returned unexpected count")
	} else if !bytes.Equal(r.buffer, r.expected) {
		return errors.New("Read results do not match expected")
	}
	return nil
}

// readByte encodes a call to Buffer.ReadByte.
type readByte struct {
	// result is the expected byte return value.
	result byte
	// err is the expected error return value.
	err error
}

// perform implements operation.perform.
func (r *readByte) perform(buffer *Buffer) error {
	if result, err := buffer.ReadByte(); err != r.err {
		if err != nil {
			return errors.New("unexpectedly nil error")
		}
		return err
	} else if result != r.result {
		return errors.New("ReadByte returned unexpected value")
	}
	return nil
}

// writeTo encodes a call to Buffer.WriteTo.
type writeTo struct {
	// expected is the data expected to be written to the bytes.Buffer passed to
	// WriteTo.
	expected []byte
}

// perform implements operation.perform.
func (w *writeTo) perform(buffer *Buffer) error {
	destination := &bytes.Buffer{}
	if result, err := buffer.WriteTo(destination); err != nil {
		return err
	} else if result != int64(destination.Len()) {
		return errors.New("WriteTo reported incorrect number of bytes")
	} else if destination.Len() != len(w.expected) {
		return errors.New("number of bytes written does not match expected")
	} else if !bytes.Equal(destination.Bytes(), w.expected) {
		return errors.New("bytes written do not match expected")
	}
	return nil
}

// TestBuffer tests Buffer.
func TestBuffer(t *testing.T) {
	// Define test cases.
	tests := []struct {
		buffer     *Buffer
		size       int
		operations []operation
		expected   *Buffer
	}{
		// Test zero-value buffer.
		{&Buffer{}, 0, []operation{&write{nil, 0, nil}}, &Buffer{}},

		// Test NewBuffer.
		{nil, -1, nil, &Buffer{}},
		{nil, 0, nil, &Buffer{}},
		{nil, 1, nil, &Buffer{storage: make([]byte, 1), size: 1}},
		{nil, 2, nil, &Buffer{storage: make([]byte, 2), size: 2}},
		{nil, 4, nil, &Buffer{storage: make([]byte, 4), size: 4}},
		{nil, 128, nil, &Buffer{storage: make([]byte, 128), size: 128}},
		{nil, 1 << 16, nil, &Buffer{storage: make([]byte, 1<<16), size: 1 << 16}},

		// Test Buffer.Write.
		{
			nil, 0,
			[]operation{
				&write{nil, 0, nil},
			},
			&Buffer{},
		},
		{
			nil, 0,
			[]operation{
				&write{[]byte{1}, 0, ErrBufferFull},
			},
			&Buffer{},
		},
		{
			nil, 1,
			[]operation{
				&write{nil, 0, nil},
			},
			&Buffer{storage: []byte{0}, size: 1},
		},
		{
			nil, 1,
			[]operation{
				&write{[]byte{1}, 1, nil},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 1,
			[]operation{
				&write{[]byte{1}, 1, nil},
				&write{[]byte{2}, 0, ErrBufferFull},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 1,
			[]operation{
				&write{[]byte{1, 2}, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 2,
			[]operation{
				&write{[]byte{1}, 1, nil},
				&write{[]byte{2}, 1, nil},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 2,
			[]operation{
				&write{[]byte{1}, 1, nil},
				&write{[]byte{2}, 1, nil},
				&write{[]byte{3}, 0, ErrBufferFull},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 2,
			[]operation{
				&write{[]byte{1, 2}, 2, nil},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 3,
			[]operation{
				&write{[]byte{1, 2}, 2, nil},
				&write{[]byte{3, 4}, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{1, 2, 3}, size: 3, used: 3},
		},
		{
			&Buffer{storage: []byte{0, 0, 1, 0}, size: 4, start: 2, used: 1}, 0,
			[]operation{
				&write{[]byte{2, 3}, 2, nil},
				&write{[]byte{4, 5}, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4, start: 2, used: 4},
		},
		{
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 2}, 0,
			[]operation{
				&write{[]byte{3, 4}, 2, nil},
				&write{[]byte{5, 6}, 0, ErrBufferFull},
			},
			&Buffer{storage: []byte{2, 3, 4, 1}, size: 4, start: 3, used: 4},
		},
		{
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 2}, 0,
			[]operation{
				&write{[]byte{3, 4, 5}, 2, ErrBufferFull},
			},
			&Buffer{storage: []byte{2, 3, 4, 1}, size: 4, start: 3, used: 4},
		},

		// Test Buffer.WriteByte.
		{
			nil, 0,
			[]operation{
				&writeByte{1, ErrBufferFull},
			},
			&Buffer{},
		},
		{
			nil, 1,
			[]operation{
				&writeByte{1, nil},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 1,
			[]operation{
				&writeByte{1, nil},
				&writeByte{2, ErrBufferFull},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 2,
			[]operation{
				&writeByte{1, nil},
			},
			&Buffer{storage: []byte{1, 0}, size: 2, used: 1},
		},
		{
			nil, 2,
			[]operation{
				&writeByte{1, nil},
				&writeByte{2, nil},
				&writeByte{3, ErrBufferFull},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			&Buffer{storage: []byte{0, 0, 1, 0}, size: 4, start: 2, used: 1}, 0,
			[]operation{
				&writeByte{2, nil},
				&writeByte{3, nil},
				&writeByte{4, nil},
				&writeByte{5, ErrBufferFull},
			},
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4, start: 2, used: 4},
		},
		{
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 2}, 0,
			[]operation{
				&writeByte{3, nil},
				&writeByte{4, nil},
				&writeByte{5, ErrBufferFull},
			},
			&Buffer{storage: []byte{2, 3, 4, 1}, size: 4, start: 3, used: 4},
		},

		// Test Buffer.ReadNFrom.
		{
			nil, 0,
			[]operation{
				&readNFrom{nil, 0, 0, nil},
			},
			&Buffer{},
		},
		{
			nil, 0,
			[]operation{
				&readNFrom{[]byte{1}, 1, 0, ErrBufferFull},
			},
			&Buffer{},
		},
		{
			nil, 1,
			[]operation{
				&readNFrom{nil, 0, 0, nil},
			},
			&Buffer{storage: []byte{0}, size: 1},
		},
		{
			nil, 1,
			[]operation{
				&readNFrom{[]byte{1}, 1, 1, nil},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 1,
			[]operation{
				&readNFrom{[]byte{1, 2}, 2, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{1}, size: 1, used: 1},
		},
		{
			nil, 2,
			[]operation{
				&readNFrom{[]byte{1, 2}, 2, 2, nil},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 2,
			[]operation{
				&readNFrom{[]byte{1, 2, 3}, 3, 2, ErrBufferFull},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 2,
			[]operation{
				&write{[]byte{1, 2}, 2, nil},
			},
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2},
		},
		{
			nil, 3,
			[]operation{
				&write{[]byte{1, 2}, 2, nil},
				&readNFrom{[]byte{3, 4}, 2, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{1, 2, 3}, size: 3, used: 3},
		},
		{
			&Buffer{storage: []byte{0, 0, 1, 0}, size: 4, start: 2, used: 1}, 0,
			[]operation{
				&readNFrom{[]byte{2, 3}, 2, 2, nil},
				&readNFrom{[]byte{4, 5}, 2, 1, ErrBufferFull},
			},
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4, start: 2, used: 4},
		},
		{
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 2}, 0,
			[]operation{
				&readNFrom{[]byte{3, 4}, 2, 2, nil},
				&write{[]byte{5, 6}, 0, ErrBufferFull},
			},
			&Buffer{storage: []byte{2, 3, 4, 1}, size: 4, start: 3, used: 4},
		},
		{
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 2}, 0,
			[]operation{
				&readNFromEOFSameCall{2},
			},
			&Buffer{storage: []byte{2, 0, 0, 1}, size: 4, start: 3, used: 4},
		},

		// Test Buffer.Read.
		{
			nil, 0,
			[]operation{
				&read{[]byte{}, []byte{}, 0, nil},
			},
			&Buffer{},
		},
		{
			nil, 0,
			[]operation{
				&read{[]byte{0}, []byte{0}, 0, io.EOF},
			},
			&Buffer{},
		},
		{
			&Buffer{storage: []byte{1, 2, 3, 4}, size: 4, used: 4}, 0,
			[]operation{
				&read{[]byte{0, 0, 0, 0}, []byte{1, 2, 3, 4}, 4, nil},
			},
			&Buffer{storage: []byte{1, 2, 3, 4}, size: 4},
		},
		{
			&Buffer{storage: []byte{1, 2, 3, 4}, size: 4, used: 4}, 0,
			[]operation{
				&read{[]byte{0, 0}, []byte{1, 2}, 2, nil},
			},
			&Buffer{storage: []byte{1, 2, 3, 4}, size: 4, start: 2, used: 2},
		},
		{
			&Buffer{storage: []byte{1, 0, 0, 0}, size: 4, used: 1}, 0,
			[]operation{
				&read{[]byte{0, 2, 3, 4}, []byte{1, 2, 3, 4}, 1, io.EOF},
			},
			&Buffer{storage: []byte{1, 0, 0, 0}, size: 4},
		},
		{
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4, start: 2, used: 4}, 0,
			[]operation{
				&read{[]byte{0, 0, 0, 0}, []byte{1, 2, 3, 4}, 4, nil},
			},
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4},
		},
		{
			&Buffer{storage: []byte{3, 4, 0, 0, 1, 2}, size: 6, start: 4, used: 4}, 0,
			[]operation{
				&read{[]byte{0, 0, 0, 0, 0}, []byte{1, 2, 3, 4, 0}, 4, nil},
			},
			&Buffer{storage: []byte{3, 4, 0, 0, 1, 2}, size: 6},
		},
		{
			&Buffer{storage: []byte{0, 0, 1, 2}, size: 4, start: 2, used: 2}, 0,
			[]operation{
				&read{[]byte{0, 0, 0, 0, 0}, []byte{1, 2, 0, 0, 0}, 2, nil},
			},
			&Buffer{storage: []byte{0, 0, 1, 2}, size: 4},
		},

		// Test Buffer.ReadByte.
		{
			nil, 0,
			[]operation{
				&readByte{0, io.EOF},
			},
			&Buffer{},
		},
		{
			nil, 1,
			[]operation{
				&readByte{0, io.EOF},
			},
			&Buffer{storage: []byte{0}, size: 1},
		},
		{
			&Buffer{storage: []byte{1, 0}, size: 2, used: 1}, 0,
			[]operation{
				&readByte{1, nil},
				&readByte{0, io.EOF},
			},
			&Buffer{storage: []byte{1, 0}, size: 2},
		},
		{
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2}, 0,
			[]operation{
				&readByte{1, nil},
				&readByte{2, nil},
			},
			&Buffer{storage: []byte{1, 2}, size: 2},
		},
		{
			&Buffer{storage: []byte{0, 1}, size: 2, start: 1, used: 1}, 0,
			[]operation{
				&readByte{1, nil},
				&readByte{0, io.EOF},
			},
			&Buffer{storage: []byte{0, 1}, size: 2},
		},

		// Test Buffer.WriteTo.
		{
			nil, 0,
			[]operation{
				&writeTo{[]byte{}},
			},
			&Buffer{},
		},
		{
			nil, 1,
			[]operation{
				&writeTo{[]byte{}},
			},
			&Buffer{storage: []byte{0}, size: 1},
		},
		{
			nil, 2,
			[]operation{
				&writeTo{[]byte{}},
			},
			&Buffer{storage: []byte{0, 0}, size: 2},
		},
		{
			&Buffer{storage: []byte{1}, size: 1, used: 1}, 0,
			[]operation{
				&writeTo{[]byte{1}},
			},
			&Buffer{storage: []byte{1}, size: 1},
		},
		{
			&Buffer{storage: []byte{1, 2}, size: 2, used: 2}, 0,
			[]operation{
				&writeTo{[]byte{1, 2}},
			},
			&Buffer{storage: []byte{1, 2}, size: 2},
		},
		{
			&Buffer{storage: []byte{1, 0, 0}, size: 3, used: 1}, 0,
			[]operation{
				&writeTo{[]byte{1}},
			},
			&Buffer{storage: []byte{1, 0, 0}, size: 3},
		},
		{
			&Buffer{storage: []byte{0, 1, 2}, size: 3, start: 1, used: 2}, 0,
			[]operation{
				&writeTo{[]byte{1, 2}},
			},
			&Buffer{storage: []byte{0, 1, 2}, size: 3},
		},
		{
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4, start: 2, used: 4}, 0,
			[]operation{
				&writeTo{[]byte{1, 2, 3, 4}},
			},
			&Buffer{storage: []byte{3, 4, 1, 2}, size: 4},
		},
		{
			&Buffer{storage: []byte{0, 1, 2, 0}, size: 4, start: 1, used: 2}, 0,
			[]operation{
				&writeTo{[]byte{1, 2}},
			},
			&Buffer{storage: []byte{0, 1, 2, 0}, size: 4},
		},
		{
			&Buffer{storage: []byte{3, 4, 0, 0, 1, 2}, size: 6, start: 4, used: 4}, 0,
			[]operation{
				&writeTo{[]byte{1, 2, 3, 4}},
			},
			&Buffer{storage: []byte{3, 4, 0, 0, 1, 2}, size: 6},
		},
	}

	// Process test cases.
ProcessTests:
	for i, test := range tests {
		// If an initial buffer has been provided, then use that, otherwise
		// create one of the specified size and ensure that it's non-nil.
		buffer := test.buffer
		if buffer == nil {
			buffer = NewBuffer(test.size)
			if buffer == nil {
				t.Errorf("test index %d: newly create buffer is nil", i)
				continue
			}
		}

		// Perform the requested operations.
		for o, operation := range test.operations {
			if err := operation.perform(buffer); err != nil {
				t.Errorf("test index %d, operation index %o: unexpected error: %s", i, o, err)
				continue ProcessTests
			}
		}

		// Compare the resulting buffer state with the expected value.
		var invalid bool
		if len(buffer.storage) != len(test.expected.storage) {
			t.Errorf("test index %d: resulting buffer storage size does not match expected: %d != %d",
				i, len(buffer.storage), len(test.expected.storage),
			)
			invalid = true
		} else if !bytes.Equal(buffer.storage, test.expected.storage) {
			t.Errorf("test index %d: resulting buffer storage does not match expected", i)
			invalid = true
		}
		if buffer.size != test.expected.size {
			t.Errorf("test index %d: resulting cached buffer size does not match expected: %d != %d",
				i, buffer.size, test.expected.size,
			)
			invalid = true
		}
		if buffer.start != test.expected.start {
			t.Errorf("test index %d: resulting buffer start index does not match expected: %d != %d",
				i, buffer.start, test.expected.start,
			)
			invalid = true
		}
		if buffer.used != test.expected.used {
			t.Errorf("test index %d: resulting buffer data count does not match expected: %d != %d",
				i, buffer.used, test.expected.used,
			)
			invalid = true
		}

		// If the buffer was invalid, then continue.
		if invalid {
			continue
		}

		// Otherwise, perform generic invariant checks.
		if bs := buffer.Size(); bs != buffer.size {
			t.Errorf("test index %d: size accessor returned incorrect value: %d != %d",
				i, bs, buffer.size,
			)
			invalid = true
		}
		if bu := buffer.Used(); bu != buffer.used {
			t.Errorf("test index %d: used accessor returned incorrect value: %d != %d",
				i, bu, buffer.used,
			)
			invalid = true
		}
		if ba := buffer.Available(); ba != (buffer.size - buffer.used) {
			t.Errorf("test index %d: available accessor returned incorrect value: %d != %d",
				i, ba, buffer.size-buffer.used,
			)
			invalid = true
		}
		if buffer.size > 0 && buffer.start >= buffer.size {
			t.Errorf("test index %d: buffer start index invalid: %d >= %d",
				i, buffer.start, buffer.size,
			)
		}
		if buffer.used > buffer.size {
			t.Errorf("test index %d: buffer data count invalid: %d > %d",
				i, buffer.used, buffer.size,
			)
		}

		// If invariants were invalid, then continue.
		if invalid {
			continue
		}

		// Otherwise, perform a reset test.
		buffer.Reset()
		if buffer.start != 0 {
			t.Errorf("test index %d: buffer start index non-0 after reset: %d", i, buffer.start)
		}
		if buffer.used != 0 {
			t.Errorf("test index %d: buffer data count non-0 after reset: %d", i, buffer.used)
		}
	}
}
