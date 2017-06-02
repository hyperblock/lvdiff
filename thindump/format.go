package thindump

import (
	"encoding/xml"
	"errors"
	"sort"
)

type SingleMapping struct {
	XMLName     xml.Name `xml:"single_mapping"`
	OriginBlock int64    `xml:"origin_block,attr"`
	DataBlock   int64    `xml:"data_block,attr"`
	Time        int64    `xml:"time,attr"`
}

type RangeMapping struct {
	XMLName     xml.Name `xml:"range_mapping"`
	OriginBegin int64    `xml:"origin_begin,attr"`
	DataBegin   int64    `xml:"data_begin,attr"`
	Length      int64    `xml:"length,attr"`
	Time        int64    `xml:"time,attr"`
}

type singleMappingsByOrigin []SingleMapping

func (a singleMappingsByOrigin) Len() int           { return len(a) }
func (a singleMappingsByOrigin) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a singleMappingsByOrigin) Less(i, j int) bool { return a[i].OriginBlock < a[j].OriginBlock }

type entryByOrigin []DeltaEntry

func (a entryByOrigin) Len() int           { return len(a) }
func (a entryByOrigin) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a entryByOrigin) Less(i, j int) bool { return a[i].OriginBlock < a[j].OriginBlock }

type Device struct {
	XMLName      xml.Name `xml:"device"`
	DevId        int64    `xml:"dev_id,attr"`
	MappedBlocks int64    `xml:"mapped_blocks,attr"`
	Transaction  int64    `xml:"transaction,attr"`
	CreateTime   int64    `xml:"creation_time,attr"`
	SnapTime     int64    `xml:"snap_time,attr"`

	SingleMappings []SingleMapping `xml:"single_mapping"`
	RangeMapping   []RangeMapping  `xml:"range_mapping"`
}

type DifferentMapping struct {
	XMLName xml.Name `xml:"different"`
	Begin   int64    `xml:"begin,attr"`
	Length  int64    `xml:"length,attr"`
}

type LeftOnlyMapping struct {
	XMLName xml.Name `xml:"left_only"`
	Begin   int64    `xml:"begin,attr"`
	Length  int64    `xml:"length,attr"`
}

type RightOnlyMapping struct {
	XMLName xml.Name `xml:"right_only"`
	Begin   int64    `xml:"begin,attr"`
	Length  int64    `xml:"length,attr"`
}

type DiffBlocks struct {
	XMLName           xml.Name           `xml:"diff"`
	DifferentMappings []DifferentMapping `xml:"different"`
	LeftOnlyMappings  []LeftOnlyMapping  `xml:"left_only"`
	RightOnlyMappings []RightOnlyMapping `xml:"right_only"`
}

func (d *Device) ExpandRangeMappings() error {
	extras := make([]SingleMapping, 0, 256)

	for _, m := range d.RangeMapping {
		for i := int64(0); i < m.Length; i++ {
			extras = append(extras, SingleMapping{
				OriginBlock: m.OriginBegin + i,
				DataBlock:   m.DataBegin + i,
				Time:        m.Time,
			})
		}
	}

	if int64(len(d.SingleMappings)+len(extras)) != d.MappedBlocks {
		return errors.New("mapped blocks count mismatch")
	}

	d.SingleMappings = append(d.SingleMappings, extras...)
	d.RangeMapping = make([]RangeMapping, 0, 0)

	sort.Sort(singleMappingsByOrigin(d.SingleMappings))
	return nil
}

type SuperBlock struct {
	XMLName          xml.Name  `xml:"superblock"`
	Time             int64     `xml:"time,attr"`
	Transaction      int64     `xml:"transaction,attr"`
	DataBlockSize    int64     `xml:"data_block_size,attr"`
	NumberDataBlocks int64     `xml:"nr_data_blocks,attr"`
	Devices          []*Device `xml:"device"`
}

type DeltaSuperBlock struct {
	XMLName          xml.Name `xml:"superblock"`
	Time             int64    `xml:"time,attr"`
	Transaction      int64    `xml:"transaction,attr"`
	DataBlockSize    int64    `xml:"data_block_size,attr"`
	NumberDataBlocks int64    `xml:"nr_data_blocks,attr"`
	//	Devices          []*Device `xml:"device"`
	DiffResult DiffBlocks `xml:"diff"`
}

func (s SuperBlock) FindDevice(devid int64) (*Device, bool) {
	for _, dev := range s.Devices {
		if dev.DevId == devid {
			return dev, true
		}
	}
	return nil, false
}

type DeltaOpType uint8

const (
	DeltaOpCreate DeltaOpType = 1
	DeltaOpUpdate DeltaOpType = 2
	DeltaOpDelete DeltaOpType = 4
)

type DeltaEntry struct {
	OriginBlock int64       `xml:"origin_block,attr"`
	DataBlock   int64       `xml:"data_block,attr"`
	OpType      DeltaOpType `xml:"op,attr"`
}

type DeltaEntriesByDataBlock []DeltaEntry

func (a DeltaEntriesByDataBlock) Len() int           { return len(a) }
func (a DeltaEntriesByDataBlock) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DeltaEntriesByDataBlock) Less(i, j int) bool { return a[i].DataBlock < a[j].DataBlock }
