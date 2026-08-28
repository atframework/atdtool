package noncloudnative

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

type DeployUnit struct {
	Name            string `json:"chart_name"`
	WorldInstance   bool   `json:"world_instance"`
	InstanceCount   uint64 `json:"instance_count"`
	StartInstanceId uint64 `json:"start_instance_id"`
	Group           uint64 `json:"group"`
}

type DeployConf struct {
	WorldID  uint64        `json:"world_id"`
	ZoneId   uint64        `json:"zone_id"`
	Instance []*DeployUnit `json:"proc_desc"`
}

// ScriptProcInstance is one expanded instance of a deploy unit.
type ScriptProcInstance struct {
	InstanceID uint64
	BusAddr    string
}

// ScriptProc is a deploy unit with all instances expanded.
type ScriptProc struct {
	Name      string
	Group     uint64
	Instances []ScriptProcInstance
}

// ScriptProcGroup collects procs in the same start/stop batch.
type ScriptProcGroup struct {
	Group uint64
	Procs []ScriptProc
}

// ExpandInstances expands all instances of the unit into instance id and bus address pairs.
// The type id comes from the chart values and the bus address zone segment
// is 0 for world instances.
func (u *DeployUnit) ExpandInstances(worldID, zoneID, typeId uint64) []ScriptProcInstance {
	instances := make([]ScriptProcInstance, 0, u.InstanceCount)
	for i := uint64(0); i < u.InstanceCount; i++ {
		instanceID := u.StartInstanceId + i
		instances = append(instances, ScriptProcInstance{
			InstanceID: instanceID,
			BusAddr:    buildBusAddr(worldID, zoneID, typeId, instanceID, u.WorldInstance),
		})
	}
	return instances
}

func buildBusAddr(worldID, zoneID, typeId, instanceID uint64, worldInstance bool) string {
	if worldInstance {
		zoneID = 0
	}
	addrCom := []string{
		fmt.Sprint(worldID),
		fmt.Sprint(zoneID),
		fmt.Sprint(typeId),
		fmt.Sprint(instanceID),
	}
	return strings.Join(addrCom, ".")
}

// BuildScriptProcGroups groups expanded procs by batch group.
// The input procs must keep the deploy.yaml declaration order. Groups are
// sorted ascending and a missing group is treated as 0. Procs inside one
// group keep the deploy.yaml order and instances are sorted by instance id.
func BuildScriptProcGroups(procs []ScriptProc) []ScriptProcGroup {
	groups := make([]ScriptProcGroup, 0, len(procs))
	groupIndex := make(map[uint64]int, len(procs))
	for _, proc := range procs {
		instances := append([]ScriptProcInstance(nil), proc.Instances...)
		sort.Slice(instances, func(i, j int) bool {
			return instances[i].InstanceID < instances[j].InstanceID
		})
		proc.Instances = instances

		if index, ok := groupIndex[proc.Group]; ok {
			groups[index].Procs = append(groups[index].Procs, proc)
			continue
		}
		groupIndex[proc.Group] = len(groups)
		groups = append(groups, ScriptProcGroup{
			Group: proc.Group,
			Procs: []ScriptProc{proc},
		})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].Group < groups[j].Group
	})
	return groups
}

func loadDeployData(filename string) (interface{}, error) {
	config := new(DeployConf)
	if err := yamlparser.LoadConfig(filename, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func parseBusAddr(addr string) ([]uint64, error) {
	vs := strings.Split(addr, ".")
	if len(vs) != 4 {
		return nil, fmt.Errorf("bus address: %s is illegal, should be a.b.c.d", addr)
	}

	vi := make([]uint64, len(vs))
	for k, s := range vs {
		i, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bus address: %s is illegal, can not convert %s to uint64", addr, s)
		}
		vi[k] = i
	}
	return vi, nil
}
