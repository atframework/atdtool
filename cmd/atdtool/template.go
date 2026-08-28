package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
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

Use '--mode deploy_script' with '--scripts' to render selected deployment script
templates from a chart using grouped instances from deploy.yaml.
`

type templateOptions struct {
	chartPath string
	outPath   string
	scripts   []string
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
	f.StringSliceVar(&o.scripts, "scripts", []string{}, "set the deploy script templates to render in deploy_script mode (paths relative to CHART, can specify multiple paths with commas:path1,path2)")
	return cmd
}

func (o *templateOptions) runNonCloudNative(out io.Writer, valuePaths []string, optVals map[string]any) (err error) {
	nonCloudNativeCfg, err := noncloudnative.LoadConfig(valuePaths)
	if err != nil {
		return fmt.Errorf("load noncloudnative configuration: %v", err)
	}

	if err := applyGlobalIDOverrides(optVals, nonCloudNativeCfg.Deploy); err != nil {
		return err
	}

	if o.outPath == "" {
		return fmt.Errorf("outPath not found")
	}

	for _, Instance := range nonCloudNativeCfg.Deploy.Instance {
		instanceTypeId, err := loadChartTypeId(filepath.Join(o.chartPath, Instance.Name))
		if err != nil {
			return err
		}

		nonCloudNativeOpt := &noncloudnative.RenderValue{
			Config: nonCloudNativeCfg,
			TypeId: instanceTypeId,
		}
		for _, inst := range Instance.ExpandInstances(nonCloudNativeCfg.Deploy.WorldID, nonCloudNativeCfg.Deploy.ZoneId, instanceTypeId) {
			nonCloudNativeOpt.BusAddr = inst.BusAddr

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

			vals, err := util.MergeChartValues(filepath.Join(o.chartPath, Instance.Name), valuePaths, copyOptVals, nonCloudNativeOpt)
			if err != nil {
				return err
			}

			if err := renderTemplate(filepath.Join(o.chartPath, Instance.Name), vals, filepath.Join(o.outPath, Instance.Name), nil); err != nil {
				return err
			}
			fmt.Fprintf(out, "create('%s', '%s') configuration success\n", Instance.Name, inst.BusAddr)
		}
	}
	return nil
}

// loadChartTypeId loads the chart and reads the instance type id from the
// chart default values.
func loadChartTypeId(chartPath string) (uint64, error) {
	chrt, err := loader.Load(chartPath)
	if err != nil {
		return 0, fmt.Errorf("load chart(%s): %v", filepath.Base(chartPath), err)
	}

	rawTypeId, ok := chrt.Values["type_id"]
	if !ok {
		return 0, fmt.Errorf("chart(%s) not found type_id", filepath.Base(chartPath))
	}

	typeId, err := convertToUint64Opt("type_id", rawTypeId)
	if err != nil {
		return 0, fmt.Errorf("chart(%s) has invalid type_id: %v", filepath.Base(chartPath), err)
	}
	return typeId, nil
}

// applyGlobalIDOverrides overrides WorldID and ZoneId of the deploy
// configuration with `--set global.world_id` and `--set global.zone_id`.
// The global options are also normalized to uint64 so later merged values
// keep the same type as the deploy configuration.
func applyGlobalIDOverrides(optVals map[string]any, deploy *noncloudnative.DeployConf) error {
	optGlobalVals, ok := optVals["global"].(map[string]any)
	if !ok {
		return nil
	}

	if w, ok := optGlobalVals["world_id"]; ok {
		worldId, err := convertToUint64Opt("world_id", w)
		if err != nil {
			return err
		}
		deploy.WorldID = worldId
		optGlobalVals["world_id"] = worldId
	}
	if z, ok := optGlobalVals["zone_id"]; ok {
		zoneId, err := convertToUint64Opt("zone_id", z)
		if err != nil {
			return err
		}
		deploy.ZoneId = zoneId
		optGlobalVals["zone_id"] = zoneId
	}
	return nil
}

func (o *templateOptions) runNonDeploy(out io.Writer, valuePaths []string, optVals map[string]any) (err error) {
	copyOptVals := make(map[string]any)
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

	vals, err := util.MergeChartValues(filepath.Join(o.chartPath), valuePaths, copyOptVals, nil)
	if err != nil {
		return err
	}

	if err := renderTemplate(filepath.Join(o.chartPath), vals, filepath.Join(o.outPath), nil); err != nil {
		return err
	}
	fmt.Fprintf(out, "create('%s', '%s') configuration success\n", o.chartPath, "")
	return nil
}

func (o *templateOptions) runDeployScript(out io.Writer, valuePaths []string, optVals map[string]any) (err error) {
	if len(o.scripts) == 0 {
		return fmt.Errorf("mode deploy_script requires --scripts to specify the script templates to render")
	}

	nonCloudNativeCfg, err := noncloudnative.LoadConfig(valuePaths)
	if err != nil {
		return fmt.Errorf("load noncloudnative configuration: %v", err)
	}

	if err := applyGlobalIDOverrides(optVals, nonCloudNativeCfg.Deploy); err != nil {
		return err
	}

	if o.outPath == "" {
		return fmt.Errorf("outPath not found")
	}

	scriptChart, wantTemplates, err := resolveScriptTemplates(o.chartPath, o.scripts)
	if err != nil {
		return err
	}

	// expand deploy units, keep the deploy.yaml declaration order
	procs := make([]noncloudnative.ScriptProc, 0, len(nonCloudNativeCfg.Deploy.Instance))
	for _, Instance := range nonCloudNativeCfg.Deploy.Instance {
		instanceTypeId, err := loadChartTypeId(filepath.Join(o.chartPath, Instance.Name))
		if err != nil {
			return err
		}
		procs = append(procs, noncloudnative.ScriptProc{
			Name:      Instance.Name,
			Group:     Instance.Group,
			Instances: Instance.ExpandInstances(nonCloudNativeCfg.Deploy.WorldID, nonCloudNativeCfg.Deploy.ZoneId, instanceTypeId),
		})
	}
	procGroups := noncloudnative.BuildScriptProcGroups(procs)

	copyOptVals := make(map[string]any)
	if val, ok := optVals[scriptChart]; ok {
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

	vals, err := util.MergeChartValues(filepath.Join(o.chartPath, scriptChart), valuePaths, copyOptVals, nil)
	if err != nil {
		return err
	}

	// runtime values of the deploy configuration win over chart defaults
	vals["world_id"] = nonCloudNativeCfg.Deploy.WorldID
	vals["zone_id"] = nonCloudNativeCfg.Deploy.ZoneId
	vals["proc_groups"] = procGroupsToRenderValues(procGroups)

	chrt, err := loader.Load(filepath.Join(o.chartPath, scriptChart))
	if err != nil {
		return fmt.Errorf("load chart(%s): %v", scriptChart, err)
	}

	available := make(map[string]bool, len(chrt.Templates)+len(chrt.Files))
	for _, f := range chrt.Templates {
		available[f.Name] = true
	}
	for _, f := range chrt.Files {
		available[f.Name] = true
	}
	for _, script := range o.scripts {
		_, rel, _ := strings.Cut(filepath.ToSlash(filepath.Clean(script)), "/")
		if !available[rel] {
			return fmt.Errorf("script template(%s) not found in chart(%s)", rel, scriptChart)
		}
	}

	if err := render(chrt, vals, o.outPath, "", wantTemplates); err != nil {
		return err
	}
	for _, script := range o.scripts {
		fmt.Fprintf(out, "create('%s') deploy script success\n", script)
	}
	return nil
}

// resolveScriptTemplates validates the --scripts options. All script templates
// must belong to the same chart directory below CHART. It returns the chart
// name and the set of template names (relative to the chart root) to render.
func resolveScriptTemplates(chartPath string, scripts []string) (string, map[string]bool, error) {
	const templatesDir = "templates/"

	wantTemplates := make(map[string]bool, len(scripts))
	scriptChart := ""
	for _, script := range scripts {
		rawName := filepath.ToSlash(script)
		if filepath.IsAbs(script) || strings.HasPrefix(rawName, "/") || strings.Contains(rawName, "..") {
			return "", nil, fmt.Errorf("script template(%s) must be a relative path under CHART", script)
		}

		name := filepath.ToSlash(filepath.Clean(script))
		chartName, rel, found := strings.Cut(name, "/")
		if !found || rel == "" {
			return "", nil, fmt.Errorf("script template(%s) must be a path like <chart>/<template>.tpl", script)
		}
		if scriptChart == "" {
			scriptChart = chartName
		} else if scriptChart != chartName {
			return "", nil, fmt.Errorf("all script templates must belong to the same chart(%s), but %s is in chart(%s)", scriptChart, script, chartName)
		}

		wantTemplates[rel] = true
		// tolerate templates written in the standard helm templates directory
		wantTemplates[strings.TrimPrefix(rel, templatesDir)] = true
		wantTemplates[templatesDir+rel] = true
	}
	return scriptChart, wantTemplates, nil
}

// procGroupsToRenderValues converts the grouped procs into lowercase map
// values so templates can access them like `.Values.proc_groups`.
func procGroupsToRenderValues(groups []noncloudnative.ScriptProcGroup) []map[string]any {
	result := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		procs := make([]map[string]any, 0, len(group.Procs))
		for _, proc := range group.Procs {
			instances := make([]map[string]any, 0, len(proc.Instances))
			for _, instance := range proc.Instances {
				instances = append(instances, map[string]any{
					"instance_id": instance.InstanceID,
					"bus_addr":    instance.BusAddr,
				})
			}
			procs = append(procs, map[string]any{
				"name":      proc.Name,
				"instances": instances,
			})
		}
		result = append(result, map[string]any{
			"group": group.Group,
			"procs": procs,
		})
	}
	return result
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

	if o.valOpts.Mode == "" || o.valOpts.Mode == "noncloudnative" {
		return o.runNonCloudNative(out, valuePaths, optVals)
	} else if o.valOpts.Mode == "nondeploy" {
		return o.runNonDeploy(out, valuePaths, optVals)
	} else if o.valOpts.Mode == "deploy_script" {
		return o.runDeployScript(out, valuePaths, optVals)
	}
	return fmt.Errorf("unsupported mode: %s, supported modes are 'noncloudnative', 'nondeploy' and 'deploy_script'", o.valOpts.Mode)
}

func renderTemplate(chartPath string, vals map[string]any, outPath string, onlyTemplates map[string]bool) error {
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
	return render(chrt, vals, outPath, suffix, onlyTemplates)
}

func convertToUint64Opt(name string, input any) (uint64, error) {
	rv := reflect.ValueOf(input)
	if rv.CanUint() {
		return rv.Uint(), nil
	}

	if rv.CanInt() {
		return uint64(rv.Int()), nil
	}

	// helm loads values.yaml through json, so integer numbers are float64
	switch rv.Kind() {
	case reflect.Float64, reflect.Float32:
		floatValue := rv.Float()
		if floatValue == math.Trunc(floatValue) {
			return uint64(floatValue), nil
		}
		return 0, fmt.Errorf("wrong type %s: %s can not convert to uint64", name, reflect.TypeOf(input).Name())
	}

	if reflect.TypeOf(input).Kind() == reflect.String {
		stringValue := strings.TrimSpace(rv.String())
		stringValue = strings.Trim(stringValue, "\"")
		stringValue = strings.Trim(stringValue, "'")

		ret, err := strconv.ParseUint(stringValue, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("wrong type %s: %s can not convert to uint64", name, reflect.TypeOf(input).Name())
		}

		return ret, nil
	}

	return 0, fmt.Errorf("wrong type %s: %s can not convert to uint64", name, reflect.TypeOf(input).Name())
}

func allConfigTemplates(chrt *chart.Chart, onlyTemplates map[string]bool) {
	selected := make([]*chart.File, 0, len(chrt.Templates))
	if onlyTemplates == nil {
		for _, f := range chrt.Files {
			if strings.HasSuffix(f.Name, ".tpl") || strings.HasSuffix(f.Name, ".template") {
				selected = append(selected, f)
			}
		}
	} else {
		// deploy_script mode only renders the templates specified by --scripts
		for _, f := range chrt.Files {
			if onlyTemplates[f.Name] {
				selected = append(selected, f)
			}
		}
		for _, f := range chrt.Templates {
			if onlyTemplates[f.Name] {
				selected = append(selected, f)
			}
		}
	}
	chrt.Templates = selected
}

// render generate service configuration file in chart.
func render(chrt *chart.Chart, vals chartutil.Values, outPath, outSuffix string, onlyTemplates map[string]bool) error {
	if err := chartutil.ProcessDependencies(chrt, vals); err != nil {
		return err
	}

	top := make(map[string]interface{})
	top["Values"] = vals
	en := &engine.Engine{
		LintMode: false,
	}

	allConfigTemplates(chrt, onlyTemplates)
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

		if outSuffix != "" {
			idx := strings.LastIndex(filename, ".")
			if idx != -1 {
				// 存在. 分割
				left := filename[:idx]
				right := filename[idx:]
				filename = left + outSuffix + right
			} else {
				// 直接拼接
				filename = filename + outSuffix
			}
		}

		outFile := path.Join(cfgOutPath, filename)

		f, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("create configuration file(%s): %v", outFile, err)
		}

		if _, err := f.WriteString(v); err != nil {
			_ = f.Close()
			return fmt.Errorf("write config file(%s): %v", outFile, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close config file(%s): %v", outFile, err)
		}
	}
	return nil
}
