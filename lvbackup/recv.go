package lvbackup

import (
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"

	"lvbackup/lvmutil"
	"lvbackup/vgcfg"

	"github.com/ncw/directio"
	yaml "gopkg.in/yaml.v2"

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

func NewStreamRecver(vgname, poolname, lvname string, r io.Reader) (*streamRecver, error) {
	return &streamRecver{
		vgname:   vgname,
		poolname: poolname,
		lvname:   lvname,
		r:        r,
		h:        md5.New(),
	}, nil
}

func print_ProcessBar(current, total int64) string {

	bar := "["
	base := int((float32(current) / float32(total)) * 100)
	delta := int(float32(base)/float32(2.5) + 0.5)
	for i := 0; i < delta; i++ {
		bar += "="
	}
	delta = 40 - delta
	for i := 0; i < delta; i++ {
		bar += " "
	}
	bar += "]"
	// //	A, B := current>>20, total>>20
	// 	if A == 0 {
	// 		A = 1
	// 	}
	// 	if B == 0 {
	// 		B = 1
	// 	}
	ret := fmt.Sprintf("%s %d%% (%d/%d)", bar, base, current, total)
	return ret
}

func (sr *streamRecver) prepare() error {

	// var buf [SchemeV1HeaderLength]byte
	// if _, err := io.ReadFull(sr.r, buf[:]); err != nil {
	// 	return err
	// }

	// if err := sr.header.UnmarshalBinary(buf[:]); err != nil {
	// 	return err
	// }

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
	// if sr.header.StreamType == StreamTypeFull {
	// 	if err := lvmutil.CreateThinLv(sr.vgname, sr.poolname, sr.lvname, int64(sr.header.VolumeSize)); err != nil {
	// 		return errors.New("can not create thin lv: " + err.Error())
	// 	}
	// }

	//create a snapshot
	fmt.Printf("Create Snapshot volume. (%s)\n", sr.header.Name)
	if err := lvmutil.CreateSnapshotLv(sr.vgname, sr.lvname, sr.header.Name); err != nil {
		return errors.New("can not create snapshotLv: " + sr.header.Name + " " + err.Error())
	}
	sr.lvname = sr.header.Name

	// if sr.header.StreamType == StreamTypeDelta {
	// 	expected := string(sr.header.DeltaSourceUUID[:])
	// 	if len(sr.prevUUID) > 0 && sr.prevUUID != expected {
	// 		return fmt.Errorf("incremental backup chain is broken; expects %s but prev is %s ", expected, sr.prevUUID)
	// 	}

	// 	// resize the volume if needed
	// 	lv, ok := root.FindThinLv(sr.lvname)
	// 	if !ok {
	// 		return errors.New("can not find thin lv " + sr.lvname)
	// 	}

	// 	size := int64(sr.header.VolumeSize)
	// 	if lv.ExtentCount*root.ExtentSize() != size {
	// 		if err := lvmutil.ResizeLv(sr.vgname, sr.lvname, size); err != nil {
	// 			return errors.New("can not resize thin lv: " + err.Error())
	// 		}
	// 	}
	// }

	return nil
}

func (sr *streamRecver) readHeader(bfRd *bufio.Reader) error {

	//info := ""
	headBuff := []byte{}
	index := 0
	for {
		pair, err := bfRd.ReadBytes('\n')
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		if pair[0] == 0xa {
			break
		}
		//info += string(pair)
		if index > 0 {
			headBuff = append(headBuff, pair[0:]...)
		}
		index++
	}

	err := yaml.Unmarshal(headBuff, &sr.header)
	if err != nil {
		return err
	}
	return nil
	//fmt.Printf("%s", info)
}

func (sr *streamRecver) recvNextStream() error {

	bfRd := bufio.NewReader(sr.r)
	sr.readHeader(bfRd)
	if err := sr.prepare(); err != nil {
		return err
	}

	err := lvmutil.ActivateLv(sr.vgname, sr.lvname)
	if err != nil {
		return err
	}
	defer lvmutil.DeactivateLv(sr.vgname, sr.lvname)

	devpath := lvmutil.LvDevicePath(sr.vgname, sr.lvname)
	//	fmt.Printf("open devpath: %s\n", devpath)
	devFile, err := directio.OpenFile(devpath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer devFile.Close()
	//fmt.Println("open dev done.")
	//	fmt.Println(sr.header)
	total := int64(sr.header.BlockCount)
	dwWritten := int64(0)
	//	fmt.Println(total)
	for {
		bar := print_ProcessBar(dwWritten, total)
		fmt.Printf("\rPatch blocks %s", bar)

		//	fmt.Println("read data.")
		//	if _, err := io.ReadFull(sr.r, subHead); err != nil {
		line, err := bfRd.ReadBytes('\n')
		//fmt.Println(string(subHead))
		if err != nil {
			//	if err==err.e
			//fmt.Println(err.Error())
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
		//	fmt.Println(offset, length)
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
		dwWritten++
		//fmt.Println("bar?")
		//	fmt.Println("done")
		//	bfRd.ReadLine()
		bfRd.ReadBytes('\n')
	}
	fmt.Println("\ndone")

	//	sr.prevUUID = string(sr.header.VolumeUUID[:])
	return nil
}

func (sr *streamRecver) Run() error {
	var err error
	//bfRd := bufio.NewReader(sr.r)
	//for {
	err = sr.recvNextStream()
	// if err != nil {
	// 	//break
	// 	return err
	// }
	// //}

	if err == io.EOF {
		fmt.Println("\nDone.")
		err = nil
	}
	return err
}
