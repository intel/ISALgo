package isal

//#cgo LDFLAGS: -ldl
//#include "igzip_lib.h"
//#include <isal_native.h>
//#include <stdio.h>
//#include <stdlib.h>
//#include <stdint.h>
import "C"

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"unsafe"
)

var errReaderClosed = errors.New("Reader is closed")
var errCouldNotLoadLib = errors.New("could not load isal library")

const (
	D_BUF_SIZE      = 640 * 1024
	C_BUF_SIZE      = 128 * 1024
	HAS_GZIP_HEADER = 1
	DEFAULT_LEVEL   = 0
)

// Variable to check if library is loaded
var LIB_LOADED = 0

// cPool is a pool of buffers for use in reader.compressionBuffer. Buffers are
// taken from the pool in NewReaderDict, returned in reader.Close(). Returns a
// pointer to a slice to avoid the extra allocation of returning the slice as a
// value.
var cPool = sync.Pool{
	New: func() interface{} {
		buff := make([]byte, D_BUF_SIZE)
		return &buff
	},
}

// dPool is a pool of buffers for use in reader.decompressionBuffer. Buffers are
// taken from the pool in NewReaderDict, returned in reader.Close(). Returns a
// pointer to a slice to avoid the extra allocation of returning the slice as a
// value.
var dPool = sync.Pool{
	New: func() interface{} {
		buff := make([]byte, 12*D_BUF_SIZE)
		return &buff
	},
}

func resize(in []byte, newSize int) []byte {
	if in == nil {
		return make([]byte, newSize)
	}
	if newSize <= cap(in) {
		return in[:newSize]
	}
	toAdd := newSize - len(in)
	return append(in, make([]byte, toAdd)...)
}

type zstream [unsafe.Sizeof(C.isal_zstream{})]C.char
type inf_state [unsafe.Sizeof(C.inflate_state{})]C.char
type isalgzheader [unsafe.Sizeof(C.isal_gzip_header{})]C.char

// Reader is a gzip/zlib/flate reader. It implements io.ReadCloser.  Calling
// Close is optional, though strongly recommended.  NewReader() also installs a
// GC finalizer that closes the Reader, in case the application forgets to call
// Close.
type Reader struct {
	underlyingReader     io.Reader
	zs                   inf_state
	inEOF                bool // true if in reaches io.EOF
	firstError           error
	gzHeader             isalgzheader
	hasGzHeader          bool // true if gzHeader was successfully set.
	compressionBuffer    []byte
	decompressionBuffer  []byte
	tempDecompressBuffer []byte
	decompOff            int
	compressionLeft      int
	remaining            int
	previous             int
	err                  error
}

// NewReader creates a gzip/flate reader. There can be at most one options arg.
func NewReader(in io.Reader) (*Reader, error) {

	var err error
	var ready bool

	// load the isal library if not loaded by Ready()
	if LIB_LOADED == 0 {
		ready = Ready()
		if !ready {
			return nil, errCouldNotLoadLib
		}
	}
	compressionBufferP := cPool.Get().(*[]byte)
	z := &Reader{
		underlyingReader:  in,
		compressionBuffer: *compressionBufferP,
		firstError:        err,
		compressionLeft:   0,
		previous:          0,
		remaining:         0,
	}
	if ec := C.ig_isal_inflate_init(&z.zs[0]); ec != 0 {
		return nil, isalReturnCodeToError(ec)
	}

	return z, nil
}

// Writer is the gzip/flate writer. It implements io.WriterCloser.
type Writer struct {
	out      io.Writer
	zs       zstream // underlying zlib implementation.
	gzHeader isalgzheader
	outBuf   []byte
	level    int
	err      error
}

//NewWriter returns a new Writer.
// Writes to the returned writer are compressed and written to w.
//
// It is the caller's responsibility to call Close on the Writer when done.
// Writes may be buffered and not flushed until Close.
//
// Callers that wish to set the fields in Writer.Header must do so before
// the first call to Write, Flush, or Close.

func NewWriter(w io.Writer) (*Writer, error) {
	z, err := NewWriterLevel(w, DEFAULT_LEVEL)
	return z, err
}

// NewWriterLevel is like NewWriter but specifies the compression level instead
// of assuming DefaultCompression.
//
// The compression level can be DefaultCompression, NoCompression, HuffmanOnly
// or any integer value between BestSpeed and BestCompression inclusive.
// The error returned will be nil if the level is valid.

