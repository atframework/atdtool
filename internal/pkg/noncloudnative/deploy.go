package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
	"sigs.k8s.io/yaml"
)

// DeployNode represents a deploy node
type DeployNode struct {
	Host       string `json:"host"`
	Num        int    `json:"num"`
	StartInsID int    `json:"start_ins_id"`
}

// DeployUnit represents a deploy unit
type DeployUnit struct {
	Name  string        `json:"name"`
	Nodes []*DeployNode `json:"nodes"`
}

// DeployData represents a deploy data
type DeployData struct {
	Alias       string        `json:"alias_name"`
	WorldID     int           `json:"world_id"`
	ZoneID      int           `json:"zone_id"`
	OpeningTime string        `json:"opening_time"`
	Units       []*DeployUnit `json:"proc_deploy"`
}

// ProcDeployConf represents a proc deploy configuration.
type ProcDeployConf struct {
	Worlds  []*DeployData
	Zones   []*DeployData
	Deploys map[string][]*DeployElement
}

// DeployElement represents a deploy element
type DeployElement struct {
	Layer   int
	WorldID int
	ZoneID  int
	Nodes   []*DeployNode
}

// DeployKey represents a deploy key
type DeployKey struct {
	World int
	Zone  int
}

// DeployNodeDetail deploy node detail description
type DeployNodeDetail struct {
	Zone     int
	InnerIP  string
	Index    int
	Instance int
	UniqID   uint32
}

// DeployNodeAttr represents a deploy node attribute
type DeployNodeAttr struct {
	ID         int    `xml:"ID,attr,omitempty"`
	WorkPath   string `xml:"WorkPath,attr,omitempty"`
	CustomAttr string `xml:"CustomAttr,attr,omitempty"`
}

// nonCloudNativeDeployGroup represents a deploy group attribute
type nonCloudNativeDeployGroup struct {
	XMLName    xml.Name `xml:"DeloyGroup"`
	WorkPath   string   `xml:"WorkPath,attr,omitempty"`
	Group      string   `xml:"Group,attr,omitempty"`
	Host       string   `xml:"Host,attr,omitempty"`
	InstID     int      `xml:"InstID,attr,omitempty"`
	CustomAttr string   `xml:"CustomAttr,attr,omitempty"`
}

// DeployGroupSetter set group value
type DeployGroupSetter interface {
	setValue(name string, node *DeployNode)
}

// nonCloudNativeZoneDeploy represents a nonCloudNative zone deploy
type nonCloudNativeZoneDeploy struct {
	XMLName xml.Name `xml:"zone"`
	DeployNodeAttr
	CustomAttr string `xml:"CustomAttr,attr,omitempty"`
	Groups     []*nonCloudNativeDeployGroup
}

// nonCloudNativeWorldDeploy represents a nonCloudNative world deploy
type nonCloudNativeWorldDeploy struct {
	XMLName xml.Name `xml:"world"`
	DeployNodeAttr
	CustomAttr string `xml:"CustomAttr,attr,omitempty"`
	Groups     []*nonCloudNativeDeployGroup
	Zones      []*nonCloudNativeZoneDeploy
}

// nonCloudNativeClusterDeploy represents a nonCloudNative cluster deploy
type nonCloudNativeClusterDeploy struct {
	XMLName xml.Name `xml:"ClusterDeploy"`
	DeployNodeAttr
	CustomAttr string `xml:"CustomAttr,attr,omitempty"`
	Groups     []*nonCloudNativeDeployGroup
	Worlds     []*nonCloudNativeWorldDeploy
}

// nonCloudNativeProcDeployConf represents a nonCloudNative configuration
type nonCloudNativeProcDeployConf struct {
	XMLName xml.Name `xml:"nonCloudNativeCenter"`
	Cluster nonCloudNativeClusterDeploy
}

