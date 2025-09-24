package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// WhiteListConf represents a tcm white list configuration
type WhiteListConf struct {
	XMLName xml.Name  `xml:"tcmcenter"`
	IPList  []*string `xml:"AccessWhiteList>ipList" json:"ip_list"`
}

func loadWhiterListConfig(filename string) (tcmConf, error) {
	config := new(WhiteListConf)
	if err := yamlparser.LoadConfig(filename, config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// validate verify that whitelist configuration data is illegal.
func (c *WhiteListConf) validate() error {
	return nil
}

// XMLExport export tcm whitelist xml configuration
func (c *WhiteListConf) XMLExport(outPath string) error {
	outfile := path.Join(outPath, "access_whitelist.xml")
	output, err := xml.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal(access_whitelist.xml): %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write(access_whitelist.xml): %v", err)
	}

	return nil
}
