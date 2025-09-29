package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/mitchellh/copystructure"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/atframework/atdtool/cli/values"
	"github.com/atframework/atdtool/internal/pkg/noncloudnative"
	"github.com/atframework/atdtool/internal/pkg/util"
	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

const templateDesc = `
Render custom chart templates locally.

To override values in a chart, use either the '--values' flag and pass file
path or use the '--set' flag and pass configuration from the command line.

You can specify the multiple replace paths with '--values'/'-p' flag.
Multiple paths are separated by commas. The priority will be given to the last 
(right-most) path specified.

You can specify the '--set'/'-s' flag multiple times. The priority will be given to the
last (right-most) set specified.
`

type templateOptions struct {
	chartPath      string
	outPath        string
	nonCloudNative bool
	develMode      bool
	valOpts        values.Options
}

func newTemplateCmd(out io.Writer) *cobra.Command {
	o := &templateOptions{}

	cmd := &cobra.Command{
		Use:   "template [CHART]",
		Short: "Render custom chart templates locally",
		Long:  templateDesc,
		Args:  require.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// Allow file completion when completing the argument for the name
				// which could be a path
				return nil, cobra.ShellCompDirectiveDefault
			}
			// No more completions, so disable file completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.chartPath = args[0]
			if err := o.run(out); err != nil {
				return err
			}
			return nil
		},
	}

	if out != nil {
		cmd.SetOut(out)
	}

	f := cmd.Flags()
	addValueOptionsFlags(f, &o.valOpts)
	f.BoolVar(&o.nonCloudNative, "non-cloud-native", false, "enable non-cloud-native mode")
	f.BoolVar(&o.develMode, "devel", false, "enable develop mode")
	f.StringVarP(&o.outPath, "output", "o", "", "specify templates rendered result save path")
	return cmd
}

func (o *templateOptions) run(out io.Writer) (err error) {
	var (
		valuePaths []string
		remoteVals map[string]any
		optVals    map[string]any
		vals       map[string]any
	)

	// TODO: 远程配置中心
	remoteVals = make(map[string]any)

	valuePaths, err = o.valOpts.MergePaths()
	if err != nil {
		return
	}

	optVals, err = o.valOpts.MergeValues()
	if err != nil {
		return
	}

	// devel mode
	if o.develMode {
		err = o.runDevel(valuePaths, remoteVals, optVals, out)
		return
	}

	var nonCloudNativeVal *noncloudnative.RenderValue
	if o.nonCloudNative {
		var nonCloudNativeCfg *noncloudnative.Config
		nonCloudNativeCfg, err = noncloudnative.LoadConfig(valuePaths)
		if err != nil {
			err = fmt.Errorf("load noncloudnative configuration: %v", err)
			return
		}

		// non cloud-native mode should pass proc bus_addr and host value by command line
		m, ok := optVals["non_cloud_native"]
		if !ok {
			err = fmt.Errorf("non cloud-native mode should pass proc bus_addr and hostname value by command line")
			return
		}

		vals, ok := m.(map[string]any)
		if !ok {
			err = fmt.Errorf("non cloud-native mode passed wrong type values by command line")
			return
		}

		busAddr, ok := vals["bus_addr"].(string)
		if !ok {
			err = fmt.Errorf("non cloud-native mode should pass proc bus_addr by command line")
			return
		}

		hostname, ok := vals["hostname"].(string)
		if !ok {
			err = fmt.Errorf("non cloud-native mode should pass proc hostname by command line")
			return
		}

		nonCloudNativeVal = &noncloudnative.RenderValue{
			BusAddr:  busAddr,
			Hostname: hostname,
			Config:   nonCloudNativeCfg,
		}
	}

	vals, err = util.MergeChartValues(o.chartPath, valuePaths, remoteVals, optVals, nonCloudNativeVal)
	if err != nil {
		return
	}
	err = renderTemplate(o.chartPath, vals, o.outPath)
	return
}

