// Package nonCloudNative provides functions for generating nonCloudNative deploy tool configuration files,
// and implements data interface for generating process configuration file.
package noncloudnative

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type RenderValue struct {
	BusAddr string  `json:"busAddr,omitempty"`
	Config  *Config `json:"config,omitempty"`
}

// Config is a set of configuration
type Config struct {
	Deploy *DeployConf `module:"deploy"`
}

var confLoader = map[string]func(string) (interface{}, error){
	"deploy": loadDeployData,
}

// LoadConfig load nonCloudNative configuration data
func LoadConfig(cfgPaths []string) (*Config, error) {
	config := new(Config)
	rtyp := reflect.TypeOf(config).Elem()
	for i := 0; i < rtyp.NumField(); i++ {
		loader, ok := confLoader[rtyp.Field(i).Tag.Get("module")]
		if ok {
			name := fmt.Sprintf("%s.yaml", rtyp.Field(i).Tag.Get("module"))
			var rs interface{}
			for _, cfgPath := range cfgPaths {
				if walkErr := filepath.Walk(cfgPath, func(filename string, fi os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !fi.IsDir() && strings.Compare(fi.Name(), name) == 0 {
						// the nonCloudNative module configuration will be completely replaced
						rs, err = loader(filename)
						if err != nil {
							return err
						}
					}
					return nil
				}); walkErr != nil {
					return nil, walkErr
				}
			}

			if rs == nil || reflect.TypeOf(rs).Kind() != reflect.Ptr || reflect.ValueOf(rs).IsNil() {
				return nil, fmt.Errorf("load nonCloudNative configuration file(%s) not found", name)
			}

			if !rtyp.Field(i).Type.AssignableTo(reflect.TypeOf(rs)) {
				return nil, fmt.Errorf("can't assign %v to %v", rtyp.Field(i).Type, reflect.TypeOf(rs))
			}
			reflect.ValueOf(config).Elem().Field(i).Set(reflect.ValueOf(rs))
		}
	}
	return config, nil
}

// UniqID returns proc uniq id.
func (c *Config) UniqID(worldID, zoneID, funcID, insID int) uint32 {
	var uniqID uint32
	uniqID = 0
	uniqID |= uint32(worldID)
	uniqID = uniqID << uint32(c.Deploy.AddrPartBits["zone"])
	uniqID |= uint32(zoneID)
	uniqID = uniqID << uint32(c.Deploy.AddrPartBits["function"])
	uniqID |= uint32(funcID)
	uniqID = uniqID << uint32(c.Deploy.AddrPartBits["instance"])
	uniqID |= uint32(insID)
	return uniqID
}

// ZoneBase returns base of logic id
func (c *Config) ZoneBase() uint32 {
	var maxVal uint32 = 1 << uint32(c.Deploy.AddrPartBits["zone"])
	var base uint32 = 1

	for base <= maxVal {
		base = base * 10
	}
	return base
}

// LogicID returns proc logic id.
func (c *Config) LogicID(worldID, zoneID int) uint32 {
	return uint32(worldID)*c.ZoneBase() + uint32(zoneID)
}

func (c *Config) ToRenderValues(addr string) (values map[string]any, err error) {
	addrs, err := parseBusAddr(addr)
	if err != nil {
		return
	}

	worldID, zoneID, funcID, insID := addrs[0], addrs[1], addrs[2], addrs[3]

	values = make(map[string]any)
	values["instance_id"] = insID
	values["bus_addr"] = addr
	values["world_id"] = worldID
	values["zone_id"] = zoneID
	values["zone_base"] = c.ZoneBase()
	values["logic_id"] = c.LogicID(worldID, zoneID)
	values["uniq_id"] = c.UniqID(worldID, zoneID, funcID, insID)

	values["bus_addr_template"] = c.Deploy.GetBriefBusAddrTemplate()
	values["addr_total_bits"] = c.Deploy.GetAddrTotalBits()
	values["world_right_bits"] = c.Deploy.GetAddrWorldRightBits()
	values["zone_right_bits"] = c.Deploy.GetAddrZoneRightBits()
	values["func_right_bits"] = c.Deploy.GetAddrFuncRightBits()
	return
}
