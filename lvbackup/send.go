package lvbackup

import (
	"crypto/md5"
	"errors"
	"hash"
	"io"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/hyperblock/lvdiff/lvbackup/lvmutil"
	"github.com/hyperblock/lvdiff/lvbackup/thindelta"
	"github.com/hyperblock/lvdiff/lvbackup/vgcfg"

	"fmt"

	"github.com/ncw/directio"
)

type streamSender struct {
	vgname   string
	lvname   string
	srcname  string
	detectLv int

	header streamHeader
	blocks []thindelta.DeltaEntry

	w io.Writer
	h hash.Hash
}

func NewStreamSender(vgname, lvname, srcname string, w io.Writer, lv int) (*streamSender, error) {
	return &streamSender{
		vgname:   vgname,
		lvname:   lvname,
		srcname:  srcname,
		detectLv: lv,
		w:        w,
		h:        md5.New(),
	}, nil
}

func (s *streamSender) prepare() error {

	root, err := vgcfg.Dump(s.vgname)
	if err != nil {
		return err
	}

	var lv, srclv *vgcfg.ThinLvInfo
	var ok bool

	lv, ok = root.FindThinLv(s.lvname)

	if !ok {
		return errors.New("can not find thin lv " + s.lvname)
	}

	if len(s.srcname) > 0 {
		srclv, ok = root.FindThinLv(s.srcname)
		if !ok {
			return errors.New("can not find thin lv " + s.srcname)
		}

	}

	if srclv != nil && lv.Pool != srclv.Pool {
		return errors.New("thin volumes must in same pool for delta backup")
	}

	pool, ok := root.FindThinPool(lv.Pool)
	if !ok {
		return errors.New("can not find thin pool " + lv.Pool)
	}
	//fmt.Println(s.vgname, pool.MetaName)
	// dump block mapping
	tpoolDev := lvmutil.TPoolDevicePath(s.vgname, pool.Name)
	tmetaDev := lvmutil.LvDevicePath(s.vgname, pool.MetaName)
	deltaBlocks, count, err := thindelta.Delta(tpoolDev, tmetaDev, lv.DeviceId, srclv.DeviceId)
	if err != nil {
		fmt.Printf("thindump.Delta: %v\n", err)
		return err
	}

	s.blocks = deltaBlocks
	//	s.header.SchemeVersion = StreamSchemeV1
	//	s.header.StreamType = StreamTypeFull
	s.header.Name = lv.Name
	s.header.VolumeSize = uint64(lv.ExtentCount) * uint64(root.ExtentSize())
	s.header.BlockSize = uint32(pool.ChunkSize)
	s.header.VolumeUUID = lv.UUID
	s.header.BlockCount = uint64(count)
	s.header.DetectLevel = s.detectLv
	if srclv != nil {
		s.header.DeltaSourceUUID = srclv.UUID
	}

	return nil
}

func (s *streamSender) Run(header []byte) error {

	if err := s.prepare(); err != nil {
		fmt.Println(err.Error())
		return err
	}

	if err := s.putHeader(header); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}

	if len(s.srcname) > 0 {
		// always activate original lv so that target lv can be activated later
		if err := lvmutil.ActivateLv(s.vgname, s.srcname); err != nil {
			return err
		}
		defer lvmutil.DeactivateLv(s.vgname, s.srcname)
	}

	if err := lvmutil.ActivateLv(s.vgname, s.lvname); err != nil {
		return err
	}
	defer lvmutil.DeactivateLv(s.vgname, s.lvname)

	dstDevpath := lvmutil.LvDevicePath(s.vgname, s.lvname)
	srcDevpath := lvmutil.LvDevicePath(s.vgname, s.srcname)
	blockSize := int64(s.header.BlockSize)
	hashBlocks, err := thindelta.GenChecksum(srcDevpath, blockSize, s.blocks, s.detectLv)

	//fmt.Fprintln(os.Stderr, checksum)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}
	if err := s.putBaseBlocks(hashBlocks); err != nil {
		return nil
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}

	devFile, err := directio.OpenFile(dstDevpath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer devFile.Close()

	buf := directio.AlignedBlock(int(s.header.BlockSize))

	for _, e := range s.blocks {
		if e.OpType == thindelta.DeltaOpIgnore {
			continue
		}
		if e.OpType == thindelta.DeltaOpDelete {
			for i := 0; i < len(buf); i++ {
				buf[i] = 0
			} // clear chunk data
		} else {
			if _, err := devFile.Seek(e.OriginBlock*blockSize, os.SEEK_SET); err != nil {
				return err
			}

			if _, err := io.ReadFull(devFile, buf); err != nil {
				return err
			}
		}
		//		if err := s.putBlock(e.OriginBlock, blockSize, buf); err != nil {
		if err := s.putBlock(e.OriginBlock, blockSize, buf); err != nil {
			return err
		}
	}
	SHA1Code := fmt.Sprintf("%x\n", s.h.Sum(nil))
	// stdErr := os.Stderr
	// stdErr.Write([]byte(SHA1Code))
	fmt.Fprint(os.Stderr, "SHA1: "+SHA1Code)
	return nil
}

//func (s *streamSender) putHeader(header *streamHeader) error {
func (s *streamSender) putHeader(header []byte) error {

	headBuf, err := yaml.Marshal(s.header)
	//customHead, err := yaml.Marshal(header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal head failed.\n")
		return err
	}
	s.w.Write([]byte(C_HEAD))
	s.w.Write(headBuf)
	s.w.Write(header)
	s.w.Write([]byte{0x0a})
	return nil
}

func (s *streamSender) putBlock(index int64, blockSize int64, buf []byte) error {

	subHead := []byte(fmt.Sprintf("W %X %X\n", index*(blockSize>>9), blockSize>>9))
	if _, err := s.w.Write(subHead); err != nil {
		return err
	}
	//buffer := make([]byte, blockSize)
	if _, err := s.w.Write(buf); err != nil {
		return err
	}
	//f.Write([]byte{0x0a})
	s.w.Write([]byte{0x0a})
	s.h.Write(buf)
	return nil
}

func (s *streamSender) putBaseBlocks(blocks []thindelta.BlockHash) error {

	for _, block := range blocks {
		subHead := []byte(fmt.Sprintf("D %X %X %s %s\n", block.Offset, block.Length, block.HashType, block.Value))
		if _, err := s.w.Write(subHead); err != nil {
			return err
		}
	}
	if len(blocks) > 0 {
		s.w.Write([]byte{0x0a})
	}

	return nil
}