func renderTemplate(chartPath string, vals map[string]any, outPath string) error {
	var err error
	var chrt *chart.Chart

	chrt, err = loader.Load(chartPath)
	if err != nil {
		return err
	}

	var suffix string
	if addr, ok := vals["bus_addr"]; ok {
		suffix = fmt.Sprintf("_%s", addr)
	}
	return render(chrt, vals, outPath, suffix)
}

// develop mode, try render all charts by specified path
// TODO non non cloud-native mode, we should query target charts by different way
func (o *templateOptions) runDevel(valuePaths []string, remoteVals, optVals map[string]any, out io.Writer) error {
	if !o.nonCloudNative {
		return fmt.Errorf("devel mode should enable non cloud-native mode")
	}

	nonCloudNativeCfg, err := noncloudnative.LoadConfig(valuePaths)
	if err != nil {
		return fmt.Errorf("load noncloudnative configuration: %v", err)
	}

	var optGlobalVals map[string]any
	var ok bool = false
	optGlobalVals, ok = optVals["global"].(map[string]any)
	if ok {
		// replace work path
		if workPath, ok := optGlobalVals["work_path"].(string); ok {
			nonCloudNativeCfg.Proc.ClusterAttr.WorkPath = workPath
		}

		// replace world id and tbus shm key
		if w, ok := optGlobalVals["world_id"]; ok {
			var worldId int = 0
			if !reflect.ValueOf(w).CanInt() {
				return fmt.Errorf("wrong type world_id")
			}

			worldId = int(reflect.ValueOf(w).Int())
			for _, v := range nonCloudNativeCfg.ProcDeploy.Worlds {
				v.WorldID = worldId
			}

			for _, v := range nonCloudNativeCfg.ProcDeploy.Zones {
				v.WorldID = worldId
			}

			for _, v := range nonCloudNativeCfg.ProcDeploy.Deploys {
				for _, e := range v {
					e.WorldID = worldId
				}
			}

			//replace tbus shm key
			nonCloudNativeCfg.TBus.GCIMShmKey = nonCloudNativeCfg.TBus.GCIMShmKey + worldId
			nonCloudNativeCfg.TBus.GRMShmKey = nonCloudNativeCfg.TBus.GRMShmKey + worldId
		}

		// replace host
		if innerIP, ok := optGlobalVals["inner_ip"].(string); ok {
			for _, v := range nonCloudNativeCfg.Host.Hosts {
				v.InnerIP = innerIP
			}
		}
	}

	configSet := make([]map[string]interface{}, 0)
	for k, v := range nonCloudNativeCfg.ProcDeploy.Deploys {
		for _, e := range v {
			g := nonCloudNativeCfg.Proc.GetProcGroup(k)
			if g == nil {
				return fmt.Errorf("deploy group: %s not exist", k)
			}

			if len(e.Nodes) == 0 {
				return fmt.Errorf("%s has no valid deploy node in world: %d zone: %d", k, e.WorldID, e.ZoneID)
			}

			for _, p := range g.Procs {
				proc := nonCloudNativeCfg.Proc.GetProcNodeByFuncName(p)
				if proc == nil {
					return fmt.Errorf("proc(%s) not found", p)
				}
				for _, u := range e.Nodes {
					for i := 0; i < u.Num; i++ {
						insID := u.StartInsID + i
						addrCom := []string{}
						addrCom = append(addrCom, fmt.Sprint(e.WorldID))
						addrCom = append(addrCom, fmt.Sprint(e.ZoneID))
						addrCom = append(addrCom, fmt.Sprint(proc.FuncID))
						addrCom = append(addrCom, fmt.Sprint(insID))
						busAddr := strings.Join(addrCom, ".")

						copyRemoteVals := make(map[string]any)
						copyOptVals := make(map[string]any)
						nonCloudNativeVals := make(map[string]any)
						copyOptVals["non_cloud_native"] = nonCloudNativeVals
						nonCloudNativeVals["bus_addr"] = busAddr
						nonCloudNativeVals["hostname"] = u.Host

						if val, ok := optVals[p]; ok {
							if vm, ok := val.(map[string]interface{}); ok {
								for k, v := range vm {
									copyVal, err := copystructure.Copy(v)
									if err != nil {
										return err
									}
									copyOptVals[k] = copyVal
								}
							}
						}

						if val, ok := remoteVals[p]; ok {
							if vm, ok := val.(map[string]interface{}); ok {
								for k, v := range vm {
									copyVal, err := copystructure.Copy(v)
									if err != nil {
										return err
									}
									copyRemoteVals[k] = copyVal
								}
							}
						}

						if val, ok := optVals["global"]; ok {
							if vm, ok := val.(map[string]interface{}); ok {
								for k, v := range vm {
									copyVal, err := copystructure.Copy(v)
									if err != nil {
										return err
									}
									copyOptVals[k] = copyVal
								}
							}
						}

						outPath := o.outPath
						if outPath == "" {
							outPath = filepath.Join(nonCloudNativeCfg.Proc.ClusterAttr.WorkPath, filepath.Join(proc.WorkPath, "../"))
						}

						nonCloudNativeOpt := &noncloudnative.RenderValue{
							BusAddr:  busAddr,
							Hostname: u.Host,
							Config:   nonCloudNativeCfg,
						}

						vals, err := util.MergeChartValues(filepath.Join(o.chartPath, p), valuePaths, copyRemoteVals, copyOptVals, nonCloudNativeOpt)
						if err != nil {
							return err
						}

						if err := renderTemplate(filepath.Join(o.chartPath, p), vals, outPath); err != nil {
							return err
						}
						delete(copyOptVals, "non_cloud_native")
						configSet = append(configSet, copyOptVals)
						fmt.Fprintf(out, "create('%s', '%s') configuration success\n", proc.FuncName, busAddr)
					}
				}
			}
		}
	}

	globalVals := make(map[string]any)
	for _, p := range valuePaths {
		// load global replace configuration
		filename := chartutil.GlobalKey + ".yaml"
		if util.FileExist(filepath.Join(p, filename)) {
			m := make(map[string]any)
			if err := yamlparser.LoadConfig(filepath.Join(p, filename), &m); err != nil {
				return err
			}
			globalVals = chartutil.CoalesceTables(m, globalVals)
		}
	}

	if remoteVals != nil {
		globalVals = chartutil.CoalesceTables(remoteVals, globalVals)
	}

	if optGlobalVals != nil {
		globalVals = chartutil.CoalesceTables(optGlobalVals, globalVals)
	}

	if err := renderTbusConf(nonCloudNativeCfg); err != nil {
		return fmt.Errorf("render tbus: %v", err)
	}

	if err := renderScript(nonCloudNativeCfg, globalVals, configSet); err != nil {
		return fmt.Errorf("render script:%v", err)
	}
	return nil
}

