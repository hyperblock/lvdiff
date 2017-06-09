package thindelta

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"sort"
)

//func Parse(r io.Reader) (*SuperBlock, error) {
func Parse(r io.Reader) (*DiffBlocks, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	//v := SuperBlock{}
	v := DiffBlocks{}
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

func Delta(tpoolDev, tmetaDev string, layer_id, parent_id int64) (*DiffBlocks, error) {
	// just try to release the metadata snap in case of error; ignore the result
	sendThinPoolMessage(tpoolDev, "release_metadata_snap")

	if err := sendThinPoolMessage(tpoolDev, "reserve_metadata_snap"); err != nil {
		return nil, errors.New("can not send reserve_metadata_snap to tpool: " + err.Error())
	}
	defer sendThinPoolMessage(tpoolDev, "release_metadata_snap")

	path, err := exec.LookPath("thin_delta")
	if err != nil {
		return nil, err
	}

	// var buf [8]byte
	// if _, err := rand.Read(buf[:]); err != nil {
	// 	return nil, err
	// }

	// tmp := filepath.Join(os.TempDir(), fmt.Sprintf("thindelta_%s.xml", hex.EncodeToString(buf[:])))
	snap1 := fmt.Sprintf("%d", layer_id)
	snap2 := fmt.Sprintf("%d", parent_id)
	cmd := exec.Command(path, "-m", "--snap1", snap1, "--snap2", snap2, tmetaDev)
	// if err := cmd.Run(); err != nil {
	// 	return nil, err
	// }
	result, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	v := DeltaSuperBlock{}
	if err = xml.Unmarshal(result, &v); err != nil {
		return nil, err
	}
	return &v.DiffResult, nil
}

func ExpandBlocks(deltablocks *DiffBlocks) []DeltaEntry {

	entries := []DeltaEntry{}
	for _, m := range deltablocks.DifferentMappings {
		for i := int64(0); i < m.Length; i++ {
			entries = append(entries, DeltaEntry{
				OriginBlock: m.Begin + i,
				OpType:      DeltaOpUpdate,
			})
		}
	}
	for _, m := range deltablocks.LeftOnlyMappings {
		for i := int64(0); i < m.Length; i++ {
			entries = append(entries, DeltaEntry{
				OriginBlock: m.Begin + i,
				OpType:      DeltaOpCreate,
			})
		}
	}
	for _, m := range deltablocks.RightOnlyMappings {
		for i := int64(0); i < m.Length; i++ {
			entries = append(entries, DeltaEntry{
				OriginBlock: m.Begin + i,
				OpType:      DeltaOpDelete,
			})
		}
	}
	sort.Sort(entryByOrigin(entries))
	return entries
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
