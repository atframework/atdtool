// Package nonCloudNative provides functions for generating nonCloudNative deploy tool configuration files,
// and implements data interface for generating process configuration file.
package noncloudnative

import (
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

type RenderValue struct {
	BusAddr  string  `json:"busAddr,omitempty"`
	Hostname string  `json:"hostname,omitempty"`
	Config   *Config `json:"config,omitempty"`
}

// Config is a set of configuration
type Config struct {
	Center     *CenterConf     `module:"nonCloudNativecenter"`
	WhiteList  *WhiteListConf  `module:"access_whitelist"`
	Proc       *ProcConf       `module:"proc"`
	Host       *HostConf       `module:"host"`
	TBus       *TbusConf       `module:"bus_relation"`
	ProcDeploy *ProcDeployConf `module:"proc_deploy"`
}

type nonCloudNativeConf interface {
	XMLExport(outPath string) error
}

var confLoader = map[string]func(string) (nonCloudNativeConf, error){
	"nonCloudNativecenter": loadCenterConfig,
	"access_whitelist":     loadWhiterListConfig,
	"proc":                 loadProcConfig,
	"host":                 loadHostConfig,
	"bus_relation":         loadTbusConfig,
	"proc_deploy":          loadProcDeployData,
}

// LoadConfig load nonCloudNative configuration data
func LoadConfig(cfgPaths []string) (*Config, error) {
	config := new(Config)
	rtyp := reflect.TypeOf(config).Elem()
	for i := 0; i < rtyp.NumField(); i++ {
		loader, ok := confLoader[rtyp.Field(i).Tag.Get("module")]
		if ok {
			name := fmt.Sprintf("%s.yaml", rtyp.Field(i).Tag.Get("module"))
			var rs nonCloudNativeConf
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

	// set atdtool configuration file path
	config.Center.ConfigValueDir = strings.Join(cfgPaths, ",")

	// set center configuration tbus addr template
	config.Center.AddrTemple = config.TBus.BusAddrTemplate

	// we shoud verify proc deploy data.
	if err := config.verifyProcDeploy(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) isValidWorldID(worldID int) bool {
	var maxWorldID int = int(math.Pow(2, float64(c.TBus.AddrPartBits["world"])))
	if worldID <= 0 || worldID >= maxWorldID {
		return false
	}
	return true
}

func (c *Config) isValidZoneID(zoneID int) bool {
	var maxZoneID int = int(math.Pow(2, float64(c.TBus.AddrPartBits["zone"])))
	if zoneID <= 0 || zoneID >= maxZoneID {
		return false
	}
	return true
}

// if host is not found in host configuration data or has repeated host, return err
// if has repeated instance,  return err
func (c *Config) validateDeployNode(unit *DeployUnit) error {
	result := []*DeployNode{}
	instances := make(map[int]bool)
	for _, n := range unit.Nodes {
		if !c.Host.HasHost(n.Host) {
			return fmt.Errorf("invalid host:%s in %s", n.Host, unit.Name)
		}

		for _, t := range result {
			if n.Host == t.Host {
				return fmt.Errorf("duplicate host:%s in %s", n.Host, unit.Name)
			}
		}

		for i := 0; i < n.Num; i++ {
			insID := n.StartInsID + i
			if insID > c.TBus.GetMaxInsID() {
				return fmt.Errorf("instance:%d overflow in %s, must less than or equal to:%d", insID, unit.Name, c.TBus.GetMaxInsID())
			}

			if _, ok := instances[insID]; ok {
				return fmt.Errorf("duplicate instance:%d in %s, begin instance:%d", insID, unit.Name, n.StartInsID)
			}
			instances[insID] = true
		}
		result = append(result, n)
	}
	return nil
}

// validate world layer deploy configuration data
func (c *Config) validateWorldDeploy(worlds []*DeployData) error {
	for _, w := range worlds {
		if !c.isValidWorldID(w.WorldID) {
			return fmt.Errorf("invalid deploy world id:%d", w.WorldID)
		}
		nodes := make(map[string]bool)
		for _, u := range w.Units {
			if _, ok := nodes[u.Name]; ok {
				return fmt.Errorf("duplicate proc group:%s in world:%d", u.Name, w.WorldID)
			}
			nodes[u.Name] = true
			g := c.Proc.GetProcGroup(u.Name)
			// If deploy group can't deploy in world layer, return err
			if g == nil || GetLayer(g.Layer) != LayerWorld {
				return fmt.Errorf("invalid proc group:%s in world:%d", u.Name, w.WorldID)
			}
			// If world deploy has invalid node, return err
			if err := c.validateDeployNode(u); err != nil {
				return fmt.Errorf("proc deploy in world(%d): %v", w.WorldID, err)
			}
		}
	}
	return nil
}

// check zone layer configuration data is illegal
// if it has repeated proc group or deploy the wrong layer proc group, return err
// if check deploy node in the zone failed, return err
func (c *Config) validateZoneDeploy(zones []*DeployData) error {
	for _, zone := range zones {
		if !c.isValidZoneID(zone.ZoneID) {
			return fmt.Errorf("invalid deploy zone id:%d, in world:%d", zone.ZoneID, zone.WorldID)
		}

		if !c.ProcDeploy.IsWorldDeployExist(zone.WorldID) {
			return fmt.Errorf("zone:%d deployed in the world:%d, but world not exist", zone.ZoneID, zone.WorldID)
		}

		nodes := make(map[string]bool)
		for _, u := range zone.Units {
			if _, ok := nodes[u.Name]; ok {
				return fmt.Errorf("duplicate proc group:%s in world:%d zone:%d", u.Name, zone.WorldID, zone.ZoneID)
			}
			nodes[u.Name] = true
			g := c.Proc.GetProcGroup(u.Name)
			// If deploy group can't deploy in zone layer, return err
			if g == nil || GetLayer(g.Layer) != LayerZone {
				return fmt.Errorf("invalid proc group:%s in world:%d zone:%d", u.Name, zone.WorldID, zone.ZoneID)
			}
			// If zone deploy has invalid node, return err
			if err := c.validateDeployNode(u); err != nil {
				return fmt.Errorf("proc deploy in world(%d) zone(%d): %v", zone.WorldID, zone.ZoneID, err)
			}
		}
	}
	return nil
}

func (c *Config) verifyProcDeploy() error {
	if err := c.validateWorldDeploy(c.ProcDeploy.Worlds); err != nil {
		return err
	}

	if err := c.validateZoneDeploy(c.ProcDeploy.Zones); err != nil {
		return err
	}
	return nil
}

// UniqID returns proc uniq id.
func (c *Config) UniqID(worldID, zoneID, funcID, insID int) uint32 {
	var uniqID uint32
	uniqID = 0
	uniqID |= uint32(worldID)
	uniqID = uniqID << uint32(c.TBus.AddrPartBits["zone"])
	uniqID |= uint32(zoneID)
	uniqID = uniqID << uint32(c.TBus.AddrPartBits["function"])
	uniqID |= uint32(funcID)
	uniqID = uniqID << uint32(c.TBus.AddrPartBits["instance"])
	uniqID |= uint32(insID)
	return uniqID
}

// ZoneBase returns base of logic id
func (c *Config) ZoneBase() uint32 {
	var maxVal uint32 = 1 << uint32(c.TBus.AddrPartBits["zone"])
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

// GetProcDeployNodes find proc deploy nodes.
// it will only collect the same world deploy node.
func (c *Config) GetProcDeployNodes(worldID int) map[string][]*DeployNodeDetail {
	procDeployNodes := make(map[string][]*DeployNodeDetail)
	for _, g := range c.Proc.Groups {
		for _, funcName := range g.Procs {
			proc := c.Proc.GetProcNodeByFuncName(funcName)
			deployElements := c.ProcDeploy.GetDeployElements(g.Name)
			if deployElements == nil {
				// current proc has no valid deploy node, do nothing
				continue
			}

			for _, e := range deployElements {
				if e.WorldID != worldID {
					// do nothing
					continue
				}

				for _, n := range e.Nodes {
					innerIP := c.Host.GetInnerIP(n.Host)
					if innerIP == "" {
						continue
					}

					for i := 0; i < n.Num; i++ {
						node := &DeployNodeDetail{}
						node.Zone = e.ZoneID
						node.InnerIP = innerIP
						node.Index = i
						node.Instance = n.StartInsID + i
						node.UniqID = c.UniqID(e.WorldID, e.ZoneID, proc.FuncID, n.StartInsID+i)
						procDeployNodes[funcName] = append(procDeployNodes[funcName], node)
					}
				}
			}
		}
	}
	return procDeployNodes
}

func (c *Config) ToRenderValues(addr, host string) (values map[string]any, err error) {
	addrs, err := parseBusAddr(addr)
	if err != nil {
		return
	}

	worldID, zoneID, funcID, insID := addrs[0], addrs[1], addrs[2], addrs[3]
	proc := c.Proc.GetProcNodeByFuncID(funcID)
	if proc == nil {
		err = fmt.Errorf("proc (%d) not exist", funcID)
		return
	}

	g := c.Proc.GetProcGroupByFuncName(proc.FuncName)
	if g == nil {
		err = fmt.Errorf("proc: %s has no valid proc group", proc.FuncName)
		return
	}

	h := c.Host.GetHost(host)
	if h == nil {
		err = fmt.Errorf("host: %s not exist", host)
		return
	}

	var (
		name string
		idx  int
	)
	name, err = c.ProcDeploy.GetDepolyHostName(worldID, zoneID, insID, g.Name)
	if err != nil {
		return
	}

	if name != host {
		err = fmt.Errorf("deploy host %s and %s missmatch", host, name)
		return
	}

	idx, err = c.ProcDeploy.GetDepolyIndex(worldID, zoneID, insID, g.Name, host)
	if err != nil {
		return
	}

	values = make(map[string]any)
	values["nonCloudNative_mode"] = true
	values["type_name"] = proc.FuncName
	values["type_id"] = proc.FuncID
	values["proc_name"] = proc.ProcName
	values["instance_id"] = insID
	values["instance_index"] = idx
	values["inner_ip"] = h.InnerIP
	values["bus_addr"] = addr
	values["world_id"] = worldID
	values["zone_id"] = zoneID
	values["zone_base"] = c.ZoneBase()
	values["logic_id"] = c.LogicID(worldID, zoneID)
	values["uniq_id"] = c.UniqID(worldID, zoneID, funcID, insID)
	values["opening_time"] = c.ProcDeploy.GetZoneOpeningTime(worldID, zoneID)

	pidPath := filepath.Dir(proc.PidFile)
	pidFile := path.Join(pidPath, fmt.Sprintf("%s.pid", addr))
	values["pid_file"] = pidFile
	values["bus_key"] = c.TBus.GCIMShmKey
	values["bus_channel_size"] = c.TBus.ChannelSize
	values["bus_addr_template"] = c.TBus.GetBriefBusAddrTemplate()
	values["bus_recv_buff_size"] = c.TBus.RecvBuff
	values["bus_send_buff_size"] = c.TBus.SendBuff
	values["addr_total_bits"] = c.TBus.GetAddrTotalBits()
	values["world_right_bits"] = c.TBus.GetAddrWorldRightBits()
	values["zone_right_bits"] = c.TBus.GetAddrZoneRightBits()
	values["func_right_bits"] = c.TBus.GetAddrFuncRightBits()
	values["proc_work_path"] = path.Join(c.Proc.ClusterAttr.WorkPath, proc.WorkPath)
	procDeployNodes := c.GetProcDeployNodes(worldID)
	for k, v := range procDeployNodes {
		values[k+"_deploy_nodes"] = v
	}
	return
}
