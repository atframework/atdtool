package noncloudnative

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

type DeployUnit struct {
	Name          string `json:"chart_name"`
	TypeId        string `json:"instance_type_id"`
	WorldInstance bool   `json:"world_instance"`
	Num           int    `json:"num"`
	StartInsId    int    `json:"start_ins_id"`
}

type DeployConf struct {
	BusAddrTemplate string        `json:"bus_addr_template"`
	WorldID         int           `json:"world_id"`
	ZoneId          int           `json:"zone_id"`
	Instance        []*DeployUnit `json:"proc_desc"`

	AddrPartBits map[string]uint8
	MaxInsID     int
}

func loadDeployData(filename string) (interface{}, error) {
	config := new(DeployConf)
	if err := yamlparser.LoadConfig(filename, &config); err != nil {
		return nil, err
	}

	if config.AddrPartBits == nil {
		config.AddrPartBits = make(map[string]uint8)
	}

	if config.BusAddrTemplate != "" {
		addrs := strings.Split(config.BusAddrTemplate, ".")
		for _, addr := range addrs {
			values := strings.Split(addr, ":")
			if len(values) != 2 {
				return nil, fmt.Errorf("bus addr template: %s is illegal", config.BusAddrTemplate)
			}
			bit, err := strconv.Atoi(values[1])
			if err != nil {
				return nil, fmt.Errorf("bus addr template: %s is illegal", config.BusAddrTemplate)
			}
			config.AddrPartBits[values[0]] = uint8(bit)
		}
	} else {
		config.AddrPartBits["world"] = 8
		config.AddrPartBits["zone"] = 8
		config.AddrPartBits["function"] = 8
		config.AddrPartBits["instance"] = 8
		config.BusAddrTemplate = fmt.Sprintf("world:%d.zone:%d.function:%d.instance:%d",
			config.AddrPartBits["world"], config.AddrPartBits["zone"], config.AddrPartBits["function"], config.AddrPartBits["instance"])
	}

	config.MaxInsID = int(math.Pow(2, float64(config.AddrPartBits["instance"]))) - 1
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func parseBusAddr(addr string) ([]int, error) {
	vs := strings.Split(addr, ".")
	if len(vs) != 4 {
		return nil, fmt.Errorf("bus address: %s is illegal", addr)
	}

	vi := make([]int, len(vs))
	for k, s := range vs {
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("bus address: %s is illegal", addr)
		}
		vi[k] = i
	}
	return vi, nil
}

// validate verify that bus configuration data is illegal.
func (c *DeployConf) validate() error {
	var totalBits uint8
	totalBits = 0
	for _, v := range c.AddrPartBits {
		totalBits = totalBits + v
	}

	if totalBits != 32 {
		return fmt.Errorf("bus addr template: %s is illegal", c.BusAddrTemplate)
	}

	bitsSlice := make([]uint8, 4)
	bitsSlice[0] = c.AddrPartBits["world"]
	bitsSlice[1] = c.AddrPartBits["zone"]
	bitsSlice[2] = c.AddrPartBits["function"]
	bitsSlice[3] = c.AddrPartBits["instance"]

	return nil
}

// GetAddrPartBit returns bits at address different segment.
func (c *DeployConf) GetAddrPartBit(name string) (uint8, error) {
	if v, ok := c.AddrPartBits[name]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("bus addr template part: %s not exist", name)
}

// GetAddrTotalBits returns total bits of address.
func (c *DeployConf) GetAddrTotalBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"] + c.AddrPartBits["zone"] + c.AddrPartBits["world"]
}

// GetAddrWorldRightBits returns world segment right side total bits.
func (c *DeployConf) GetAddrWorldRightBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"] + c.AddrPartBits["zone"]
}

// GetAddrZoneRightBits returns zone segment right side total bits.
func (c *DeployConf) GetAddrZoneRightBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"]
}

// GetAddrFuncRightBits returns func segment right side total bits.
func (c *DeployConf) GetAddrFuncRightBits() uint8 {
	return c.AddrPartBits["instance"]
}

// GetBriefBusAddrTemplate returns bus address template.
func (c *DeployConf) GetBriefBusAddrTemplate() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		c.AddrPartBits["world"], c.AddrPartBits["zone"], c.AddrPartBits["function"], c.AddrPartBits["instance"])
}

// GetMaxInsID returns max instance id.
func (c *DeployConf) GetMaxInsID() int {
	return c.MaxInsID
}
