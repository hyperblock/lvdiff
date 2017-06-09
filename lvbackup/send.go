package lvbackup

import (
	"crypto/md5"
	"errors"
	"hash"
	"io"
	"os"

	yaml "gopkg.in/yaml.v2"

	"lvbackup/lvmutil"
	"lvbackup/thindelta"
	"lvbackup/vgcfg"

	"fmt"

	"github.com/ncw/directio"
)

type streamSender struct {
	vgname  string
	lvname  string
	srcname string

	header streamHeader
	blocks []thindelta.DeltaEntry

	w io.Writer
	h hash.Hash
}

func NewStreamSender(vgname, lvname, srcname string, w io.Writer) (*streamSender, error) {
	return &streamSender{
		vgname:  vgname,
		lvname:  lvname,
		srcname: srcname,
		w:       w,
		h:       md5.New(),
	}, nil
}

func (s *streamSender) prepare() error {

	//fmt.Println("prepare")
	root, err := vgcfg.Dump(s.vgname)
	//fmt.Println("dump finish.")
	if err != nil {
		//	fmt.Printf("%v\n", err)
		return err
	}

	var lv, srclv *vgcfg.ThinLvInfo
	var ok bool

	lv, ok = root.FindThinLv(s.lvname)

	if !ok {
		return errors.New("can not find thin lv " + s.lvname)
	}
	// fmt.Println(lv.Tags)
	// fmt.Printf("name: [%s, %s]\n", s.srcname, s.lvname)
	//	return errors.New("can not find thin lv " + s.lvname)

	//	return nil
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
	//fmt.Println(tpoolDev, tmetaDev)
	//fmt.Println(lv.DeviceId, srclv.DeviceId)
	deltaResult, err := thindelta.Delta(tpoolDev, tmetaDev, lv.DeviceId, srclv.DeviceId)
	if err != nil {
		fmt.Printf("thindump.Dump: %v\n", err)
		return err
	}
	//fmt.Println(deltaResult)

	// var dev, srcdev *thindump.Device

	// dev, ok = superblk.FindDevice(lv.DeviceId)
	// if !ok {
	// 	return errors.New("super block do not have device " + string(lv.DeviceId))
	// }

	// if srclv != nil {
	// 	srcdev, ok = superblk.FindDevice(srclv.DeviceId)
	// 	if !ok {
	// 		return errors.New("super block do not have device " + string(srclv.DeviceId))
	// 	}
	// }

	// list all blocks for full backup, or find changed blocks by comparing block
	// mappings for incremental backup.
	//deltas, err := thindump.CompareDeviceBlocks(srcdev, dev)
	deltas := thindelta.ExpandBlocks(deltaResult)

	if err != nil {
		return err
	}
	//fmt.Println(deltas)
	//return errors.New("-.- ")
	s.blocks = deltas
	//	s.header.SchemeVersion = StreamSchemeV1
	//	s.header.StreamType = StreamTypeFull
	s.header.Name = lv.Name
	s.header.VolumeSize = uint64(lv.ExtentCount) * uint64(root.ExtentSize())
	s.header.BlockSize = uint32(pool.ChunkSize)
	s.header.VolumeUUID = lv.UUID
	s.header.BlockCount = uint64(len(s.blocks))

	//	copy(s.header.VolumeUUID[:], []byte(lv.UUID))

	if srclv != nil {
		//s.header.StreamType = StreamTypeDelta
		//	copy(s.header.DeltaSourceUUID[:], []byte(srclv.UUID))
		s.header.DeltaSourceUUID = srclv.UUID
	}

	return nil
}

func (s *streamSender) Run(header string) error {

	if err := s.prepare(); err != nil {
		fmt.Println(err.Error())
		return err
	}

	if err := s.putHeader(header); err != nil {
		fmt.Println(err.Error())
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

	blockSize := int64(s.header.BlockSize) //chunk size: 64KB,65536
	//fmt.Println(blockSize)
	//return nil
	for _, e := range s.blocks {
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
func (s *streamSender) putHeader(header string) error {

	// h, err := os.Open(header)
	// if err != nil {
	// 	return err
	// }
	// defer h.Close()
	// buff, err := ioutil.ReadAll(h)
	// //_, err = h.Read(buff)
	// if err != nil {
	// 	return err
	// }
	// f.Write(buff)
	headBuf, err := yaml.Marshal(s.header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal head failed.\n")
		return err
	}
	s.w.Write([]byte(C_HEAD))
	s.w.Write(headBuf)
	s.w.Write([]byte{0x0a})
	return nil
}

// Wire Format of LVM Block in the stream:
//   8 bytes: index; big endian
//   N bytes: data
//   M bytes: check sum of (index + data); currently always use MD5
func (s *streamSender) putBlock(index int64, blockSize int64, buf []byte) error {

	subHead := []byte(fmt.Sprintf("%X %X\n", index*(blockSize>>9), blockSize>>9))
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
	// s.h.Reset()
	// s.h.Write(b[:])
	// s.h.Write(buf)

	// if _, err := s.w.Write(s.h.Sum(nil)); err != nil {
	// 	return err
	// }

	return nil
}