// ProcStartSeq represents a proc start sequence
type ProcStartSeq struct {
	Name string
	Seq  int
}

// ProcStartSeqSlice alias for proc start sequence slice
type ProcStartSeqSlice []ProcStartSeq

func (s ProcStartSeqSlice) Len() int      { return len(s) }
func (s ProcStartSeqSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ProcStartSeqSlice) Less(i, j int) bool {
	if s[i].Seq == s[j].Seq {
		return s[i].Name < s[j].Name
	}
	return s[i].Seq < s[j].Seq
}

// it should only be called in develop environment
func parseTbusChannelsData(nonCloudNativeCfg *noncloudnative.Config) ([]map[string]interface{}, error) {
	busChannels := make([]map[string]interface{}, 0)
	for _, v := range nonCloudNativeCfg.TBus.Channels {
		p1 := nonCloudNativeCfg.Proc.GetProcNodeByFuncName(v.ProcSet1)
		if p1 == nil {
			return nil, fmt.Errorf("unable find proc: %s", v.ProcSet1)
		}

		p2 := nonCloudNativeCfg.Proc.GetProcNodeByFuncName(v.ProcSet2)
		if p2 == nil {
			return nil, fmt.Errorf("unable find proc: %s", v.ProcSet2)
		}

		if p1.FuncID&v.MaskArray[2] != p2.FuncID&v.MaskArray[2] {
			continue
		}

		g1 := nonCloudNativeCfg.Proc.GetProcGroupByFuncName(v.ProcSet1)
		if g1 == nil {
			return nil, fmt.Errorf("proc: %s has no valid deploy group", v.ProcSet1)
		}

		g2 := nonCloudNativeCfg.Proc.GetProcGroupByFuncName(v.ProcSet2)
		if g2 == nil {
			return nil, fmt.Errorf("proc: %s has no valid deploy group", v.ProcSet2)
		}

		g1Deploys := nonCloudNativeCfg.ProcDeploy.GetDeployElements(g1.Name)
		g2Deploys := nonCloudNativeCfg.ProcDeploy.GetDeployElements(g2.Name)

		for _, u1 := range g1Deploys {
			for _, u2 := range g2Deploys {
				if u1.WorldID&v.MaskArray[0] != u2.WorldID&v.MaskArray[0] {
					continue
				}

				if u1.ZoneID&v.MaskArray[1] != u2.ZoneID&v.MaskArray[1] {
					continue
				}

				for _, n1 := range u1.Nodes {
					for i := 0; i < n1.Num; i++ {
						inst1 := n1.StartInsID + i
						for _, n2 := range u2.Nodes {
							for j := 0; j < n2.Num; j++ {
								inst2 := n2.StartInsID + j
								if inst1&v.MaskArray[3] == inst2&v.MaskArray[3] {
									channel := make(map[string]interface{})
									channel["proc_set_1_bus_addr"] = fmt.Sprintf("%d.%d.%d.%d", u1.WorldID, u1.ZoneID, p1.FuncID, inst1)
									channel["proc_set_2_bus_addr"] = fmt.Sprintf("%d.%d.%d.%d", u2.WorldID, u2.ZoneID, p2.FuncID, inst2)
									channel["proc_set_1_name"] = v.ProcSet1
									channel["proc_set_2_name"] = v.ProcSet2
									busChannels = append(busChannels, channel)
								}
							}
						}
					}
				}
			}
		}
	}
	return busChannels, nil
}

