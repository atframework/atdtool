package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// Host represents a deploy host element
type Host struct {
	Name      string `xml:"Name,attr" json:"name"`
	InnerIP   string `xml:"InnerIP,attr" json:"inner_ip"`
	IsVirtual int    `xml:"IsVirtual,attr,omitempty" json:"is_virtual"`
}

// HostConf is host configuration
type HostConf struct {
	XMLName                xml.Name `xml:"tcmcenter"`
	AllowDuplicatedInnerIP bool     `xml:"-" json:"allow_duplicated_inner_ip"`
	Hosts                  []*Host  `xml:"HostTab>Host" json:"hosts"`
}

func loadHostConfig(filename string) (tcmConf, error) {
	config := new(HostConf)
	if err := yamlparser.LoadConfig(filename, config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// validate verify that host configuration data is illegal
func (c *HostConf) validate() error {
	names := make(map[string]bool)
	innerIPs := make(map[string]bool)
	for _, host := range c.Hosts {
		if _, ok := names[host.Name]; ok {
			return fmt.Errorf("duplicate host: %s", host.Name)
		}

		names[host.Name] = true

		if !c.AllowDuplicatedInnerIP {
			if _, ok := innerIPs[host.InnerIP]; ok {
				return fmt.Errorf("duplicate inner IP: %s", host.InnerIP)
			}
		}
		innerIPs[host.InnerIP] = true
	}
	return nil
}

// HasHost if host configuration data has input key, return true
func (c *HostConf) HasHost(name string) bool {
	for _, h := range c.Hosts {
		if h.Name == name {
			return true
		}
	}
	return false
}

// GetHost get host data by name
func (c *HostConf) GetHost(name string) *Host {
	for _, h := range c.Hosts {
		if h.Name == name {
			return h
		}
	}
	return nil
}

// GetInnerIP get deploy inner ip
func (c *HostConf) GetInnerIP(name string) string {
	for _, h := range c.Hosts {
		if h.Name == name {
			return h.InnerIP
		}
	}
	return ""
}

// XMLExport export tcm host configuration
func (c *HostConf) XMLExport(outPath string) error {
	outfile := path.Join(outPath, "host.xml")
	output, err := xml.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal(host.xml): %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write(host.xml): %v", err)
	}
	return nil
}
