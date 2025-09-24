package yaml

import (
	"encoding/json"
	"os"

	"sigs.k8s.io/yaml"
)

// Load YAML document from file and assigns decoded values into the out value.
func LoadConfig(name string, out any) (err error) {
	var data []byte
	data, err = os.ReadFile(name)
	if err != nil {
		return
	}

	err = yaml.UnmarshalStrict(data, out, func(d *json.Decoder) *json.Decoder {
		d.UseNumber()
		return d
	})
	return
}
