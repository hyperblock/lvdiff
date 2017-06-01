package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"

	"hyperblock/lvbackup/lvmutil"
	"hyperblock/lvbackup/vgcfg"

	"github.com/ncw/directio"

	"strings"

	"bufio"
)

type streamRecver struct {
	vgname   string
	poolname string
	lvname   string

	header   streamHeader
	prevUUID string

	r io.Reader
	h hash.Hash
}

func newStreamRecver(vgname, poolname, lvname string, r io.Reader) (*streamRecver, error) {
	return &streamRecver{
		vgname:   vgname,
		poolname: poolname,
		lvname:   lvname,
		r:        r,
		h:        md5.New(),
	}, nil
}

func (sr *streamRecver) prepare() error {
	var buf [SchemeV1HeaderLength]byte
	if _, err := io.ReadFull(sr.r, buf[:]); err != nil {
		return err
	}

	if err := sr.header.UnmarshalBinary(buf[:]); err != nil {
		return err
	}

	// check whether block size of pool match with the stream
	root, err := vgcfg.Dump(sr.vgname)
	if err != nil {
		return err
	}

	pool, ok := root.FindThinPool(sr.poolname)
	if !ok {
		return errors.New("can not find thin pool " + sr.poolname)
	}

	if pool.ChunkSize != int64(sr.header.BlockSize) {
		return errors.New("block size does not match with that of local pool")
	}

	// create new lv if needed
	if sr.header.StreamType == StreamTypeFull {
		if err := lvmutil.CreateThinLv(sr.vgname, sr.poolname, sr.lvname, int64(sr.header.VolumeSize)); err != nil {
			return errors.New("can not create thin lv: " + err.Error())
		}
	}

	if sr.header.StreamType == StreamTypeDelta {
		expected := string(sr.header.DeltaSourceUUID[:])
		if len(sr.prevUUID) > 0 && sr.prevUUID != expected {
			return fmt.Errorf("incremental backup chain is broken; expects %s but prev is %s ", expected, sr.prevUUID)
		}

		// resize the volume if needed
		lv, ok := root.FindThinLv(sr.lvname)
		if !ok {
			return errors.New("can not find thin lv " + sr.lvname)
		}

		size := int64(sr.header.VolumeSize)
		if lv.ExtentCount*root.ExtentSize() != size {
			if err := lvmutil.ResizeLv(sr.vgname, sr.lvname, size); err != nil {
				return errors.New("can not resize thin lv: " + err.Error())
			}
		}
	}

	return nil
}

func (sr *streamRecver) recvNextStream() error {
	// if err := sr.prepare(); err != nil {
	// 	return err
	// }

	devpath := lvmutil.LvDevicePath(sr.vgname, sr.lvname)
	fmt.Printf("open devpath: %s\n", devpath)
	devFile, err := directio.OpenFile(devpath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	//defer devFile.Close()
	fmt.Println("open dev done.")

	bfRd := bufio.NewReader(sr.r)
	//	subHead := make([]byte, 18)
	for {
		//	if _, err := io.ReadFull(sr.r, subHead); err != nil {
		line, err := bfRd.ReadBytes('\n')
		//fmt.Println(string(subHead))
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		subHead := string(line[:len(line)-1])

		args := strings.Split(subHead, " ")
		if len(args) != 2 {
			break
		}
		offset, _ := strconv.ParseInt(args[0], 16, 64)
		length, _ := strconv.ParseInt(args[1], 16, 64)
		length <<= 9
		offset <<= 9
		fmt.Println(offset, length)
		//		sr.h.Reset()
		//		sr.h.Write(b[:n])
		tmpbuf := make([]byte, length)
		buf := make([]byte, length)
		var cnt int64
		for {
			n, err := bfRd.Read(tmpbuf)
			if err != nil {
				return err
			}
			copy(buf[cnt:], tmpbuf)
			cnt += int64(n)
			if cnt >= length {
				break
			}
			tmpbuf = make([]byte, length-cnt)
		}
		//	newline := make([]byte, 1)
		//io.ReadFull(sr.r, newline)
		// if !bytes.Equal(sr.h.Sum(nil), b[n:n+md5.Size]) {
		// 	return fmt.Errorf("check sum mismatch for %dst block", i)
		// }

		// index := int64(binary.BigEndian.Uint64(b))
		// copy(buf, b[8:n])

		//	if _, err := devFile.Seek(int64(sr.header.BlockSize)*index, os.SEEK_SET); err != nil {
		if _, err := devFile.Seek(offset, os.SEEK_SET); err != nil {
			fmt.Println("seek error.")
			return err
		}

		if _, err := devFile.Write(buf); err != nil {
			fmt.Println("dev write error.")
			return err
		}
		fmt.Println("done")
		//	bfRd.ReadLine()
		bfRd.ReadBytes('\n')
	}

	//	sr.prevUUID = string(sr.header.VolumeUUID[:])
	return nil
}

func (sr *streamRecver) Run() error {
	var err error
	//bfRd := bufio.NewReader(sr.r)
	//for {
	err = sr.recvNextStream()
	if err != nil {
		//break
		return err
	}
	//}

	if err == io.EOF {
		err = nil
	}
	return err
}
