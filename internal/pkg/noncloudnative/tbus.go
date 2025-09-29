package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// TbusChannel descript the bus relation of two different proc
type TbusChannel struct {
	ProcSet1  string `xml:"ProcSet1,attr,omitempty" json:"proc_set_1"`
	ProcSet2  string `xml:"ProcSet2,attr,omitempty" json:"proc_set_2"`
	Mask      string `xml:"Mask,attr,omitempty" json:"mask"`
	MaskArray []int  `xml:"-"`
}

// TbusConf represents a tbus configuration
type TbusConf struct {
	XMLName          xml.Name         `xml:"TbusConf"`
	BusAddrTemplate  string           `xml:"-" json:"tbus_addr_template"`
	BussinessID      int              `xml:"BussinessID,attr,omitempty" json:"bussiness_id"`
	GCIMShmKey       int              `xml:"GCIMShmKey,attr,omitempty" json:"gcim_shm_key"`
	GRMShmKey        int              `xml:"GRMShmKey,attr,omitempty" json:"grm_shm_key"`
	ChannelSize      int              `xml:"ChannelSize,attr,omitempty" json:"channel_size"`
	TbusdServicePort int              `xml:"TbusdServicePort,attr,omitempty" json:"tbusd_port"`
	TbusdConfPath    string           `xml:"TbusdConfPath,attr,omitempty" json:"tbusd_conf_path"`
	PkgMaxSize       int              `xml:"PkgMaxSize,attr,omitempty" json:"pkg_max_size"`
	SendBuff         int              `xml:"SendBuff,attr,omitempty" json:"send_buff"`
	RecvBuff         int              `xml:"RecvBuff,attr,omitempty" json:"recv_buff"`
	Channels         []*TbusChannel   `xml:"Channel,omitempty" json:"tbus_channel"`
	AddrPartBits     map[string]uint8 `xml:"-"`
	MaxInsID         int              `xml:"-"`
}

func parseBusAddr(addr string) ([]int, error) {
	vs := strings.Split(addr, ".")
	if len(vs) != 4 {
		return nil, fmt.Errorf("tbus address: %s is illegal", addr)
	}

	vi := make([]int, len(vs))
	for k, s := range vs {
		i, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("tbus address: %s is illegal", addr)
		}
		vi[k] = i
	}
	return vi, nil
}

func loadTbusConfig(filename string) (nonCloudNativeConf, error) {
	config := new(TbusConf)
	if err := yamlparser.LoadConfig(filename, config); err != nil {
		return nil, err
	}

	// gen mask bit array
	for _, ch := range config.Channels {
		values := strings.Split(ch.Mask, ".")
		if len(values) != 4 {
			return nil, fmt.Errorf("mask: %s, between %s and %s is illegal", ch.Mask, ch.ProcSet1, ch.ProcSet2)
		}
		ch.MaskArray = make([]int, 4)
		for i := 0; i < len(values); i++ {
			m, err := strconv.Atoi(values[i])
			if err != nil {
				return nil, fmt.Errorf("mask: %s, between %s and %s is illegal", ch.Mask, ch.ProcSet1, ch.ProcSet2)
			}
			ch.MaskArray[i] = m
		}
	}

	if config.AddrPartBits == nil {
		config.AddrPartBits = make(map[string]uint8)
	}

	if config.BusAddrTemplate != "" {
		addrs := strings.Split(config.BusAddrTemplate, ".")
		for _, addr := range addrs {
			values := strings.Split(addr, ":")
			if len(values) != 2 {
				return nil, fmt.Errorf("tbus addr template: %s is illegal", config.BusAddrTemplate)
			}
			bit, err := strconv.Atoi(values[1])
			if err != nil {
				return nil, fmt.Errorf("tbus addr template: %s is illegal", config.BusAddrTemplate)
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

// validate verify that tbus configuration data is illegal.
func (c *TbusConf) validate() error {
	var totalBits uint8
	totalBits = 0
	for _, v := range c.AddrPartBits {
		totalBits = totalBits + v
	}

	if totalBits != 32 {
		return fmt.Errorf("tbus addr template: %s is illegal", c.BusAddrTemplate)
	}

	bitsSlice := make([]uint8, 4)
	bitsSlice[0] = c.AddrPartBits["world"]
	bitsSlice[1] = c.AddrPartBits["zone"]
	bitsSlice[2] = c.AddrPartBits["function"]
	bitsSlice[3] = c.AddrPartBits["instance"]

	// verify bus channel mask
	for _, v := range c.Channels {
		for i := 0; i < len(v.MaskArray); i++ {
			if v.MaskArray[i] < 0 || v.MaskArray[i] >= int(math.Pow(2, float64(bitsSlice[i]))) {
				return fmt.Errorf("bus releation between %s and %s, mask: %s is illegal", v.ProcSet1, v.ProcSet2, v.Mask)
			}
		}
	}

	return nil
}

// GetAddrPartBit returns bits at address different segment.
func (c *TbusConf) GetAddrPartBit(name string) (uint8, error) {
	if v, ok := c.AddrPartBits[name]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("tbus addr template part: %s not exist", name)
}

// GetAddrTotalBits returns total bits of address.
func (c *TbusConf) GetAddrTotalBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"] + c.AddrPartBits["zone"] + c.AddrPartBits["world"]
}

// GetAddrWorldRightBits returns world segment right side total bits.
func (c *TbusConf) GetAddrWorldRightBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"] + c.AddrPartBits["zone"]
}

// GetAddrZoneRightBits returns zone segment right side total bits.
func (c *TbusConf) GetAddrZoneRightBits() uint8 {
	return c.AddrPartBits["instance"] + c.AddrPartBits["function"]
}

// GetAddrFuncRightBits returns func segment right side total bits.
func (c *TbusConf) GetAddrFuncRightBits() uint8 {
	return c.AddrPartBits["instance"]
}

// GetBriefBusAddrTemplate returns tbus address template.
func (c *TbusConf) GetBriefBusAddrTemplate() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		c.AddrPartBits["world"], c.AddrPartBits["zone"], c.AddrPartBits["function"], c.AddrPartBits["instance"])
}

// GetMaxInsID returns max instance id.
func (c *TbusConf) GetMaxInsID() int {
	return c.MaxInsID
}

// XMLExport export nonCloudNative tbus configuration.
func (c *TbusConf) XMLExport(outPath string) error {
	outfile := path.Join(outPath, "bus_relation.xml")
	wrapCfg := struct {
		XMLName struct{} `xml:"nonCloudNativecenter"`
		BusCfg  *TbusConf
	}{BusCfg: c}

	output, err := xml.MarshalIndent(wrapCfg, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal(bus_relation.xml): %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write(bus_relation.xml): %v", err)
	}
	return nil
}
