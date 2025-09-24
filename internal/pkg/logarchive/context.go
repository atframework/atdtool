package logarchive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"go.uber.org/zap"
)

// Context is a type which defines the lifetime of file archive that are loaded.
type Context struct {
	context.Context

	cfg *Config

	moduleInstances map[string][]Module
}

// NewContext provides a new context derived from the given context.
func NewContext(ctx Context) (Context, context.CancelFunc) {
	newCtx := Context{
		cfg:             ctx.cfg,
		moduleInstances: make(map[string][]Module),
	}

	stdCtx, cancel := context.WithCancel(ctx.Context)
	wrappedCancel := func() {
		cancel()

		for name, modInstances := range newCtx.moduleInstances {
			for _, inst := range modInstances {
				if cu, ok := inst.(CleanerUpper); ok {
					err := cu.Cleanup()
					if err != nil {
						log.Printf("[ERROR] %s (%p): cleanup: %v", name, inst, err)
					}
				}
			}
		}
	}

	newCtx.Context = stdCtx
	return newCtx, wrappedCancel
}

// LoadModule loads and initializes a module from a struct field
func (ctx Context) LoadModule(structPointer any, fieldName string) (any, error) {
	val := reflect.ValueOf(structPointer).Elem().FieldByName(fieldName)
	typ := val.Type()

	field, ok := reflect.TypeOf(structPointer).Elem().FieldByName(fieldName)
	if !ok {
		panic(fmt.Sprintf("field %s does not exist in %#v", fieldName, structPointer))
	}

	opts, err := ParseStructTag(field.Tag.Get("logarchive"))
	if err != nil {
		panic(fmt.Sprintf("malformed tag on field %s: %v", fieldName, err))
	}

	moduleNamespace, ok := opts["namespace"]
	if !ok {
		panic(fmt.Sprintf("missing 'namespace' key in struct tag on field %s", fieldName))
	}
	inlineModuleKey := opts["inline_key"]

	var result any

	switch val.Kind() {
	case reflect.Slice:
		if isJSONRawMessage(typ) {
			// val is `json.RawMessage` ([]uint8 under the hood)

			if inlineModuleKey == "" {
				panic("unable to determine module name without inline_key when type is not a ModuleMap")
			}
			val, err := ctx.loadModuleInline(inlineModuleKey, moduleNamespace, val.Interface().(json.RawMessage))
			if err != nil {
				return nil, err
			}
			result = val
		} else if isJSONRawMessage(typ.Elem()) {
			// val is `[]json.RawMessage`

			if inlineModuleKey == "" {
				panic("unable to determine module name without inline_key because type is not a ModuleMap")
			}
			var all []any
			for i := 0; i < val.Len(); i++ {
				val, err := ctx.loadModuleInline(inlineModuleKey, moduleNamespace, val.Index(i).Interface().(json.RawMessage))
				if err != nil {
					return nil, fmt.Errorf("position %d: %v", i, err)
				}
				all = append(all, val)
			}
			result = all
		} else if typ.Elem().Kind() == reflect.Slice && isJSONRawMessage(typ.Elem().Elem()) {
			// val is `[][]json.RawMessage`

			if inlineModuleKey == "" {
				panic("unable to determine module name without inline_key because type is not a ModuleMap")
			}
			var all [][]any
			for i := 0; i < val.Len(); i++ {
				innerVal := val.Index(i)
				var allInner []any
				for j := 0; j < innerVal.Len(); j++ {
					innerInnerVal, err := ctx.loadModuleInline(inlineModuleKey, moduleNamespace, innerVal.Index(j).Interface().(json.RawMessage))
					if err != nil {
						return nil, fmt.Errorf("position %d: %v", j, err)
					}
					allInner = append(allInner, innerInnerVal)
				}
				all = append(all, allInner)
			}
			result = all
		} else if isModuleMapType(typ.Elem()) {
			// val is `[]map[string]json.RawMessage`

			var all []map[string]any
			for i := 0; i < val.Len(); i++ {
				thisSet, err := ctx.loadModulesFromSomeMap(moduleNamespace, inlineModuleKey, val.Index(i))
				if err != nil {
					return nil, err
				}
				all = append(all, thisSet)
			}
			result = all
		}

	case reflect.Map:
		// val is a ModuleMap or some other kind of map
		result, err = ctx.loadModulesFromSomeMap(moduleNamespace, inlineModuleKey, val)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unrecognized type for module: %s", typ)
	}

	val.Set(reflect.Zero(typ))
	return result, nil
}

