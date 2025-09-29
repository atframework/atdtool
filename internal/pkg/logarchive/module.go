package logarchive

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Module is a type that contains a group of logarchive module.
type Module interface {
	ArchiveModule() ModuleInfo
}

// ModuleInfo represents a registered logarchive module.
type ModuleInfo struct {
	ID  ModuleID
	New func() Module
}

type ModuleMap map[string]json.RawMessage

type ModuleID string

// Namespace returns the namespace of a module ID.
func (id ModuleID) Namespace() string {
	lastDot := strings.LastIndex(string(id), ".")
	if lastDot < 0 {
		return ""
	}
	return string(id)[:lastDot]
}

// Name returns the Name of a module ID.
func (id ModuleID) Name() string {
	if id == "" {
		return ""
	}
	parts := strings.Split(string(id), ".")
	return parts[len(parts)-1]
}

// String implements stringer interface
func (m ModuleInfo) String() string { return string(m.ID) }

// RegisterModule registers an logarchive module.
func RegisterModule(instance Module) {
	mod := instance.ArchiveModule()

	if mod.ID == "" {
		panic("module ID missing")
	}

	if mod.New == nil {
		panic("missing ModuleInfo.New")
	}
	if val := mod.New(); val == nil {
		panic("ModuleInfo.New must return a non-nil module instance")
	}

	if _, ok := modules[string(mod.ID)]; ok {
		panic(fmt.Sprintf("module already registered: %s", mod.ID))
	}
	modules[string(mod.ID)] = mod
}

// Provisioner is implemented by module which may need to perform
// some additional "setup" steps immediately after being loaded.
type Provisioner interface {
	Provision(Context) error
}

// Validator is implemented by module which can verify that their
// configurations are valid.
type Validator interface {
	Validate() error
}

// CleanerUpper is implemented by module which can cleanup resources.
type CleanerUpper interface {
	Cleanup() error
}

// ParseStructTag parses a logarchive struct tag into its keys and values.
func ParseStructTag(tag string) (map[string]string, error) {
	results := make(map[string]string)
	pairs := strings.Split(tag, " ")
	for i, pair := range pairs {
		if pair == "" {
			continue
		}
		before, after, isCut := strings.Cut(pair, "=")
		if !isCut {
			return nil, fmt.Errorf("missing key in '%s' (pair %d)", pair, i)
		}
		results[before] = after
	}
	return results, nil
}

func isJSONRawMessage(typ reflect.Type) bool {
	return typ.PkgPath() == "encoding/json" && typ.Name() == "RawMessage"
}

func isModuleMapType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Map &&
		typ.Key().Kind() == reflect.String &&
		isJSONRawMessage(typ.Elem())
}

var (
	modules = make(map[string]ModuleInfo)
)
