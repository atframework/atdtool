package noncloudnative

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"

	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// CenterConf nonCloudNative center configuration
type CenterConf struct {
	XMLName               xml.Name `xml:"nonCloudNativecenter"`
	ConfigTemplateDir     string   `xml:"ConfigTemplatePath,attr" json:"config_template_dir"`
	ConfigValueDir        string   `xml:"ConfigValueDir,attr" json:"-"`
	AutoScriptDir         string   `xml:"AutoScriptDir,attr" json:"auto_script_dir"`
	AddrTemple            string   `xml:"AddrTemple,attr" json:"-"`
	BussinessID           string   `xml:"BussinessID,attr" json:"bussiness_id"`
	ConfigSoName          string   `xml:"ConfigSoName,attr" json:"nonCloudNative_so_file"`
	ConfigBaseDir         string   `xml:"ConfigBaseDir,attr" json:"conf_ouput_base_dir"`
	HostConfFile          string   `xml:"HostConfFile,attr" json:"host_conf_file"`
	ProcConfFile          string   `xml:"ProcConfFile,attr" json:"proc_conf_file"`
	ProcDeployFile        string   `xml:"ProcDeployFile,attr" json:"proc_deploy_conf_file"`
	BusRelationFile       string   `xml:"BusRelationFile,attr" json:"bus_relation_conf_file"`
	TbusConfOutputDir     string   `xml:"TbusConfOutputDir,attr" json:"tbus_conf_output_dir"`
	ExportDeployOutputDir string   `xml:"ExportDeployOutputDir,attr" json:"deploy_output_dir"`
	CenterdAddr           string   `xml:"CenterdAddr,attr" json:"center_addr"`
	TconndAddr            string   `xml:"TconndAddr,attr" json:"tconnd_addr"`
	HostConfDir           string   `xml:"HostConfDir,attr" json:"host_conf_output_dir"`
	AccessWhiteListFile   string   `xml:"AccessWhiteListFile,attr" json:"acc_whitelist_file"`
	OpenCreateCfgByScript string   `xml:"OpenCreateCfgByScript,attr" json:"open_create_cfg_by_script"`
	CreateCfgScriptFile   string   `xml:"CreateCfgScriptFile,attr" json:"create_cfg_script_file"`
	NonCloudNativeDumpDir string   `xml:"NonCloudNativeDumpDir,attr" json:"nonCloudNative_dump_dir"`
	BinBaseDir            string   `xml:"BinBaseDir,attr" json:"bin_base_dir"`
	WaterLogBaseDir       string   `xml:"WaterLogBaseDir,attr" json:"water_log_base_dir"`
	ToolsSrcBaseDir       string   `xml:"ToolsSrcBaseDir,attr" json:"tool_src_base_dir"`
	ToolsDstBaseDir       string   `xml:"ToolsDstBaseDir,attr" json:"tool_dst_base_dir"`
	TappPidFileDir        string   `xml:"TappPidFileDir,attr" json:"tapp_pid_file_dir"`
	IsUseDBConfig         string   `xml:"IsUseDBConfig,attr" json:"is_use_db_config"`
	ProcStatusBaseDir     string   `xml:"ProcStatusBaseDir,attr" json:"proc_status_base_dir"`
	TransFileType         string   `xml:"TransFileType,attr" json:"trans_file_type"`
	FtpSvrIP              string   `xml:"FtpSvrIp,attr" json:"ftp_svr_ip"`
	FtpSvrPort            string   `xml:"FtpSvrPort,attr" json:"ftp_svr_port"`
	FtpUser               string   `xml:"FtpUser,attr" json:"ftp_user"`
	FtpPasswd             string   `xml:"FtpPasswd,attr" json:"ftp_password"`
	FtpBaseDir            string   `xml:"FtpBaseDir,attr" json:"ftp_base_dir"`
}

func loadCenterConfig(filename string) (nonCloudNativeConf, error) {
	config := new(CenterConf)
	if err := yamlparser.LoadConfig(filename, config); err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

// validate verify that nonCloudNativecenter configuration data is illegal.
func (c *CenterConf) validate() error {
	return nil
}

// XMLExport export nonCloudNative center xml configuration
func (c *CenterConf) XMLExport(outPath string) error {
	outfile := path.Join(outPath, "nonCloudNativecenter.xml")
	output, err := xml.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal(nonCloudNativecenter.xml): %v", err)
	}

	if err := os.WriteFile(outfile, output, 0644); err != nil {
		return fmt.Errorf("write(nonCloudNativecenter.xml): %v", err)
	}

	return nil
}
