package thindump

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func Parse(r io.Reader) (*SuperBlock, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	v := SuperBlock{}
	if err := xml.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

func sendThinPoolMessage(tpoolDev, message string) error {
	path, err := exec.LookPath("dmsetup")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "message", tpoolDev, "0", message)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func Dump(tpoolDev, tmetaDev string) (*SuperBlock, error) {
	// just try to release the metadata snap in case of error; ignore the result
	sendThinPoolMessage(tpoolDev, "release_metadata_snap")

	if err := sendThinPoolMessage(tpoolDev, "reserve_metadata_snap"); err != nil {
		return nil, errors.New("can not send reserve_metadata_snap to tpool: " + err.Error())
	}
	defer sendThinPoolMessage(tpoolDev, "release_metadata_snap")

	path, err := exec.LookPath("thin_dump")
	if err != nil {
		return nil, err
	}

	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, err
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("thindump_%s.xml", hex.EncodeToString(buf[:])))

	cmd := exec.Command(path, "-o", tmp, tmetaDev)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	defer os.Remove(tmp)

	f, err := os.Open(tmp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

func CompareDeviceBlocks(src, dst *Device) ([]DeltaEntry, error) {
	entries := make([]DeltaEntry, 0, 256)

	if err := dst.ExpandRangeMappings(); err != nil {
		return nil, err
	}

	if src == nil {
		for _, m := range dst.SingleMappings {
			entries = append(entries, DeltaEntry{
				OriginBlock: m.OriginBlock,
				OpType:      DeltaOpCreate,
			})
		}

		return entries, nil
	}

	if err := src.ExpandRangeMappings(); err != nil {
		return nil, err
	}

	i := 0
	j := 0

	for {
		if i >= len(src.SingleMappings) || j >= len(dst.SingleMappings) {
			break
		}

		s := src.SingleMappings[i]
		d := dst.SingleMappings[j]

		switch {
		case s.OriginBlock < d.OriginBlock:
			entries = append(entries, DeltaEntry{
				OriginBlock: s.OriginBlock,
				OpType:      DeltaOpDelete,
			})
			i++
		case s.OriginBlock == d.OriginBlock:
			if s.DataBlock != d.DataBlock {
				entries = append(entries, DeltaEntry{
					OriginBlock: d.OriginBlock,
					OpType:      DeltaOpUpdate,
				})
			}
			i++
			j++
		case s.OriginBlock > d.OriginBlock:
			entries = append(entries, DeltaEntry{
				OriginBlock: d.OriginBlock,
				OpType:      DeltaOpCreate,
			})
			j++
		}
	}

	for _, m := range src.SingleMappings[i:] {
		entries = append(entries, DeltaEntry{
			OriginBlock: m.OriginBlock,
			OpType:      DeltaOpDelete,
		})
	}

	for _, m := range dst.SingleMappings[j:] {
		entries = append(entries, DeltaEntry{
			OriginBlock: m.OriginBlock,
			OpType:      DeltaOpCreate,
		})
	}

	return entries, nil
}