// LoadModuleByID decodes raw config into a new instance.
func (ctx Context) LoadModuleByID(id string, raw json.RawMessage) (any, error) {
	info, ok := modules[id]
	if !ok {
		return nil, fmt.Errorf("unknown module: %s", id)
	}

	if info.New == nil {
		return nil, fmt.Errorf("module '%s' has no constructor", info)
	}

	val := info.New()

	if rv := reflect.ValueOf(val); rv.Kind() != reflect.Ptr {
		log.Printf("[WARNING] ModuleInfo.New() for module '%s' did not return a pointer,"+
			" using new(Type) or &Type notation in your module's New() function.", id)
		val = reflect.New(rv.Type()).Elem().Addr().Interface().(Module)
	}

	// fill in its config only if there is a config to fill in
	if len(raw) > 0 {
		err := json.Unmarshal(raw, &val)
		if err != nil {
			return nil, fmt.Errorf("decoding module config: %s: %v", info, err)
		}
	}

	if val == nil {
		return nil, errors.New("module value cannot be null")
	}

	if prov, ok := val.(Provisioner); ok {
		err := prov.Provision(ctx)
		if err != nil {
			if cleanerUpper, ok := val.(CleanerUpper); ok {
				err2 := cleanerUpper.Cleanup()
				if err2 != nil {
					err = fmt.Errorf("%v; additionally, cleanup: %v", err, err2)
				}
			}
			return nil, fmt.Errorf("provision %s: %v", info, err)
		}
	}

	if validator, ok := val.(Validator); ok {
		err := validator.Validate()
		if err != nil {
			// since the module was already provisioned, make sure we cleanup
			if cleanerUpper, ok := val.(CleanerUpper); ok {
				err2 := cleanerUpper.Cleanup()
				if err2 != nil {
					err = fmt.Errorf("%v; additionally, cleanup: %v", err, err2)
				}
			}
			return nil, fmt.Errorf("%s: invalid configuration: %v", info, err)
		}
	}
	ctx.moduleInstances[id] = append(ctx.moduleInstances[id], val)
	return val, nil
}

func (ctx Context) loadModuleInline(moduleNameKey, moduleNamespace string, raw json.RawMessage) (any, error) {
	moduleName, raw, err := getModuleName(moduleNameKey, raw)
	if err != nil {
		return nil, err
	}

	val, err := ctx.LoadModuleByID(moduleNamespace+"."+moduleName, raw)
	if err != nil {
		return nil, fmt.Errorf("loading module '%s': %v", moduleName, err)
	}

	return val, nil
}

func (ctx Context) loadModulesFromSomeMap(namespace, inlineModuleKey string, val reflect.Value) (map[string]any, error) {
	// if no inline_key is specified, then val must be a ModuleMap,
	// where the key is the module name
	if inlineModuleKey == "" {
		if !isModuleMapType(val.Type()) {
			panic(fmt.Sprintf("expected ModuleMap because inline_key is empty; but we do not recognize this type: %s", val.Type()))
		}
		return ctx.loadModuleMap(namespace, val)
	}

	// otherwise, val is a map with modules, but the module name is
	// inline with each value (the key means something else)
	return ctx.loadModulesFromRegularMap(namespace, inlineModuleKey, val)
}

func (ctx Context) loadModulesFromRegularMap(namespace, inlineModuleKey string, val reflect.Value) (map[string]any, error) {
	mods := make(map[string]any)
	iter := val.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()
		mod, err := ctx.loadModuleInline(inlineModuleKey, namespace, v.Interface().(json.RawMessage))
		if err != nil {
			return nil, fmt.Errorf("key %s: %v", k, err)
		}
		mods[k.String()] = mod
	}
	return mods, nil
}

func (ctx Context) loadModuleMap(namespace string, val reflect.Value) (map[string]any, error) {
	all := make(map[string]any)
	iter := val.MapRange()
	for iter.Next() {
		k := iter.Key().Interface().(string)
		v := iter.Value().Interface().(json.RawMessage)
		moduleName := namespace + "." + k
		if namespace == "" {
			moduleName = k
		}
		val, err := ctx.LoadModuleByID(moduleName, v)
		if err != nil {
			return nil, fmt.Errorf("module name '%s': %v", k, err)
		}
		all[k] = val
	}
	return all, nil
}

func getModuleName(moduleNameKey string, raw json.RawMessage) (string, json.RawMessage, error) {
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return "", nil, err
	}

	moduleName, ok := m[moduleNameKey].(string)
	if !ok || moduleName == "" {
		return "", nil, fmt.Errorf("module name not specified with key '%s' in %+v", moduleNameKey, m)
	}

	delete(m, moduleNameKey)
	result, err := json.Marshal(m)
	if err != nil {
		return "", nil, fmt.Errorf("encoding module configuration: %v", err)
	}

	return moduleName, result, nil
}

// Archive retrieves or loads an archive module by name
func (ctx Context) Archive(name string) (any, error) {
	if ar, ok := ctx.cfg.archives[name]; ok {
		return ar, nil
	}

	archiveRaw := ctx.cfg.ArchivesRaw[name]
	modVal, err := ctx.LoadModuleByID(name, archiveRaw)
	if err != nil {
		return nil, fmt.Errorf("loading %s app module: %v", name, err)
	}
	if archiveRaw != nil {
		ctx.cfg.ArchivesRaw[name] = nil // allow GC to deallocate
	}
	ctx.cfg.archives[name] = modVal.(Archive)
	return modVal, nil
}

// Logger returns a logger that is ready for the logarchive to use.
func (ctx Context) Logger() *zap.Logger {
	return ctx.cfg.Logging.logger
}
