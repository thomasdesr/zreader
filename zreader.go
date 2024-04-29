// Copyright 2022 Chris Palmer, https://noncombatant.org/
// SPDX-License-Identifier: Apache-2.0

// Package zreader provides an [io.ReadCloser] for a variety of compression
// formats.
package zreader

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

// TODO: https://github.com/ulikunitz/xz
//
// TODO: Support all of: compress/{bzip2,flate,gzip,lzw,zlib}.
//
// TODO: Consider using more or all of the klauspost implementations.

type zType int

const (
	zNone zType = iota
	zBzip2
	zGzip
	zZip
	zZstd
)

// ZReader is an [io.ReadCloser] that reads compressed files.
//
// It currently supports bzip2, gzip, and zstd.
type ZReader struct {
	zType

	decompressor io.ReadCloser
}

// Open opens pathname and returns an appropriate ZReader. See [NewReader] for
// guidance on its behavior.
func Open(pathname string) (*ZReader, error) {
	file, e := os.Open(pathname)
	if e != nil {
		return nil, e
	}

	zr, err := NewReader(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return nil, closeErr
		}

		return nil, err
	}

	return zr, nil
}

// NewReader returns a ZReader for the given io.ReadCloser. It selects a
// decompressor based on the first few bytes of data. If it does not have a
// decompressor to match the bytes, subsequent calls to [Read] will return the
// raw bytes of the reader. (That might, or might not, be what you want.)
func NewReader(r io.Reader) (*ZReader, error) {
	return fromBufferedReader(bufio.NewReader(r))
}

func fromBufferedReader(uncompressed *bufio.Reader) (*ZReader, error) {
	magicBlock, err := uncompressed.Peek(magicBytePrefixSize)
	if err != nil {
		return nil, err
	}

	switch zTypeFromBytes(magicBlock) {
	case zBzip2:
		return &ZReader{zType: zBzip2, decompressor: io.NopCloser(bzip2.NewReader(uncompressed))}, nil
	case zGzip:
		r, e := gzip.NewReader(uncompressed)
		if e != nil {
			return nil, e
		}

		return &ZReader{zType: zGzip, decompressor: r}, nil
	case zZip:
		// TODO
		return &ZReader{zType: zZip, decompressor: io.NopCloser(uncompressed)}, nil
	case zZstd:
		d, e := zstd.NewReader(uncompressed)
		if e != nil {
			return nil, e
		}
		return &ZReader{zType: zZstd, decompressor: io.NopCloser(d)}, nil
	default:
		return &ZReader{zType: zNone, decompressor: io.NopCloser(uncompressed)}, nil
	}
}

// Read reads from the appropriate decompressor.
func (z *ZReader) Read(p []byte) (int, error) {
	return z.decompressor.Read(p)
}

// Close closes the ZReader, will close the underlying reader if it has one.
func (z *ZReader) Close(p []byte) error {
	return z.decompressor.Close()
}