func loadProcDeployData(filename string) (nonCloudNativeConf, error) {
	deployMap := make(map[string]any)
	if err := yamlparser.LoadConfig(filename, &deployMap); err != nil {
		return nil, err
	}

	config := new(ProcDeployConf)
	config.Deploys = make(map[string][]*DeployElement)
	for k, v := range deployMap {
		keys := strings.Split(k, "_")
		if len(keys) <= 0 || (keys[0] != "world" && keys[0] != "zone") {
			return nil, fmt.Errorf("invalid deploy: (%s)", k)
		}

		// parse world area deploy
		if strings.Compare(keys[0], "world") == 0 {
			if len(keys) != 2 {
				return nil, fmt.Errorf("invalid deploy: (%s)", k)
			}

			worldID, err := strconv.ParseInt(keys[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("(%s) is not vaild world id", keys[1])
			}

			worldDeployData := &DeployData{}
			out, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}

			if err := yaml.Unmarshal(out, worldDeployData); err != nil {
				return nil, err
			}

			if worldID != int64(worldDeployData.WorldID) {
				return nil, fmt.Errorf("proc deploy: (%s) not match with world id: (%d)", k, worldDeployData.WorldID)
			}

			config.Worlds = append(config.Worlds, worldDeployData)
		}

		// parse zone area deploy
		if strings.Compare(keys[0], "zone") == 0 {
			if len(keys) != 3 {
				return nil, fmt.Errorf("invalid deploy: (%s)", k)
			}

			worldID, err := strconv.ParseInt(keys[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("(%s) is not vaild world id", keys[1])
			}

			zoneID, err := strconv.ParseInt(keys[2], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("(%s) is not vaild zone id", keys[2])
			}

			zoneDeployData := &DeployData{}
			out, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}

			if err := yaml.Unmarshal(out, zoneDeployData); err != nil {
				return nil, err
			}

			if worldID != int64(zoneDeployData.WorldID) {
				return nil, fmt.Errorf("proc deploy: (%s) not match with world id: (%d)", k, zoneDeployData.WorldID)
			}

			if zoneID != int64(zoneDeployData.ZoneID) {
				return nil, fmt.Errorf("proc deploy: (%s) not match with zone id: (%d)", k, zoneDeployData.ZoneID)
			}

			if zoneDeployData.OpeningTime != "" {
				if _, err := time.Parse("2006-01-02 15:04:05", zoneDeployData.OpeningTime); err != nil {
					return nil, fmt.Errorf("proc deploy: (%s) invalid opening time(%s) format", k, zoneDeployData.OpeningTime)
				}
			}

			config.Zones = append(config.Zones, zoneDeployData)
		}
	}

	if err := config.parseWorldDeploy(); err != nil {
		return nil, err
	}

	if err := config.parseZoneDeploy(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *ProcDeployConf) parseWorldDeploy() error {
	worldIdx := make(map[int]bool)
	for _, w := range c.Worlds {
		if _, ok := worldIdx[w.WorldID]; ok {
			return fmt.Errorf("duplicate world: %d deploy", w.WorldID)
		}

		worldIdx[w.WorldID] = true
		for _, u := range w.Units {
			if _, ok := c.Deploys[u.Name]; ok {
				bExist := false
				for _, d := range c.Deploys[u.Name] {
					if d.WorldID == w.WorldID {
						d.Nodes = append(d.Nodes, u.Nodes...)
						bExist = true
						break
					}
				}

				if !bExist {
					e := &DeployElement{}
					e.Layer = LayerWorld
					e.WorldID = w.WorldID
					e.ZoneID = 0
					e.Nodes = append(e.Nodes, u.Nodes...)
					c.Deploys[u.Name] = append(c.Deploys[u.Name], e)
				}
			} else {
				s := []*DeployElement{}
				e := &DeployElement{}
				e.Layer = LayerWorld
				e.WorldID = w.WorldID
				e.ZoneID = 0
				e.Nodes = append(e.Nodes, u.Nodes...)
				s = append(s, e)
				c.Deploys[u.Name] = s
			}
		}
	}
	return nil
}

func (c *ProcDeployConf) parseZoneDeploy() error {
	deployIdx := make(map[DeployKey]bool)
	for _, z := range c.Zones {
		key := DeployKey{
			World: z.WorldID,
			Zone:  z.ZoneID,
		}

		if _, ok := deployIdx[key]; ok {
			return fmt.Errorf("duplicate zone: %d, %d deploy", key.World, key.Zone)
		}

		deployIdx[key] = true
		for _, u := range z.Units {
			if _, ok := c.Deploys[u.Name]; ok {
				bExist := false
				for _, w := range c.Deploys[u.Name] {
					if w.WorldID == z.WorldID && w.ZoneID == z.ZoneID {
						w.Nodes = append(w.Nodes, u.Nodes...)
						bExist = true
						break
					}
				}

				if !bExist {
					e := &DeployElement{}
					e.Layer = LayerZone
					e.WorldID = z.WorldID
					e.ZoneID = z.ZoneID
					e.Nodes = append(e.Nodes, u.Nodes...)
					c.Deploys[u.Name] = append(c.Deploys[u.Name], e)
				}
			} else {
				s := make([]*DeployElement, 0)
				e := &DeployElement{}
				e.Layer = LayerZone
				e.WorldID = z.WorldID
				e.ZoneID = z.ZoneID
				e.Nodes = append(e.Nodes, u.Nodes...)
				s = append(s, e)
				c.Deploys[u.Name] = s
			}
		}
	}
	return nil
}

// GetDeployElements find depoly element by proc group name.
func (c *ProcDeployConf) GetDeployElements(procGroup string) []*DeployElement {
	if v, ok := c.Deploys[procGroup]; ok {
		return v
	}
	return nil
}

// GetDepolyNodes find depoly nodes.
func (c *ProcDeployConf) GetDepolyNodes(world int, zone int, procGroup string) []*DeployNode {
	if s, ok := c.Deploys[procGroup]; ok {
		for _, v := range s {
			if v.WorldID == world && v.ZoneID == zone {
				return v.Nodes
			}
		}
	}
	return nil
}

// GetDepolyIndex find depoly instance index.
func (c *ProcDeployConf) GetDepolyIndex(world, zone, instance int, procGroup, hostName string) (int, error) {
	if s, ok := c.Deploys[procGroup]; ok {
		for _, v := range s {
			if v.WorldID == world && v.ZoneID == zone {
				for _, n := range v.Nodes {
					if n.Host == hostName {
						if instance-n.StartInsID > n.Num {
							return 0, fmt.Errorf("no invalid instance")
						}
						return instance - n.StartInsID, nil
					}
				}
				break
			}
		}
	}
	return 0, fmt.Errorf("no invalid instance")
}

// GetZoneOpeningTime return zone opening time
func (c *ProcDeployConf) GetZoneOpeningTime(world, zone int) string {
	for _, v := range c.Zones {
		if v.WorldID == world && v.ZoneID == zone {
			return v.OpeningTime
		}
	}
	// if zone config is not found return empty
	return ""
}

// GetDepolyHostName find depoly host name.
func (c *ProcDeployConf) GetDepolyHostName(world, zone, instance int, procGroup string) (string, error) {
	if s, ok := c.Deploys[procGroup]; ok {
		for _, v := range s {
			if v.WorldID == world && v.ZoneID == zone {
				for _, n := range v.Nodes {
					if instance >= n.StartInsID && instance < n.StartInsID+n.Num {
						return n.Host, nil
					}
				}
				break
			}
		}
	}
	return "", fmt.Errorf("no valid host")
}

// IsWorldDeployExist returns true if query world deploy unit exist.
func (c *ProcDeployConf) IsWorldDeployExist(worldID int) bool {
	for _, v := range c.Worlds {
		if v.WorldID == worldID {
			return true
		}
	}
	return false
}

func (m *nonCloudNativeDeployGroup) setValue(name string, node *DeployNode, idx int) {
	if node != nil {
		m.Group = name
		m.Host = node.Host
		m.InstID = node.StartInsID + idx
	}
}

func createDeployNodes(procGroup string, nodes []*DeployNode) []*nonCloudNativeDeployGroup {
	groups := make([]*nonCloudNativeDeployGroup, 0)
	for _, node := range nodes {
		for i := 0; i < node.Num; i++ {
			g := &nonCloudNativeDeployGroup{}
			g.setValue(procGroup, node, i)
			groups = append(groups, g)
		}
	}
	return groups
}

// XMLExport export nonCloudNative proc deploy xml configuration
func (c *ProcDeployConf) XMLExport(outPath string) error {
	worlds := make(map[int]*nonCloudNativeWorldDeploy)
	zones := make(map[DeployKey]*nonCloudNativeZoneDeploy)
	for k, v := range c.Deploys {
		for _, e := range v {
			switch e.Layer {
			case LayerWorld: // deploy in the world layer
				if w, ok := worlds[e.WorldID]; ok {
					w.Groups = append(w.Groups, createDeployNodes(k, e.Nodes)...)
				} else {
					myWorld := &nonCloudNativeWorldDeploy{}
					myWorld.DeployNodeAttr.ID = e.WorldID
					myWorld.Groups = append(myWorld.Groups, createDeployNodes(k, e.Nodes)...)
					worlds[e.WorldID] = myWorld
				}
			case LayerZone: // deploy in the zone layer
				key := DeployKey{
					World: e.WorldID,
					Zone:  e.ZoneID,
				}
				if w, ok := worlds[e.WorldID]; ok {
					if z, ok := zones[key]; ok {
						z.Groups = append(z.Groups, createDeployNodes(k, e.Nodes)...)
					} else {
						myZone := &nonCloudNativeZoneDeploy{}
						myZone.DeployNodeAttr.ID = e.ZoneID
						myZone.Groups = append(myZone.Groups, createDeployNodes(k, e.Nodes)...)
						w.Zones = append(w.Zones, myZone)
						zones[key] = myZone
					}
				} else {
					myWorld := &nonCloudNativeWorldDeploy{}
					myWorld.DeployNodeAttr.ID = e.WorldID
					myZone := &nonCloudNativeZoneDeploy{}
					myZone.DeployNodeAttr.ID = e.ZoneID
					myZone.Groups = append(myZone.Groups, createDeployNodes(k, e.Nodes)...)
					myWorld.Zones = append(myWorld.Zones, myZone)
					worlds[e.WorldID] = myWorld
					zones[key] = myZone
				}
			}
		}
	}

	config := &nonCloudNativeProcDeployConf{}
	worldKeys := make([]int, 0)
	for k, v := range worlds {
		worldKeys = append(worldKeys, k)
		sort.SliceStable(v.Zones, func(i, j int) bool {
			return v.Zones[i].DeployNodeAttr.ID < v.Zones[j].DeployNodeAttr.ID
		})
	}

	sort.Ints(worldKeys)
	for _, key := range worldKeys {
		config.Cluster.Worlds = append(config.Cluster.Worlds, worlds[key])
	}

	outfile := path.Join(outPath, "proc_deploy.xml")
	output, err := xml.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal proc_deploy.xml: %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write proc_deploy.xml: %v", err)
	}
	return nil
}