func NewWriterLevel(w io.Writer, level int) (*Writer, error) {
	var ready bool
	if LIB_LOADED == 0 {
		ready = Ready()
		if !ready {
			return nil, errCouldNotLoadLib
		}
	}

	z := &Writer{
		out:    w,
		outBuf: make([]byte, C_BUF_SIZE),
		level:  level,
	}

	if level < 0 || level > 3 {
		return z, errors.New("isal.gzip: invalid compression level")
	}

	ec := C.ig_isal_deflate_init(&z.zs[0], C.int(level))

	if HAS_GZIP_HEADER == 1 {
		C.ig_isal_gzip_header_init(&z.gzHeader[0])
	}

	if ec != 0 {
		return nil, isalReturnCodeToError(ec)
	}
	return z, nil
}

//Adds a Ready() API that will check if the ISAL library is loadable and will load it
//If NewWriter/NewReader or NewWriterLevel is called before Ready, initialize the dynamic loads using Ready()

func Ready() bool {

	// load the isal library and the symbols
	ret := C.isal_dload_functions()
	if ret != 0 {
		fmt.Printf("isal: library could not be dynamically loaded %d", 1)
		return false
	} else {
		LIB_LOADED = 1
	}

	return true
}

// Write implements io.Writer.
func (z *Writer) Write(in []byte) (int, error) {

	if len(in) == 0 {
		return 0, nil
	}
	var (
		outLen     = C.int(len(z.outBuf))
		inConsumed C.int
		length     int
	)

	if len(z.outBuf) > len(in) {

		ret := C.ig_isal_deflate_stateless(&z.zs[0], (*C.uint8_t)(unsafe.Pointer(&in[0])), C.int(len(in)), (*C.uint8_t)(unsafe.Pointer(&z.outBuf[0])), &outLen, &inConsumed, HAS_GZIP_HEADER, &z.gzHeader[0])

		if ret != 0 {
			return 0, isalReturnCodeToError(ret)
		}

		nOut := len(z.outBuf) - int(outLen)
		if err := z.flush(z.outBuf[:nOut]); err != nil {
			return 0, err
		}
		return len(in), nil

	} else {
		ec := C.ig_isal_deflate_init(&z.zs[0], C.int(z.level))
		if ec != 0 {
			return 0, isalReturnCodeToError(ec)
		}

		if HAS_GZIP_HEADER == 1 {
			C.ig_isal_gzip_header_init(&z.gzHeader[0])
		}

		outLen = C.int(len(z.outBuf))

		infile_size := C.int(len(in))

		end_of_stream := C.int(0)
		state := C.int(0)
		for {

			avail_in := C.int(C_BUF_SIZE)
			if infile_size < avail_in {
				end_of_stream = 1
				avail_in = infile_size
			}

			for {
				avail_out := C.int(C_BUF_SIZE)

				C.ig_isal_deflate(&z.zs[0], (*C.uint8_t)(unsafe.Pointer(&in[0])), (*C.uint8_t)(unsafe.Pointer(&z.outBuf[0])), &avail_out, &end_of_stream, &state, &avail_in, HAS_GZIP_HEADER, &z.gzHeader[0])

				nOut := C_BUF_SIZE - avail_out

				if nOut != 0 {
					length = length + int(nOut)
					if err := z.flush(z.outBuf[:nOut]); err != nil {
						return 0, err
					}

				}

				if avail_out != 0 {
					break
				}
			}

			infile_size = infile_size - C_BUF_SIZE
			if state != 0 {
				break
			}

		}

		return len(in), nil
	}

}

