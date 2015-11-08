package main

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"hash"
	"io"
	"os"

	"github.com/yangjian/lvbackup/lvmutil"
	"github.com/yangjian/lvbackup/thindump"
	"github.com/yangjian/lvbackup/vgcfg"

	"github.com/ncw/directio"
)

type streamSender struct {
	vgname  string
	lvname  string
	srcname string

	header streamHeader
	blocks []thindump.DeltaEntry

	w io.Writer
	h hash.Hash
}

func newStreamSender(vgname, lvname, srcname string, w io.Writer) (*streamSender, error) {
	return &streamSender{
		vgname:  vgname,
		lvname:  lvname,
		srcname: srcname,
		w:       w,
		h:       md5.New(),
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

	// dump block mapping
	tpoolDev := lvmutil.TPoolDevicePath(s.vgname, pool.Name)
	tmetaDev := lvmutil.LvDevicePath(s.vgname, pool.MetaName)
	superblk, err := thindump.Dump(tpoolDev, tmetaDev)
	if err != nil {
		return err
	}

	var dev, srcdev *thindump.Device

	dev, ok = superblk.FindDevice(lv.DeviceId)
	if !ok {
		return errors.New("super block do not have device " + string(lv.DeviceId))
	}

	if srclv != nil {
		srcdev, ok = superblk.FindDevice(srclv.DeviceId)
		if !ok {
			return errors.New("super block do not have device " + string(srclv.DeviceId))
		}
	}

	// list all blocks for full backup, or find changed blocks by comparing block
	// mappings for incremental backup.
	deltas, err := thindump.CompareDeviceBlocks(srcdev, dev)
	if err != nil {
		return err
	}

	s.blocks = deltas
	s.header.SchemeVersion = StreamSchemeV1
	s.header.StreamType = StreamTypeFull
	s.header.VolumeSize = uint64(lv.ExtentCount) * uint64(root.ExtentSize())
	s.header.BlockSize = uint32(pool.ChunkSize)
	s.header.BlockCount = uint64(len(s.blocks))

	copy(s.header.VolumeUUID[:], []byte(lv.UUID))

	if srclv != nil {
		s.header.StreamType = StreamTypeDelta
		copy(s.header.DeltaSourceUUID[:], []byte(srclv.UUID))
	}

	return nil
}

func (s *streamSender) Run() error {
	if err := s.prepare(); err != nil {
		return err
	}

	if err := s.putHeader(&s.header); err != nil {
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

	devpath := lvmutil.LvDevicePath(s.vgname, s.lvname)
	devFile, err := directio.OpenFile(devpath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer devFile.Close()

	buf := directio.AlignedBlock(int(s.header.BlockSize))

	blockSize := int64(s.header.BlockSize)
	for _, e := range s.blocks {
		if e.OpType == thindump.DeltaOpDelete {
			for i := 0; i < len(buf); i++ {
				buf[i] = 0
			}
		} else {
			if _, err := devFile.Seek(e.OriginBlock*blockSize, os.SEEK_SET); err != nil {
				return err
			}

			if _, err := io.ReadFull(devFile, buf); err != nil {
				return err
			}
		}
		if err := s.putBlock(e.OriginBlock, buf); err != nil {
			return err
		}
	}

	return nil
}

func (s *streamSender) putHeader(header *streamHeader) error {
	data, err := header.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = s.w.Write(data)
	return err
}

// Wire Format of LVM Block in the stream:
//   8 bytes: index; big endian
//   N bytes: data
//   M bytes: check sum of (index + data); currently always use MD5
func (s *streamSender) putBlock(index int64, buf []byte) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(index))

	if _, err := s.w.Write(b[:]); err != nil {
		return err
	}

	if _, err := s.w.Write(buf); err != nil {
		return err
	}

	s.h.Reset()
	s.h.Write(b[:])
	s.h.Write(buf)

	if _, err := s.w.Write(s.h.Sum(nil)); err != nil {
		return err
	}

	return nil
}
