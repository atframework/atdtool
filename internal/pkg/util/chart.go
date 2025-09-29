package util

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/atframework/atdtool/internal/pkg/noncloudnative"
	yamlparser "github.com/atframework/atdtool/pkg/confparser/yaml"
)

// MergeChartValues merges multiple sources of Helm chart values into a single values map
func MergeChartValues(chartPath string, valuesPaths []string, remoteVals, optVals map[string]any, nonCloudNativeVal *noncloudnative.RenderValue) (values map[string]any, err error) {
	var chrt *chart.Chart
	chrt, err = loader.Load(chartPath)
	if err != nil {
		return
	}

	name := chrt.Name()
	if n, ok := chrt.Values["type_name"]; ok {
		name = n.(string)
	} else if n, ok := chrt.Values["func_name"]; ok {
		name = n.(string)
	}

	values = make(map[string]any)
	globalVals := make(map[string]any)
	for _, p := range valuesPaths {
		// load global replace configuration
		filename := chartutil.GlobalKey + ".yaml"
		if FileExist(filepath.Join(p, filename)) {
			m := make(map[string]any)
			err = yamlparser.LoadConfig(filepath.Join(p, filename), &m)
			if err != nil {
				return
			}
			globalVals = chartutil.CoalesceTables(m, globalVals)
		}

		// load service replace configuration
		filename = name + ".yaml"
		if FileExist(filepath.Join(p, filename)) {
			m := make(map[string]any)
			err = yamlparser.LoadConfig(filepath.Join(p, filename), &m)
			if err != nil {
				return
			}
			values = chartutil.CoalesceTables(m, values)
		}
	}

	values = chartutil.CoalesceTables(values, chrt.Values)
	values = chartutil.CoalesceTables(values, globalVals)

	if nonCloudNativeVal != nil {
		if nonCloudNativeVal.Config == nil {
			err = fmt.Errorf("nil nonCloudNative configuration")
			return
		}

		var m map[string]any
		m, err = nonCloudNativeVal.Config.ToRenderValues(nonCloudNativeVal.BusAddr, nonCloudNativeVal.Hostname)
		if err != nil {
			return
		}
		values = chartutil.CoalesceTables(m, values)
	}

	if remoteVals != nil {
		// merge remote global config
		if v, ok := remoteVals["global"]; ok {
			if !reflect.ValueOf(v).CanConvert(reflect.TypeOf(map[string]any{})) {
				err = fmt.Errorf("can not convert to map")
				return
			}
			m := reflect.ValueOf(v).Convert(reflect.TypeOf(map[string]any{})).Interface().(map[string]any)
			values = chartutil.CoalesceTables(m, values)
		}

		// merge remote server config
		if v, ok := remoteVals[name]; ok {
			if !reflect.ValueOf(v).CanConvert(reflect.TypeOf(map[string]any{})) {
				err = fmt.Errorf("can not convert to map")
				return
			}
			m := reflect.ValueOf(v).Convert(reflect.TypeOf(map[string]any{})).Interface().(map[string]any)
			values = chartutil.CoalesceTables(m, values)
		}
	}

	// command line options has higher precedence
	if optVals != nil {
		values = chartutil.CoalesceTables(optVals, values)
	}

	values, err = mergeEnabledModuleValues(valuesPaths, values)
	return
}

// merge enabled module values
func mergeEnabledModuleValues(valuesPaths []string, dst map[string]any) (map[string]any, error) {
	moduleVals := make(map[string]any)
	for _, p := range valuesPaths {
		modulesPath := filepath.Join(p, "modules")
		if PathExist(modulesPath) {
			walkErr := filepath.WalkDir(modulesPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					fmt.Printf("failure accessing a path %q: %v\n", path, err)
					return err
				}
				if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
					return nil
				}

				moduleName := strings.TrimSuffix(d.Name(), ".yaml")
				val := make(map[string]any)
				if err := yamlparser.LoadConfig(path, &val); err != nil {
					return err
				}

				m := make(map[string]any)
				m[moduleName] = val
				moduleVals = chartutil.CoalesceTables(m, moduleVals)
				return nil
			})
			if walkErr != nil {
				return nil, fmt.Errorf("walking the path(%q):%v", modulesPath, walkErr)
			}
		}
	}

	for k, v := range moduleVals {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid module(%s) value type", k)
		}

		moduleEnabled := false
		if val, ok := dst[k].(map[string]any); ok {
			if flag, ok := val["enabled"].(bool); ok {
				// module disabled
				if !flag {
					continue
				}
				moduleEnabled = true
			}
		}

		// the module enable flag is not specified,
		// if it enabled by default, we will still load it
		if !moduleEnabled {
			if flag, ok := m["enabled"].(bool); ok {
				moduleEnabled = flag
			}
		}

		if !moduleEnabled {
			delete(moduleVals, k)
		}
	}

	dst = chartutil.CoalesceTables(dst, moduleVals)
	return dst, nil
}
