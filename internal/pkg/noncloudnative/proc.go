package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"strings"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// ProcNode is deploy proc node
type ProcNode struct {
	FuncID            int      `xml:"FuncID,attr,omitempty" json:"func_id"`
	FuncName          string   `xml:"FuncName,attr,omitempty" json:"func_name"`
	ProcName          string   `xml:"ProcName,attr,omitempty" json:"proc_name"`
	Agrs              string   `xml:"Agrs,attr,omitempty" json:"agrs"`
	Flag              string   `xml:"Flag,attr,omitempty" json:"flag"`
	WorkPath          string   `xml:"WorkPath,attr,omitempty" json:"work_path"`
	StartCmd          string   `xml:"StartCmd,attr,omitempty" json:"start_cmd"`
	ReStartCmd        string   `xml:"ReStartCmd,attr,omitempty" json:"restart_cmd"`
	StopCmd           string   `xml:"StopCmd,attr,omitempty" json:"stop_cmd"`
	KillCmd           string   `xml:"KillCmd,attr,omitempty" json:"kill_cmd"`
	PidFile           string   `xml:"PidFile,attr,omitempty" json:"pid_file"`
	ConfigPath        string   `xml:"ConfigPath,attr,omitempty" json:"conf_path"`
	ReloadCmd         string   `xml:"ReloadCmd,attr,omitempty" json:"reload_cmd"`
	RunShellCmd       string   `xml:"RunShellCmd,attr,omitempty" json:"runshell_cmd"`
	StartCheckCmd     string   `xml:"StartCheckCmd,attr,omitempty" json:"start_check_cmd"`
	StartCheckEndTime int      `xml:"StartCheckEndTime,attr,omitempty" json:"start_check_endtime"`
	IsCommon          int      `xml:"IsCommon,attr,omitempty" json:"is_common"`
	Seq               int      `xml:"Seq,attr,omitempty" json:"seq"`
	AutoScript        string   `xml:"AutoScript,attr,omitempty" json:"auto_script"`
	OpTimeout         int      `xml:"OpTimeout,attr,omitempty" json:"op_timeout"`
	DependModules     []string `xml:"-" json:"depend_modules"`
}

// ProcGroupNode represents a proc group node
type ProcGroupNode struct {
	Name  string   `json:"name"`
	Layer string   `json:"layer"`
	Procs []string `json:"proc_funcs"`
}

// LayerNodeAttr is proc attribute
type LayerNodeAttr struct {
	WorkPath     string `xml:"WorkPath,attr,omitempty" json:"work_path"`
	AutoTimeGap  int    `xml:"AutoTimeGap,attr,omitempty" json:"auto_time_gap"`
	MsgRoundTime int    `xml:"MsgRoundTime,attr,omitempty" json:"msg_round_time"`
	OpTimeout    int    `xml:"OpTimeout,attr,omitempty" json:"op_timeout"`
	Isolated     int    `xml:"Isolated,attr,omitempty" json:"isolated"`
}

// ZoneLayer is nonCloudNative zone layer configuration
type ZoneLayer struct {
	XMLName xml.Name `xml:"zone"`
	LayerNodeAttr
	Procs []*ProcNode `xml:"Proc"`
}

// WorldLayer is nonCloudNative world layer configuration
type WorldLayer struct {
	XMLName xml.Name `xml:"world"`
	LayerNodeAttr
	Procs []*ProcNode `xml:"Proc"`
	Zone  *ZoneLayer
}

// ClusterLayer is nonCloudNative cluster layer configuration
type ClusterLayer struct {
	XMLName xml.Name `xml:"cluster"`
	LayerNodeAttr
	Procs []*ProcNode `xml:"Proc"`
	World *WorldLayer
}

// ProcConf represents the proc configuration
type ProcConf struct {
	ClusterAttr LayerNodeAttr    `json:"cluster_attr"`
	WorldAttr   LayerNodeAttr    `json:"world_attr"`
	ZoneAttr    LayerNodeAttr    `json:"zone_attr"`
	Groups      []*ProcGroupNode `json:"proc_group"`
	Procs       []*ProcNode      `json:"proc_desc"`
}

