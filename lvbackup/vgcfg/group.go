package vgcfg

import (
	"fmt"
	"sort"
)

type ThinPoolInfo struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	MetaName      string `json:"tmeta_name"`
	DataName      string `json:"tdata_name"`
	StartExtent   int64  `json:"start_extent"`
	ExtentCount   int64  `json:"extent_count" `
	ChunkSize     int64  `json:"chunk_size"`
	ZeroNewBlocks bool   `json:"zero_new_blocks"`
}

type ThinLvInfo struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Pool string `json:"pool"`
	//	Tags          []string `json:"tags"`
	Origin        string `json:"origin"`
	TransactionId int64  `json:"tx_id"`
	DeviceId      int64  `json:"dev_id"`
	StartExtent   int64  `json:"start_extent"`
	ExtentCount   int64  `json:"extent_count" `
}

func (t *ThinLvInfo) String() string {
	return fmt.Sprintf("[ThinLV: N=%s, P=%s O=%s, T=%d, D=%d]",
		t.Name, t.Pool, t.Origin, t.TransactionId, t.DeviceId)
}

type ThinLvByTxId []*ThinLvInfo

func (a ThinLvByTxId) Len() int           { return len(a) }
func (a ThinLvByTxId) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ThinLvByTxId) Less(i, j int) bool { return a[i].TransactionId < a[j].TransactionId }

type Group struct {
	name      string
	parent    *Group
	childs    map[string]*Group
	variables map[string]interface{}
}

func newGroup(name string, parent *Group) *Group {
	return &Group{
		name:      name,
		parent:    parent,
		childs:    make(map[string]*Group, 10),
		variables: make(map[string]interface{}, 64),
	}
}

func (g *Group) addVar(name string, val interface{}) {
	g.variables[name] = val
}

func (g *Group) addSubGroup(sub *Group) {
	g.childs[sub.Name()] = sub
}

func (g *Group) IsRoot() bool   { return g.parent == nil }
func (g *Group) Name() string   { return g.name }
func (g *Group) Parent() *Group { return g.parent }

func (g *Group) GetVar(key string) (interface{}, bool) {
	v, ok := g.variables[key]
	return v, ok
}

func (g *Group) VarStringValue(key string) (string, bool) {
	v, ok := g.variables[key]
	if !ok {
		return "", false
	}

	s, ok := v.(string)
	return s, ok
}

// func (g *Group) VarStringArrayValue(key string) ([]string, bool) {
// 	v, ok := g.variables[key]
// 	if !ok {
// 		return []string{}, false
// 	}

// 	s, ok := v.([]string)
// 	return s, ok
// }

func (g *Group) VarIntegerValue(key string) (int64, bool) {
	v, ok := g.variables[key]
	if !ok {
		return 0, false
	}

	s, ok := v.(int64)
	return s, ok
}

func (g *Group) SubGroup(name string) (*Group, bool) {
	v, ok := g.childs[name]
	return v, ok
}

func (g *Group) VgName() string {
	if !g.IsRoot() || len(g.childs) != 1 {
		return ""
	}

	for name, _ := range g.childs {
		return name
	}

	return ""
}

func (g *Group) ExtentSize() int64 {
	name := g.VgName()
	if len(name) == 0 {
		return 0
	}

	vg, _ := g.SubGroup(name)
	value, ok := vg.VarIntegerValue("extent_size")
	if !ok {
		return 0
	}

	return value * 512
}

func (g *Group) IsThinPool() bool {
	s1, ok := g.SubGroup("segment1")
	if !ok {
		return false
	}

	t, ok := s1.VarStringValue("type")
	return ok && t == "thin-pool"
}

func (g *Group) IsThinLv() bool {
	s1, ok := g.SubGroup("segment1")
	if !ok {
		return false
	}

	t, ok := s1.VarStringValue("type")
	return ok && t == "thin"
}

func (g *Group) ThinPoolInfo() *ThinPoolInfo {
	if !g.IsThinPool() {
		return nil
	}

	info := ThinPoolInfo{}

	if val, ok := g.VarStringValue("id"); !ok {
		return nil
	} else {
		info.UUID = val
	}

	s1, _ := g.SubGroup("segment1")

	if val, ok := s1.VarIntegerValue("start_extent"); !ok {
		return nil
	} else {
		info.StartExtent = val
	}

	if val, ok := s1.VarIntegerValue("extent_count"); !ok {
		return nil
	} else {
		info.ExtentCount = val
	}

	if val, ok := s1.VarStringValue("metadata"); !ok {
		return nil
	} else {
		info.MetaName = val
	}

	if val, ok := s1.VarStringValue("pool"); !ok {
		return nil
	} else {
		info.DataName = val
	}

	if val, ok := s1.VarIntegerValue("chunk_size"); !ok {
		return nil
	} else {
		info.ChunkSize = val * 512
	}

	if val, ok := s1.VarIntegerValue("zero_new_blocks"); ok {
		info.ZeroNewBlocks = val != 0
	}

	info.Name = g.Name()
	return &info
}

