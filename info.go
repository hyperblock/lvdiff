package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func showInfo(path string, verbose bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	var buf [SchemeV1HeaderLength]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return err
	}

	var header streamHeader
	if err := header.UnmarshalBinary(buf[:]); err != nil {
		return err
	}

	var typ string
	switch header.StreamType {
	case StreamTypeFull:
		typ = "full"
	case StreamTypeDelta:
		typ = "incremental"
	default:
		return fmt.Errorf("unknown stream type %d", header.StreamType)
	}

	fmt.Printf("Scheme Ver : %d\n", header.SchemeVersion)
	fmt.Printf("Stream Type: %s\n", typ)
	fmt.Printf("Volume Size: %d\n", header.VolumeSize)
	fmt.Printf("Block Size : %d\n", header.BlockSize)
	fmt.Printf("Block Count: %d\n", header.BlockCount)
	fmt.Printf("Volume UUID: %s\n", string(header.VolumeUUID[:]))
	if header.StreamType == StreamTypeDelta {
		fmt.Printf("Source UUID: %s\n", string(header.VolumeUUID[:]))
	}

	if !verbose {
		return nil
	}

	fmt.Printf("Blocks List:\n")

	n := 8 + header.BlockSize
	b := make([]byte, n+md5.Size)
	h := md5.New()

	for i := uint64(0); i < header.BlockCount; i++ {
		binary.BigEndian.PutUint64(b, 0)

		if _, err := io.ReadFull(f, b); err != nil {
			return err
		}

		h.Reset()
		h.Write(b[:n])
		if !bytes.Equal(h.Sum(nil), b[n:n+md5.Size]) {
			return fmt.Errorf("check sum mismatch for %dst block", i)
		}

		index := binary.BigEndian.Uint64(b)
		fmt.Printf("  i=%d, block_index=%d\n", i, index)
	}

	return nil
}
