package isal
//#cgo LDFLAGS: -L/usr/bin -lisal
//#include <igzip_lib.h>
//#include <isal_native.h>
//#include <stdio.h>
//#include <stdlib.h>
//#include <stdint.h>
import "C"

import (
	"fmt"
	"io"
	"unsafe"
	"errors"
)

const (
	BUF_SIZE = 8*1024
	HAS_GZIP_HEADER = 0
	DEFAULT_LEVEL = 1
)

type zstream [unsafe.Sizeof(C.isal_zstream{})]C.char
type inf_state [unsafe.Sizeof(C.inflate_state{})]C.char
type isalgzheader [unsafe.Sizeof(C.isal_gzip_header{})]C.char

// Reader is a gzip/zlib/flate reader. It implements io.ReadCloser.  Calling
// Close is optional, though strongly recommended.  NewReader() also installs a
// GC finalizer that closes the Reader, in case the application forgets to call
// Close.
type Reader struct {
	in          io.Reader
	zs          inf_state
	inEOF       bool    // true if in reaches io.EOF
	gzHeader    isalgzheader
	hasGzHeader bool    // true if gzHeader was successfully set.
	inBuf       []byte
	err         error
}
// NewReader creates a gzip/flate reader. There can be at most one options arg.
func NewReader(in io.Reader) (*Reader) {

	z := &Reader{
		in:         in,
		inBuf:      make([] byte, BUF_SIZE),
	}
	if ec := C.ig_isal_inflate_init(&z.zs[0]); ec != 0 {
		return nil
	}
	return z
}



// Writer is the gzip/flate writer. It implements io.WriterCloser.
type Writer struct {
	out      io.Writer
	zs       zstream // underlying zlib implementation.
	gzHeader isalgzheader
	outBuf   []byte
	level    int
	err         error
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
	z, _ := NewWriterLevel(w, DEFAULT_LEVEL)
	return z, nil
}

// NewWriterLevel is like NewWriter but specifies the compression level instead
// of assuming DefaultCompression.
//
// The compression level can be DefaultCompression, NoCompression, HuffmanOnly
// or any integer value between BestSpeed and BestCompression inclusive.
// The error returned will be nil if the level is valid.



func NewWriterLevel(w io.Writer, level int) (*Writer, error) {

	z := &Writer{
		out:    w,
		outBuf: make([]byte, 512*1024),
		level: level,
	}

	if (level < 0 || level > 3) {
		return z, errors.New("isal.gzip: invalid compression level")
	}

	ec := C.ig_isal_deflate_init(&z.zs[0],C.int(level))

	if HAS_GZIP_HEADER ==1 {
		C.ig_isal_gzip_header_init(&z.gzHeader[0]);
	}

	if ec != 0 {
		return nil, isalReturnCodeToError(ec)
	}
	return z, nil
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

	if( len(z.outBuf) > len(in)) {


		ret := C.ig_isal_deflate_stateless(&z.zs[0],(*C.uint8_t)(unsafe.Pointer(&in[0])), C.int(len(in)), (*C.uint8_t)(unsafe.Pointer(&z.outBuf[0])), &outLen, &inConsumed, HAS_GZIP_HEADER, &z.gzHeader[0])

		if ret != 0 {
			return 0, isalReturnCodeToError(ret)
		}

		nOut := len(z.outBuf) - int(outLen)
		//	fmt.Printf("\n %d len(z.outBuf) %d outLen %d nOut", len(z.outBuf), outLen, nOut) 
		if err := z.flush(z.outBuf[:nOut]); err != nil {
			return 0, err
		}
		return nOut, nil

	} else {
		ec := C.ig_isal_deflate_init(&z.zs[0],C.int(z.level))
		if ec != 0 {
			return 0, isalReturnCodeToError(ec)
		}

		if HAS_GZIP_HEADER ==1 {
			C.ig_isal_gzip_header_init(&z.gzHeader[0]);
		}



		outLen = C.int(len(z.outBuf))

		infile_size := C.int(len(in))

		end_of_stream := C.int(0)
		state := C.int(0)
		for {


			//fmt.Printf("%d infilesize\n", int(infile_size))
			avail_in := C.int(BUF_SIZE)
			if(infile_size < avail_in) {
				end_of_stream =  1;
				avail_in = infile_size;
			}

			for {
				avail_out := C.int(BUF_SIZE)

				C.ig_isal_deflate(&z.zs[0],(*C.uint8_t)(unsafe.Pointer(&in[0])),(*C.uint8_t)(unsafe.Pointer(&z.outBuf[0])),&avail_out, &end_of_stream, &state, &avail_in, HAS_GZIP_HEADER, &z.gzHeader[0])
				//	fmt.Printf("%d avail_out", int(avail_out))

				nOut := BUF_SIZE - avail_out
				//	 fmt.Printf("%d nOut\n", nOut)

				if nOut !=0 {
					length = length + int(nOut)
					if err := z.flush(z.outBuf[:nOut]); err != nil {
						return 0, err
					}

				}

				if avail_out !=0 {
					break
				}
			}

			infile_size = infile_size - BUF_SIZE
			if state !=0 {
				break
			}


		}

		return length, nil
	}

}


// Read implements io.Reader, reading uncompressed bytes from its underlying Reader.
func (z *Reader) Read(p []byte) (n int) {

	if z.err != nil {
		return 0
	}

	outLen := C.int(len(p))

	inbytes,err := z.in.Read(z.inBuf)

	  state := C.int(0)
        avail_in := C.int(inbytes)


//	fmt.Printf("%d inbytes %d len(z.inBuf)\n", inbytes, len(z.inBuf))

	if inbytes < len(z.inBuf) {


		ret := C.ig_isal_inflate_stateless(&z.zs[0],(*C.uint8_t)(unsafe.Pointer(&z.inBuf[0])), C.int(inbytes),
		(*C.uint8_t)(unsafe.Pointer(&p[0])), &outLen, &state, &avail_in, HAS_GZIP_HEADER,&z.gzHeader[0] )

		if ret != 0 {
			z.err = isalReturnCodeToError(ret)
		}

	} else {

		for  {
			avail_in := C.int(inbytes)
			ret := C.ig_isal_inflate(&z.zs[0],(*C.uint8_t)(unsafe.Pointer(&z.inBuf[0])), C.int(inbytes),
			(*C.uint8_t)(unsafe.Pointer(&p[0])), &outLen,&state, &avail_in, HAS_GZIP_HEADER,&z.gzHeader[0] )

			if ret != 0 {
				z.err = isalReturnCodeToError(ret)
			}
			 if( err == io.EOF ){

                                break
                        }


			inbytes, err = z.in.Read(z.inBuf)

		}

	}

//		fmt.Printf("%d befor return \n",len(p) - int(outLen))
	return (len(p) - int(outLen))

}




// Close implements io.Closer
func (z *Reader) Close() error {

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

	return nil
}





func isalReturnCodeToError(r C.int) error {
	if r == 0 {
		return nil
	}
	if r == C.ISAL_END_INPUT {
		return fmt.Errorf("isal: End of input reached%d", r)
	}
	if r == C.ISAL_INVALID_LEVEL {
		return fmt.Errorf("isal: End of input reached%d", r)
	}
	if r == C.ISAL_INVALID_LEVEL_BUF {
		return fmt.Errorf("isal: End of input reached%d", r)
	}
	if r == C.STATELESS_OVERFLOW {
		return fmt.Errorf("isal: End of input reached%d", r)
	}

	return fmt.Errorf("isal: unknown error %d", r)
}

