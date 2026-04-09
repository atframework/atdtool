package noncloudnative

import (
	"fmt"
	"strconv"
	"strings"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

type DeployUnit struct {
	Name            string `json:"chart_name"`
	TypeId          string `json:"instance_type_id"`
	WorldInstance   bool   `json:"world_instance"`
	InstanceCount   uint64 `json:"instance_count"`
	StartInstanceId uint64 `json:"start_instance_id"`
}

type DeployConf struct {
	WorldID  uint64        `json:"world_id"`
	ZoneId   uint64        `json:"zone_id"`
	Instance []*DeployUnit `json:"proc_desc"`
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
