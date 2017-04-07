package multiplex

import (
	"bufio"
	"encoding/gob"
	"io"
	"os"
	"testing"
)

// benchmarkMessageSend underlies all message sending benchmarks and contains
// the primary benchmarking implementation. The individual benchmarks set up the
// streams they want to test and then pass them to this function, which performs
// gob-encoded message sending across the stream. It sends a message that
// approximates an rsync operation message of ~50kB in size.
func benchmarkMessageSend(b *testing.B, reader io.Reader, writer io.Writer, closers ...io.Closer) {
	// Defer closure of all closers to make sure that the receiver Goroutine is
	// terminated.
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	// Create encoder and decoder.
	encoder := gob.NewEncoder(writer)
	decoder := gob.NewDecoder(reader)

	// Start our receiver.
	go func() {
		var received testMessage
		for {
			if decoder.Decode(&received) != nil {
				break
			}
		}
	}()

	// Create our message. We try to create something approximating an rsync
	// operation message in terms of size.
	operation := testMessage{
		Index:   1234,
		Values:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		Bytes:   make([]byte, 50000),
		Message: "this is a sample message",
	}

	// Send an initial message to avoid the cost of building the gob "compiler."
	encoder.Encode(operation)

	// Reset the benchmark timer to exclude the setup time.
	b.ResetTimer()

	// Perform the benchmark.
	for i := 0; i < b.N; i++ {
		if err := encoder.Encode(operation); err != nil {
			b.Fatal("unable to encode message:", err)
		}
	}
}

// BenchmarkMessageSendIOPipe benchmarks messaging over a raw io.Pipe.
func BenchmarkMessageSendIOPipe(b *testing.B) {
	// Create our transport.
	reader, writer := io.Pipe()

	// Benchmark.
	benchmarkMessageSend(b, reader, writer, writer)
}

// BenchmarkMessageSendIOPipeBuffered benchmarks messaging over a buffered
// io.Pipe.
func BenchmarkMessageSendIOPipeBuffered(b *testing.B) {
	// Create our transport.
	reader, writer := io.Pipe()
	bufferedReader := bufio.NewReader(reader)
	bufferedWriter := bufio.NewWriter(writer)

	// Benchmark.
	benchmarkMessageSend(b, bufferedReader, bufferedWriter, writer)
}

// BenchmarkMessageSendOSPipe benchmarks messaging over a raw OS pipe.
func BenchmarkMessageSendOSPipe(b *testing.B) {
	// Create our transport.
	reader, writer, err := os.Pipe()
	if err != nil {
		b.Fatal("unable to create OS pipe:", err)
	}

	// Benchmark.
	benchmarkMessageSend(b, reader, writer, writer)
}

// BenchmarkMessageSendOSPipeBuffered benchmarks messaging over a buffered
// OS pipe.
func BenchmarkMessageSendOSPipeBuffered(b *testing.B) {
	// Create our transport.
	reader, writer, err := os.Pipe()
	if err != nil {
		b.Fatal("unable to create OS pipe:", err)
	}
	bufferedReader := bufio.NewReader(reader)
	bufferedWriter := bufio.NewWriter(writer)

	// Benchmark.
	benchmarkMessageSend(b, bufferedReader, bufferedWriter, writer)
}

// BenchmarkMessageSendIOPipeMultiplexed benchmarks messaging over a multiplexed
// io.Pipe.
func BenchmarkMessageSendIOPipeMultiplexed(b *testing.B) {
	// Create an underlying transport.
	reader, writer := io.Pipe()

	// Multiplex.
	readers, readerCloser := Reader(reader, 1)
	writers := Writer(writer, 1)

	// Benchmark.
	benchmarkMessageSend(b, readers[0], writers[0], writer, readerCloser)
}

// BenchmarkMessageSendOSPipeMultiplexed benchmarks messaging over a multiplexed
// OS pipe.
func BenchmarkMessageSendOSPipeMultiplexed(b *testing.B) {
	// Create an underlying transport.
	reader, writer, err := os.Pipe()
	if err != nil {
		b.Fatal("unable to create OS pipe:", err)
	}

	// Multiplex.
	readers, readerCloser := Reader(reader, 1)
	writers := Writer(writer, 1)

	// Benchmark.
	benchmarkMessageSend(b, readers[0], writers[0], writer, readerCloser)
}
