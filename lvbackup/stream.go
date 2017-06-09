package lvbackup

import (
	"crypto/md5"
)

const (
	StreamSchemeV1 = 1

	StreamTypeFull  = 1
	StreamTypeDelta = 2

	SchemeV1HeaderLength = 94 + md5.Size
)

const C_HEAD = "HYPERLAYER/1.0\n"

type streamHeader struct {
	//	SchemeVersion uint8  // scheme version
	//StreamType    uint8  // type of the stream: full or delta
	Name       string `yaml:"Name"`
	VolumeSize uint64 `yaml:"Volume size"`  // full size of logical volume
	BlockSize  uint32 `yaml:"Chunk size"`   // block size of logical volume
	BlockCount uint64 `yaml:"Delta blocks"` // how many blocks in the stream
	//VolumeUUID    [36]byte `yaml:"UUID"` // UUID of logical volume
	VolumeUUID string `yaml:"VolumeUUID"`
	//	DeltaSourceUUID [36]byte       // only for delta stream
	DeltaSourceUUID string `yaml:"Backing volumeUUID"` // only for delta stream
	//	CheckSum        [md5.Size]byte // MD5 hash of stream header except this field
}

// func (h *streamHeader) MarshalBinary() ([]byte, error) {
// 	var b [SchemeV1HeaderLength]byte

// 	buf := bytes.NewBuffer(b[:0])

// 	buf.WriteByte(h.SchemeVersion)
// 	buf.WriteByte(h.StreamType)

// 	binary.Write(buf, binary.BigEndian, h.VolumeSize)
// 	binary.Write(buf, binary.BigEndian, h.BlockSize)
// 	binary.Write(buf, binary.BigEndian, h.BlockCount)

// 	buf.Write(h.VolumeUUID[:])
// 	buf.Write(h.DeltaSourceUUID[:])

// 	hash := md5.Sum(buf.Bytes())
// 	buf.Write(hash[:])

// 	if buf.Len() != SchemeV1HeaderLength {
// 		panic("why binary stream header length is wrong?")
// 	}

// 	return b[:], nil
// }

// func (h *streamHeader) UnmarshalBinary(b []byte) error {
// 	if b[0] != StreamSchemeV1 {
// 		return fmt.Errorf("invalid scheme version %d", b[0])
// 	}

// 	if len(b) != SchemeV1HeaderLength {
// 		return errors.New("length of stream header data is wrong")
// 	}

// 	n := SchemeV1HeaderLength - md5.Size
// 	sum := md5.Sum(b[:n])
// 	if !bytes.Equal(sum[:], b[n:n+md5.Size]) {
// 		return errors.New("stream header hash mismatch")
// 	}

// 	buf := bytes.NewReader(b[:n])
// 	h.SchemeVersion, _ = buf.ReadByte()
// 	h.StreamType, _ = buf.ReadByte()

// 	binary.Read(buf, binary.BigEndian, &h.VolumeSize)
// 	binary.Read(buf, binary.BigEndian, &h.BlockSize)
// 	binary.Read(buf, binary.BigEndian, &h.BlockCount)

// 	buf.Read(h.VolumeUUID[:])
// 	buf.Read(h.DeltaSourceUUID[:])
// 	buf.Read(h.CheckSum[:])

// 	return nil
// }
