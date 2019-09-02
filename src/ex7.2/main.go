package main

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
)

var (
	log, _ = zap.NewDevelopment()
)

type ByteCounter struct {
	count int64
	wr    io.Writer
}

// The CountingWriter is a writer that counts the total number of bytes written.
func CountingWriter(w io.Writer) (io.Writer, *int64) {
	bc := ByteCounter{wr: w, count: 0}
	return &bc, &bc.count
}

// Write to the underlying writer, counting the total number of bytes written.
func (bc *ByteCounter) Write(p []byte) (int, error) {
	n, err := bc.wr.Write(p)
	bc.count += int64(n)
	return n, err
}

func main() {
	writeThis := "Hello World\n"

	// Create a wrapper around stdout
	writer, count := CountingWriter(os.Stdout)

	// Write some content to the writer
	if _, err := fmt.Fprint(writer, "Hello world\n"); err != nil {
		log.Fatal("Failed to write", zap.Error(err))
	}

	// The string length should match the bytes written
	log.Sugar().Infof("%d bytes written of %d", *count, len(writeThis))
	if int64(len(writeThis)) != *count {
		log.Fatal("Byte counts are different")
	}
}