func renderTbusConf(nonCloudNativeCfg *noncloudnative.Config) error {
	config := make(map[string]interface{})

	config["bus_addr_template"] = nonCloudNativeCfg.TBus.GetBriefBusAddrTemplate()
	config["bus_key"] = nonCloudNativeCfg.TBus.GCIMShmKey
	config["bus_channel_size"] = nonCloudNativeCfg.TBus.ChannelSize

	busChannels, err := parseTbusChannelsData(nonCloudNativeCfg)
	if err != nil {
		return err
	}
	config["bus_channels"] = busChannels

	tplFilePath := path.Join(nonCloudNativeCfg.Proc.ClusterAttr.WorkPath, "tbus/cfg")
	tplFiles, err := filepath.Glob(path.Join(tplFilePath, "*.template"))
	if err != nil {
		return err
	}

	for _, tplFile := range tplFiles {
		tpl := template.New(path.Base(tplFile)).Funcs(sprig.TxtFuncMap())
		tpl = tpl.Option("missingkey=error")
		tpl = template.Must(tpl.ParseFiles(tplFile))

		suffix := path.Ext(path.Base(tplFile))
		filename := strings.TrimSuffix(path.Base(tplFile), suffix)
		outFile := path.Join(tplFilePath, filename)

		f, err := os.Create(outFile)
		if err != nil {
			return err
		}

		if err := tpl.Execute(f, config); err != nil {
			return err
		}
	}
	return nil
}