func (g *Group) ThinLvInfo() *ThinLvInfo {
	if !g.IsThinLv() {
		return nil
	}

	info := ThinLvInfo{}

	if val, ok := g.VarStringValue("id"); !ok {
		return nil
	} else {
		info.UUID = val
	}

	s1, _ := g.SubGroup("segment1")

	if val, ok := s1.VarIntegerValue("start_extent"); !ok {
		return nil
	} else {
		info.StartExtent = val
	}

	if val, ok := s1.VarIntegerValue("extent_count"); !ok {
		return nil
	} else {
		info.ExtentCount = val
	}

	if val, ok := s1.VarStringValue("thin_pool"); !ok {
		return nil
	} else {
		info.Pool = val
	}

	if val, ok := s1.VarIntegerValue("transaction_id"); !ok {
		return nil
	} else {
		info.TransactionId = val
	}

	if val, ok := s1.VarIntegerValue("device_id"); !ok {
		return nil
	} else {
		info.DeviceId = val
	}

	if val, ok := s1.VarStringValue("origin"); ok {
		info.Origin = val
	}

	// if val, ok := s1.VarStringArrayValue("tags"); ok {
	// 	info.Tags = val
	// }

	info.Name = g.Name()
	return &info
}

func (g *Group) LogicalVolumes() ([]*Group, bool) {
	if !g.IsRoot() {
		return nil, false
	}

	// ROOT > vg_name > "logical_volumes" > lv_name
	vgname := g.VgName()
	if len(vgname) == 0 {
		return nil, false
	}

	vg, ok := g.SubGroup(vgname)
	if !ok {
		return nil, false
	}

	lvsGroup, ok := vg.SubGroup("logical_volumes")
	if !ok {
		return nil, false
	}

	lvs := make([]*Group, 0, 32)
	for _, lv := range lvsGroup.childs {
		if lv.IsThinLv() || lv.IsThinPool() {
			lvs = append(lvs, lv)
		}
	}

	return lvs, true
}

func (g *Group) FindThinPool(poolname string) (*ThinPoolInfo, bool) {
	pools, ok := g.ListThinPools()
	if !ok {
		return nil, false
	}

	for _, pool := range pools {
		if pool.Name == poolname {
			return pool, true
		}
	}

	return nil, false
}

func (g *Group) FindThinLv(lvname string) (*ThinLvInfo, bool) {
	lvs, ok := g.ListThinLvInfo(func(lv *ThinLvInfo) bool {
		return lv.Name == lvname
	})

	if !ok || len(lvs) != 1 {
		return nil, false
	}

	return lvs[0], true
}

func (g *Group) ListThinPools() ([]*ThinPoolInfo, bool) {
	lvs, ok := g.LogicalVolumes()
	if !ok {
		return nil, false
	}

	infos := make([]*ThinPoolInfo, 0, len(lvs))
	for _, lv := range lvs {
		if !lv.IsThinPool() {
			continue
		}

		info := lv.ThinPoolInfo()
		if info == nil {
			return nil, false
		}

		infos = append(infos, info)
	}

	return infos, true
}

func (g *Group) ListThinLvInfo(filter func(*ThinLvInfo) bool) ([]*ThinLvInfo, bool) {
	lvs, ok := g.LogicalVolumes()
	if !ok {
		return nil, false
	}

	infos := make([]*ThinLvInfo, 0, len(lvs))
	for _, lv := range lvs {
		if !lv.IsThinLv() {
			continue
		}

		info := lv.ThinLvInfo()
		if info == nil {
			return nil, false
		}

		if filter == nil || filter(info) == true {
			infos = append(infos, info)
		}
	}

	sort.Sort(ThinLvByTxId(infos))
	return infos, true
}

func (g *Group) String() string {
	return fmt.Sprintf("Group %s Variables %#v SubGroups %#v", g.Name, g.variables, g.childs)
}
