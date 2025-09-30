package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

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
	chartPath string
	outPath   string
	valOpts   values.Options
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
	f.StringVarP(&o.outPath, "output", "o", "", "specify templates rendered result save path")
	return cmd
}

func (o *templateOptions) run(out io.Writer) (err error) {
	var (
		valuePaths []string
		optVals    map[string]any
	)

	valuePaths, err = o.valOpts.MergePaths()
	if err != nil {
		return
	}

	optVals, err = o.valOpts.MergeValues()
	if err != nil {
		return
	}

	nonCloudNativeCfg, err := noncloudnative.LoadConfig(valuePaths)
	if err != nil {
		return fmt.Errorf("load noncloudnative configuration: %v", err)
	}

	var optGlobalVals map[string]any
	var ok bool = false
	optGlobalVals, ok = optVals["global"].(map[string]any)
	if ok {
		// 覆盖 WorldId 与 ZoneId
		if w, ok := optGlobalVals["world_id"]; ok {
			var worldId int = 0
			if !reflect.ValueOf(w).CanInt() {
				return fmt.Errorf("wrong type world_id")
			}

			worldId = int(reflect.ValueOf(w).Int())
			nonCloudNativeCfg.Deploy.WorldID = worldId
		}
		if z, ok := optGlobalVals["zone_id"]; ok {
			var zoneId int = 0
			if !reflect.ValueOf(z).CanInt() {
				return fmt.Errorf("wrong type zone_id")
			}

			zoneId = int(reflect.ValueOf(z).Int())
			nonCloudNativeCfg.Deploy.ZoneId = zoneId
		}
	}

	if o.outPath == "" {
		return fmt.Errorf("outPath not found")
	}

	for _, Instance := range nonCloudNativeCfg.Deploy.Instance {
		for i := 0; i < Instance.Num; i++ {
			insID := Instance.StartInsId + i
			addrCom := []string{}
			addrCom = append(addrCom, fmt.Sprint(nonCloudNativeCfg.Deploy.WorldID))
			if Instance.WorldInstance {
				addrCom = append(addrCom, fmt.Sprint(0))
			} else {
				addrCom = append(addrCom, fmt.Sprint(nonCloudNativeCfg.Deploy.ZoneId))
			}
			addrCom = append(addrCom, fmt.Sprint(Instance.TypeId))
			addrCom = append(addrCom, fmt.Sprint(insID))
			busAddr := strings.Join(addrCom, ".")

			copyOptVals := make(map[string]any)
			if val, ok := optVals[Instance.Name]; ok {
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

			copyOptVals["type_id"] = Instance.TypeId

			nonCloudNativeOpt := &noncloudnative.RenderValue{
				BusAddr: busAddr,
				Config:  nonCloudNativeCfg,
			}

			vals, err := util.MergeChartValues(filepath.Join(o.chartPath, Instance.Name), valuePaths, copyOptVals, nonCloudNativeOpt)
			if err != nil {
				return err
			}

			if err := renderTemplate(filepath.Join(o.chartPath, Instance.Name), vals, filepath.Join(o.outPath, Instance.Name)); err != nil {
				return err
			}
			fmt.Fprintf(out, "create('%s', '%s') configuration success\n", Instance.Name, busAddr)
		}
	}

	return nil
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