func renderScript(nonCloudNativeCfg *noncloudnative.Config, globalVals map[string]interface{}, elements []map[string]interface{}) error {
	procStartSeqSet := make(map[string]int)
	for _, m := range elements {
		if n, ok := m["type_name"].(string); ok {
			if _, ok := procStartSeqSet[n]; !ok {
				proc := nonCloudNativeCfg.Proc.GetProcNodeByFuncName(n)
				if proc != nil {
					procStartSeqSet[n] = proc.Seq
				}
			}
		}
	}

	vals := make(map[string]interface{})
	vals["global"] = globalVals
	vals["elements"] = elements

	// ordered proc start sequence
	sortedStartSeqs := ProcStartSeqSlice{}
	for procName, seq := range procStartSeqSet {
		seq := ProcStartSeq{
			Name: procName,
			Seq:  seq,
		}
		sortedStartSeqs = append(sortedStartSeqs, seq)
	}

	sort.Stable(sortedStartSeqs)
	vals["sorted_proc_seq"] = sortedStartSeqs

	scriptPath, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), "../script"))
	if err != nil {
		return err
	}

	tplFiles := make([]string, 0)
	wakErr := filepath.WalkDir(scriptPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("failure accessing a path %q: %v\n", path, err)
			return err
		}

		if d.IsDir() {
			t, tplErr := filepath.Glob(filepath.Join(path, "*.template"))
			if tplErr != nil {
				return tplErr
			}
			tplFiles = append(tplFiles, t...)

			t, tplErr = filepath.Glob(filepath.Join(path, "*.tpl"))
			if tplErr != nil {
				return tplErr
			}
			tplFiles = append(tplFiles, t...)
			return nil
		}
		return nil
	})

	if wakErr != nil {
		fmt.Printf("error walking script path %v\n", wakErr)
		return fmt.Errorf("walking the script path: %v", wakErr)
	}

	for _, tplFile := range tplFiles {
		tpl := template.New(path.Base(tplFile)).Funcs(sprig.TxtFuncMap())
		tpl = tpl.Option("missingkey=error")
		tpl = template.Must(tpl.ParseFiles(tplFile))

		suffix := path.Ext(path.Base(tplFile))
		filename := strings.TrimSuffix(path.Base(tplFile), suffix)
		outFile := path.Join(nonCloudNativeCfg.Proc.ClusterAttr.WorkPath, filename)

		f, err := os.Create(outFile)
		if err != nil {
			return err
		}

		if err := os.Chmod(outFile, 0755); err != nil {
			return err
		}

		if err := tpl.Execute(f, vals); err != nil {
			return err
		}
	}
	return nil
}

func allConfigTemplates(chrt *chart.Chart) {
	chrt.Templates = chrt.Templates[:0]
	for _, f := range chrt.Files {
		if strings.HasSuffix(f.Name, ".tpl") || strings.HasSuffix(f.Name, "*.template") {
			chrt.Templates = append(chrt.Templates, f)
		}
	}
}

// render generate service configuration file in chart.
func render(chrt *chart.Chart, vals chartutil.Values, outPath, outSuffix string) error {
	if err := chartutil.ProcessDependencies(chrt, vals); err != nil {
		return err
	}

	top := make(map[string]interface{})
	top["Values"] = vals
	en := &engine.Engine{
		LintMode: false,
	}

	allConfigTemplates(chrt)
	output, err := en.Render(chrt, top)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var cfgOutPath string
	for k, v := range output {
		// no output path specified, use standard output
		if outPath == "" {
			fmt.Println("---")
			fmt.Printf("# Source: %s\n", k)
			fmt.Println(v)
			continue
		}

		suffix := filepath.Ext(path.Base(k))
		if suffix != ".tpl" {
			continue
		}

		relPath := strings.TrimPrefix(filepath.Dir(k), chrt.Name())
		cfgOutPath = filepath.Join(outPath, relPath)
		if !util.PathExist(cfgOutPath) {
			if err := os.MkdirAll(cfgOutPath, os.ModePerm); err != nil {
				return fmt.Errorf("make configuration output path(%s): %v", cfgOutPath, err)
			}
		}

		filename := strings.TrimSuffix(path.Base(k), suffix)
		var outFile string
		if outSuffix != "" {
			outFile = path.Join(cfgOutPath, filename+outSuffix)
		} else {
			outFile = path.Join(cfgOutPath, filename)
		}

		f, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("create configuration file(%s): %v", outFile, err)
		}

		if _, err := f.WriteString(v); err != nil {
			return fmt.Errorf("write config file(%s): %v", outFile, err)
		}
	}
	return nil
}
