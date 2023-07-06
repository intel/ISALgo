# isal-cgo Go wrapper library for the Intel(R) Intelligent Storage Acceleration Library (ISA-L)

This ISA-L wrapper allows Go applications to make use of the optimized low-level functions provided by the Intel(R) ISA-L library including gzip, compression and decompression. <br>

The performance gains are obtained for in-memory workloads only.  Streaming is not supported.

For full details on the ISA-L compression performance (C-library) refer the ISAL library [github page](https://github.com/intel/isa-l) <br>

# Table of Contents
- Features
- Installation
  - Prerequisites (cgo)
  - Download and Installation
- Usage
  - Compress
  - Decompress
- Notes

# Features
Industry leading gzip, and raw deflate compression / decompression <br>
Convenience functions for quicker one-time compression / decompression <br>
Supports compression levels 0 through 3 for better compression ratios and performance <br>
Simple implementation. Supports go Reader/Writer API and offers:<br>
 /gzip/deflate compression <br>
 /gzip/Inflate decompression <br>
 Decompression w/ info about number of compressed bytes and uncompressed bytes. <br>

# Installation
## Prerequisites (working cgo)
This library makes use of Golang's CGO functionality which requires the 64-bit version of the GCC compiler to be installed on the development machine.  Install the latest 64-bit GCC if necessary. <br>

## Download and Install Intel(r) ISA-L
Install the Intel(R) ISAL (Intelligent Storage Libarary) available here: (https://github.com/intel/isa-l) <br>
Once installed, export LD_LIBRARY_PATH as appropriate to allow the CGO build process to link it. Set the include path to point to igzip_lib.h provided by this wrapper. <br>

export LD_LIBRARY_PATH = --path-to-installed-isal/bin <br>
CGO_CFLAGS="-I/--path-to-installed-isal--/include/ -L/--path-to-installed-isal--/bin" <br>
It is essential that CGO is enabled and the latest version of ISA-L (currently 2.30.0) installed before proceeding. <br><br>
## Initialize isal module
Use "go mod init" to initialize the isal module to use for your application. Instructions to initialize the module are available in go help documentation. <br>

# Usage
## Compress (Deflate)
Create a compressor that can be used for any type of compression (gzip or deflate compatible) <br>

Specify the desired level of compression from 1 through 3.  ***Only these 3 compression levels are supported <br> 
Note that, high levels provide higher compression at the expense of speed.  Lower levels provide lower compression at higher speed.<br>

// Compressor with default compression level. Errors if out of memory, supports Go Writer <br>
w, err := isal.NewWriter(buffer) <br><br>

// Compressor with custom compression level. Errors if out of memory or if an illegal level was passed. <br>
w, err = isal.NewCompressorLevel(buffer, 2) <br> <br>
Now compress the actual data with a given mode of compression (currently supported: gzip, raw deflate): <br>

// Use the compressor to actually compress the data. Uses the go API write function <br>
w.Write(string) <br> <br>

// Close the writer <br>
w.Close()<br><br>

## Decompress (Inflate)

As with compression, create a decompressor.<br>

// Decompressor; works for all compression levels. Errors if out of memory. Supports Go Reader API <br>
r, err := isal.NewReader(buf) <br> <br>

// Using "defer r.Close" at the top so that the developer does not need to remember to call r.Close() at the end <br>
defer r.Close() <br><br>

//supply a buffer for decompressed string <br>
s := make([]byte, len(decomp)) <br><br>

// Decompress the actual data (currently supported: gzip, raw deflate): <br>
// Supports Go Read function from Reader API, the Read function returns the number of uncompressed bytes <br>
decompressed, err = r. Read(s) <br><br>


## Notes

Code supports both gzip and flate formats. By default the wrapper code and test code supports flate. In order to enable the code to support gzip "HAS_GZIP_HEADER" flag needs to be set to "1" in the isal_cgo.go file. <br>

Always Close() the Compressor / Decompressor when finished using it - especially if you create a new compressor/decompressor for each compression/decompression you undertake (which is generally discouraged anyway). As the C-part of this library is not subject to the Go garbage collector, the memory allocated by it must be released manually (by a call to Close()) to avoid memory leakage. <br>

isal_test.go is provided. It tests the package functionality. It also runs the go-benchmarks for level 1,2,and 3 for 3 different files. It benchmarks both inflate and deflate aka. decode and encode. <br>