// ProcBriefNode is proc node brief description
type ProcBriefNode struct {
	FuncName string `xml:"FuncName,attr"`
}

// nonCloudNativeProcGroupNode represents a proc group node in xml configuration
type nonCloudNativeProcGroupNode struct {
	Name  string           `xml:"Name,attr"`
	Layer string           `xml:"Layer,attr"`
	Procs []*ProcBriefNode `xml:"Proc"`
}

// nonCloudNativeProcConf represents a proc description in xml configuration
type nonCloudNativeProcConf struct {
	XMLName xml.Name `xml:"nonCloudNativecenter"`
	Cluster ClusterLayer
	Groups  []*nonCloudNativeProcGroupNode `xml:"ProcGroup"`
}

func loadProcConfig(filename string) (nonCloudNativeConf, error) {
	config := new(ProcConf)
	if err := yamlparser.LoadConfig(filename, config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *ProcConf) validateProcGroup() error {
	groups := make([]*ProcGroupNode, 0)
	for _, g := range c.Groups {
		// If current group has repeated proc, return err
		procs := make(map[string]bool)
		for _, funcName := range g.Procs {
			if _, ok := procs[funcName]; ok {
				return fmt.Errorf("%s has repeated proc: %s", g.Name, funcName)
			}
			procs[funcName] = true

			// If proc not in proc list, return err
			found := false
			for _, p := range c.Procs {
				if p.FuncName == funcName {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("%s has unregistered proc: %s", g.Name, funcName)
			}
		}

		for _, t := range groups {
			// If has repeated group, return err
			if g.Name == t.Name {
				return fmt.Errorf("duplicate proc group: %s", g.Name)
			}

			// If different group has same proc, return err
			for _, e := range g.Procs {
				for _, l := range t.Procs {
					if strings.Compare(strings.ToLower(e), strings.ToLower(l)) == 0 {
						return fmt.Errorf("duplicate proc: %s, between %s and %s", e, g.Name, t.Name)
					}
				}
			}
		}
		groups = append(groups, g)
	}
	return nil
}

func (c *ProcConf) validateProc() error {
	procs := make([]*ProcNode, 0)
	for _, proc := range c.Procs {
		for _, t := range procs {
			// If current proc has same func ID with othe proc, return err
			if proc.FuncID == t.FuncID {
				return fmt.Errorf("duplicate proc func id: %d, between %s and %s", proc.FuncID, proc.ProcName, t.FuncName)
			}

			// If current proc has same fun name with other proc, return err
			if strings.Compare(strings.ToLower(proc.FuncName), strings.ToLower(t.FuncName)) == 0 {
				return fmt.Errorf("duplicate proc func name: %s", proc.FuncName)
			}

			// If current proc has same proc name with other proc, return err
			//if strings.Compare(strings.ToLower(proc.ProcName), strings.ToLower(t.ProcName)) == 0 {
			//	return fmt.Errorf("duplicate proc name: %s, between %s and %s", proc.ProcName, proc.FuncName, t.FuncName)
			//}

			// If current proc has same workpath with other proc, return err
			//if strings.Compare(strings.ToLower(proc.WorkPath), strings.ToLower(t.WorkPath)) == 0 {
			//	return fmt.Errorf("duplicate proc workpath: %s, between %s and %s", proc.WorkPath, proc.FuncName, t.FuncName)
			//}
		}
		procs = append(procs, proc)
	}
	return nil
}

// validate verify that proc configuration data.
func (c *ProcConf) validate() error {
	if err := c.validateProcGroup(); err != nil {
		return err
	}

	if err := c.validateProc(); err != nil {
		return err
	}
	return nil
}

// update proc auto script
func (c *ProcConf) updateAutoScript(autoScript string) {
	for _, v := range c.Procs {
		v.AutoScript = autoScript
	}
}

// GetProcNodeByFuncName find proc node data by func name.
func (c *ProcConf) GetProcNodeByFuncName(name string) *ProcNode {
	for _, Proc := range c.Procs {
		if Proc.FuncName == name {
			return Proc
		}
	}
	return nil
}

// GetProcNodeByFuncID find proc node data by func id.
func (c *ProcConf) GetProcNodeByFuncID(id int) *ProcNode {
	for _, Proc := range c.Procs {
		if Proc.FuncID == id {
			return Proc
		}
	}
	return nil
}

// GetProcGroup find proc group node data by name.
func (c *ProcConf) GetProcGroup(name string) *ProcGroupNode {
	for _, g := range c.Groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// GetProcGroupByFuncName find proc group node data by func name.
func (c *ProcConf) GetProcGroupByFuncName(name string) *ProcGroupNode {
	for _, g := range c.Groups {
		for _, v := range g.Procs {
			if v == name {
				return g
			}
		}
	}
	return nil
}

// GetGroupLayer find proc group layer by name.
func (c *ProcConf) GetGroupLayer(name string) int {
	if group := c.GetProcGroup(name); group != nil {
		return GetLayer(group.Layer)
	}
	return LayerNone
}

// GetProcGroupsByLayer find current layer all proc groups.
func (c *ProcConf) GetProcGroupsByLayer(layer int) []ProcGroupNode {
	groups := []ProcGroupNode{}

	layerName := GetLayerName(layer)
	if layerName != "unknown" {
		for _, g := range c.Groups {
			if g.Layer == layerName {
				groups = append(groups, *g)
			}
		}
	}
	return groups
}

// XMLExport export nonCloudNative proc configuration
func (c *ProcConf) XMLExport(outPath string) error {
	nonCloudNativeCfg := &nonCloudNativeProcConf{}
	nonCloudNativeCfg.Cluster.LayerNodeAttr = c.ClusterAttr

	for _, deployGroup := range c.Groups {
		g := &nonCloudNativeProcGroupNode{Name: deployGroup.Name, Layer: deployGroup.Layer}
		for _, FuncName := range deployGroup.Procs {
			proc := c.GetProcNodeByFuncName(FuncName)
			if proc == nil {
				return fmt.Errorf("proc[%s]: not found in the configuration file", FuncName)
			}

			g.Procs = append(g.Procs, &ProcBriefNode{FuncName: FuncName})

			switch GetLayer(deployGroup.Layer) {
			case LayerCluster:
				nonCloudNativeCfg.Cluster.Procs = append(nonCloudNativeCfg.Cluster.Procs, proc)
			case LayerWorld:
				if nonCloudNativeCfg.Cluster.World == nil {
					nonCloudNativeCfg.Cluster.World = &WorldLayer{}
					nonCloudNativeCfg.Cluster.World.LayerNodeAttr = c.WorldAttr
				}
				nonCloudNativeCfg.Cluster.World.Procs = append(nonCloudNativeCfg.Cluster.World.Procs, proc)
			case LayerZone:
				if nonCloudNativeCfg.Cluster.World == nil {
					nonCloudNativeCfg.Cluster.World = &WorldLayer{}
					nonCloudNativeCfg.Cluster.World.LayerNodeAttr = c.WorldAttr
				}

				if nonCloudNativeCfg.Cluster.World.Zone == nil {
					nonCloudNativeCfg.Cluster.World.Zone = &ZoneLayer{}
					nonCloudNativeCfg.Cluster.World.Zone.LayerNodeAttr = c.ZoneAttr
				}

				nonCloudNativeCfg.Cluster.World.Zone.Procs = append(nonCloudNativeCfg.Cluster.World.Zone.Procs, proc)
			default:
				return fmt.Errorf("layer[%s]: invalid", deployGroup.Layer)
			}
		}

		nonCloudNativeCfg.Groups = append(nonCloudNativeCfg.Groups, g)
	}

	if nonCloudNativeCfg.Cluster.World.Zone == nil {
		nonCloudNativeCfg.Cluster.World.Zone = new(ZoneLayer)
		nonCloudNativeCfg.Cluster.World.Zone.LayerNodeAttr = c.ZoneAttr
	}

	outfile := path.Join(outPath, "proc.xml")
	output, err := xml.MarshalIndent(nonCloudNativeCfg, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal(proc.xml): %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write(proc.xml): %v", err)
	}
	return nil
}
