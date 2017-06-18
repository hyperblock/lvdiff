package thindelta

import (
	"io"

	"github.com/ncw/directio"

	"fmt"
	"hash/crc32"
	"os"
)

type BlockHash struct {
	Offset, Length int64
	HashType       string
	Value          string
}

func CheckBase(devpath string, blocksize int64, baseBlocks []BlockHash) (bool, error) {

	if len(baseBlocks) == 0 {
		return true, nil
	}
	devFile, err := directio.OpenFile(devpath, os.O_RDONLY, 0644)
	if err != nil {
		return false, err
	}
	defer devFile.Close()
	buf := directio.AlignedBlock(int(blocksize))

	for _, block := range baseBlocks {
		addr := block.Offset / (blocksize >> 9)
		length := block.Length / (blocksize >> 9)
		//	fmt.Fprintln(os.Stderr, addr, length)
		hash := crc32.NewIEEE()
		for offset := int64(0); offset < length; offset++ {
			if _, err := devFile.Seek((addr+offset)*blocksize, os.SEEK_SET); err != nil {
				return false, err
			}
			if _, err := io.ReadFull(devFile, buf); err != nil {
				return false, err
			}
			hash.Write(buf)
		}
		value := fmt.Sprintf("%x", hash.Sum32())
		if value != block.Value {
			return false, nil
		}

	}
	return true, nil
}

func GenChecksum(devpath string, blocksize int64, blocks []DeltaEntry, level int) ([]BlockHash, error) {

	if level == 0 {
		return nil, nil
	}
	devFile, err := directio.OpenFile(devpath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()
	buf := directio.AlignedBlock(int(blocksize))

	//blockId := ""
	ret := []BlockHash{}
	p := 0

	for {
		//fmt.Fprintln(os.Stderr, p)
		if p >= len(blocks) {
			break
		}
		e := blocks[p]

		if e.OriginBlock == 0 || e.OriginBlock-blocks[p-1].OriginBlock > 1 {
			q := p
			hash := crc32.NewIEEE()
			for {
				addr := blocks[q].OriginBlock
				if _, err := devFile.Seek(addr*blocksize, os.SEEK_SET); err != nil {
					return nil, err
				}
				if _, err := io.ReadFull(devFile, buf); err != nil {
					return nil, err
				}
				hash.Write(buf)
				q++
				if q >= len(blocks) || blocks[q].OriginBlock-e.OriginBlock > 1 {
					break
				}
				if level != 3 {
					break
				}

			}
			ret = append(ret, BlockHash{
				Offset:   e.OriginBlock * blocksize >> 9,
				Length:   int64(q-p) * blocksize >> 9,
				HashType: "CRC32",
				Value:    fmt.Sprintf("%x", hash.Sum32()),
			})
			if level == 1 {
				break
			}
			p = q
		} else {
			p++
		}
	}

	return ret, nil
}
