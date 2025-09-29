package values

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/strvals"

	"github.com/atframework/atdtool/internal/pkg/util"
)

type Options struct {
	Values []string
	Paths  []string
}

func (opts *Options) MergeValues() (map[string]interface{}, error) {
	base := make(map[string]interface{})
	// User specified a value via --set
	for _, value := range opts.Values {
		if err := strvals.ParseInto(value, base); err != nil {
			return nil, fmt.Errorf("failed parsing --set data: %v", err)
		}
	}
	return base, nil
}

func (opts *Options) MergePaths() ([]string, error) {
	paths := make([]string, 0)
	var lastError error
	for _, v := range opts.Paths {
		var path string = v
		var err error

		if !filepath.IsAbs(path) {
			if strings.HasPrefix(path, "~") {
				path = strings.TrimPrefix(path, "~")
				var home string
				home, err = os.UserHomeDir()
				if err != nil {
					lastError = err
					continue
				}
				path = filepath.Join(home, path)
			} else {
				path, err = filepath.Abs(v)
				if err != nil {
					lastError = err
					continue
				}
			}
		}

		if !util.PathExist(path) {
			lastError = fmt.Errorf("target path(%s) is not exist", path)
			continue
		}
		paths = append(paths, path)
	}
	return paths, lastError
}