// Read implements io.Reader, reading uncompressed bytes from its underlying Reader.
func (z *Reader) Read(p []byte) (n int, err error) {

	if z.err != nil {
		return 0, z.err
	}
	if z.remaining > 0 {

		n = copy(p, z.tempDecompressBuffer[z.decompOff:z.decompOff+z.remaining])
		z.decompOff += n
		z.remaining -= n
		if z.remaining <= 0 {
			return n, io.EOF
		} else {
			return n, nil
		}
	}
	avail_out := C.int(0)

	if len(p) < 1024 {
		if len(z.compressionBuffer) == 0 {
			compressionBufferP := cPool.Get().(*[]byte)
			z.compressionBuffer = *compressionBufferP
		}
		decompressionBufferP := dPool.Get().(*[]byte)
		tempDecompressBufferP := dPool.Get().(*[]byte)
		z.decompressionBuffer = *decompressionBufferP
		z.tempDecompressBuffer = *tempDecompressBufferP
		avail_out = C.int(len(z.decompressionBuffer))
	} else {
		runtime.GC()
		avail_out = C.int(len(p))

	}
	inbytes, err := z.underlyingReader.Read(z.compressionBuffer)

	state := C.int(0)
	totalOut := C.int(0)
	avail_in := C.int(inbytes)
	ret := C.int(0)
	prevOut := 0

	for {
		avail_in = C.int(inbytes)
		if len(p) > 1024 {
			ret = C.ig_isal_inflate(&z.zs[0], (*C.uint8_t)(unsafe.Pointer(&z.compressionBuffer[0])), C.int(inbytes),
				(*C.uint8_t)(unsafe.Pointer(&p[0])), &avail_out, &totalOut, &state, &avail_in, HAS_GZIP_HEADER, &z.gzHeader[0])
		} else {
			ret = C.ig_isal_inflate_buffered(&z.zs[0], (*C.uint8_t)(unsafe.Pointer(&z.compressionBuffer[0])), C.int(inbytes),
				(*C.uint8_t)(unsafe.Pointer(&z.decompressionBuffer[0])), &avail_out, &totalOut, &state, &avail_in, HAS_GZIP_HEADER, &z.gzHeader[0])
		}
		if ret != 0 {
			z.err = isalReturnCodeToError(ret)
			return 0, z.err
		}

		if len(p) < 1024 {
			n = copy(z.tempDecompressBuffer[prevOut:], z.decompressionBuffer[:int(totalOut)-prevOut])
			prevOut = prevOut + n
			avail_out = C.int(len(z.decompressionBuffer))
		}
		if err == io.EOF {

			break
		}

		if len(p) < 1024 {
			if len(z.tempDecompressBuffer) < (len(z.decompressionBuffer) + int(totalOut)) {
				z.tempDecompressBuffer = append(z.tempDecompressBuffer, make([]byte, 12*D_BUF_SIZE)...)
			}
		}

		inbytes, err = z.underlyingReader.Read(z.compressionBuffer)
		if inbytes == 0 {
			break

		}

	}

	if len(p) < 1024 {
		z.decompOff = copy(p, z.tempDecompressBuffer[:int(totalOut)])
		if z.decompOff < int(totalOut) {
			z.remaining = int(totalOut) - z.decompOff
		}
		if z.decompOff < int(totalOut) {
			return z.decompOff, nil
		} else {
			return z.decompOff, io.EOF
		}
	} else {
		return int(totalOut), err
	}

}

// Close implements io.Closer
func (z *Reader) Close() error {

	if z.firstError != nil {
		return z.firstError
	}

	cb := z.compressionBuffer
	db := z.decompressionBuffer
	// Ensure that we won't resuse buffer
	z.firstError = errReaderClosed
	z.compressionBuffer = nil
	z.decompressionBuffer = nil

	cPool.Put(&cb)
	dPool.Put(&db)

	return nil

}

// Flush writes the data to the output.
func (z *Writer) flush(data []byte) error {
	n, err := z.out.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) { // shouldn't happen in practice
		return fmt.Errorf("zlib: n=%d, outLen=%d", n, len(data))
	}
	return nil
}

// Close implements io.Closer
func (z *Writer) Close() error {

	if z.err != nil {
		return z.err
	}

	return nil
}

// Reset discards the Writer z's state and makes it equivalent to the
// result of its original state from NewWriter or NewWriterLevel, but
// writing to w instead. This permits reusing a Writer rather than
// allocating a new one.
func (z *Writer) Reset(w io.Writer) error {

	z = &Writer{
		out:   w,
		level: z.level,
	}

	C.ig_isal_deflate_reset(&z.zs[0])
	return z.err

}

func (z *Reader) Reset(r io.Reader) error {

	z.underlyingReader = r
	C.ig_isal_inflate_init(&z.zs[0])
	return z.err

}

func isalReturnCodeToError(r C.int) error {
	if r == 0 {
		return nil
	}
	if r == C.ISAL_END_INPUT {
		return fmt.Errorf("isal: End of input reached%d", r)
	}
	if r == C.ISAL_INVALID_LEVEL {
		return fmt.Errorf("isal: invalid level passed %d", r)
	}
	if r == C.ISAL_INVALID_LEVEL_BUF {
		return fmt.Errorf("isal: isal_invalid_level_buf %d", r)
	}
	if r == C.STATELESS_OVERFLOW {
		return fmt.Errorf("isal: stateless overflow %d", r)
	}

	return fmt.Errorf("isal: unknown error %d", r)
}
