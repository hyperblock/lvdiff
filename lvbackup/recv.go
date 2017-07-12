package lvbackup

import (
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"

	"github.com/hyperblock/lvdiff/lvbackup/lvmutil"
	"github.com/hyperblock/lvdiff/lvbackup/thindelta"
	"github.com/hyperblock/lvdiff/lvbackup/vgcfg"

	"github.com/ncw/directio"
	yaml "gopkg.in/yaml.v2"

	"strings"

	"bufio"
)

type streamRecver struct {
	vgname       string
	poolname     string
	lvname       string
	disableCheck bool

	header   streamHeader
	prevUUID string

	baseBlocks []thindelta.BlockHash

	r io.Reader
	h hash.Hash
}

func NewStreamRecver(vgname, poolname, lvname string, flg bool, r io.Reader) (*streamRecver, error) {
	return &streamRecver{
		vgname:       vgname,
		poolname:     poolname,
		lvname:       lvname,
		disableCheck: flg,
		r:            r,
		h:            md5.New(),
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

	ret := fmt.Sprintf("%s %d%% (%d/%d)", bar, base, current, total)
	return ret
}

func (sr *streamRecver) prepare() error {

	// check whether block size of pool match with the stream

	root, err := vgcfg.Dump(sr.vgname)
	if err != nil {
		return err
	}
	baseLv, ok := root.FindThinLv(sr.lvname)
	if !ok {
		return errors.New("can not find thin lv " + sr.lvname)
	}
	pool, ok := root.FindThinPool(baseLv.Pool)
	if !ok {
		return errors.New("can not find thin pool " + baseLv.Pool)
	}

	if pool.ChunkSize != int64(sr.header.BlockSize) {
		return errors.New("block size does not match with that of local pool")
	}

	if sr.disableCheck == false {

		devPath := lvmutil.LvDevicePath(sr.vgname, sr.lvname)
		ok, err := thindelta.CheckBase(devPath, pool.ChunkSize, sr.baseBlocks)
		if err != nil {
			return fmt.Errorf("Get volume checksum (%s/%s) error. %s", sr.vgname, sr.lvname, err.Error())
		}
		//	fmt.Println(lvChecksum)

		if !ok {
			return fmt.Errorf("checksum incorrect.")
		}
	}

	//create a snapshot
	fmt.Printf("Create Snapshot volume. (%s)\n", sr.header.Name)
	if err := lvmutil.CreateSnapshotLv(sr.vgname, sr.lvname, sr.header.Name); err != nil {
		return errors.New("can not create snapshotLv: " + sr.header.Name + " " + err.Error())
	}
	sr.lvname = sr.header.Name

	return nil
}

func (sr *streamRecver) readHeader(bfRd *bufio.Reader) error {

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
		if index > 0 {
			headBuff = append(headBuff, pair[0:]...)
		}
		fmt.Fprintf(os.Stderr, "%s", string(pair))
		index++
	}

	err := yaml.Unmarshal(headBuff, &sr.header)
	if err != nil {
		return err
	}
	return nil

}

func (sr *streamRecver) readBaseBlocks(bfRd *bufio.Reader) error {

	if sr.header.DetectLevel == 0 {
		return nil
	}
	for {
		buf, err := bfRd.ReadBytes('\n')
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		if buf[0] == 0xa {
			break
		}
		tokens := strings.Split(string(buf), " ")
		offset, err := strconv.ParseInt(tokens[1], 16, 64)
		if err != nil {
			return err
		}
		length, err := strconv.ParseInt(tokens[2], 16, 64)
		if err != nil {
			return err
		}
		baseBlock := thindelta.BlockHash{
			Offset:   offset,
			Length:   length,
			HashType: tokens[3],
			Value:    tokens[4][:len(tokens[4])-1],
		}
		sr.baseBlocks = append(sr.baseBlocks, baseBlock)
	}
	//	fmt.Println(sr.baseBlocks)
	return nil
}

func (sr *streamRecver) recvDiffStream() error {

	bfRd := bufio.NewReader(sr.r)
	sr.readHeader(bfRd)
	sr.readBaseBlocks(bfRd)

	if err := sr.prepare(); err != nil {
		return err
	}

	err := lvmutil.ActivateLv(sr.vgname, sr.lvname)
	if err != nil {
		return err
	}
	//return nil
	defer lvmutil.DeactivateLv(sr.vgname, sr.lvname)

	devpath := lvmutil.LvDevicePath(sr.vgname, sr.lvname)

	devFile, err := directio.OpenFile(devpath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer devFile.Close()

	total := int64(sr.header.BlockCount)
	dwWritten := int64(0)
	//	fmt.Println(total)
	for {

		line, err := bfRd.ReadBytes('\n')
		//fmt.Println(string(subHead))
		if err != nil {

			return err
		}
		subHead := string(line[:len(line)-1])

		args := strings.Split(subHead, " ")
		if len(args) != 3 {
			break
		}
		offset, _ := strconv.ParseInt(args[1], 16, 64)
		length, _ := strconv.ParseInt(args[2], 16, 64)
		length <<= 9
		offset <<= 9

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

		if _, err := devFile.Seek(offset, os.SEEK_SET); err != nil {
			fmt.Println("seek error.")
			return err
		}
		if _, err := devFile.Write(buf); err != nil {
			fmt.Println("dev write error.")
			return err
		}
		dwWritten++
		//fmt.Println(dwWritten)
		bfRd.ReadBytes('\n')
		bar := print_ProcessBar(dwWritten, total)
		fmt.Printf("\rPatch blocks %s", bar)
	}
	fmt.Println("\ndone")

	//	sr.prevUUID = string(sr.header.VolumeUUID[:])
	return nil
}

func (sr *streamRecver) Run() error {
	var err error
	//bfRd := bufio.NewReader(sr.r)
	//for {
	err = sr.recvDiffStream()

	if err == io.EOF {
		fmt.Println("\nDone.")
		err = nil
	}
	return err
}
