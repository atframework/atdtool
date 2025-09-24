package noncloudnative

import "strings"

// deploy layer
const (
	LayerNone = iota
	LayerCluster
	LayerWorld
	LayerZone
)

// GetLayer get layer id by name
func GetLayer(name string) int {
	layer := LayerNone
	if strings.Compare(strings.ToLower(name), "cluster") == 0 {
		layer = LayerCluster
	} else if strings.Compare(strings.ToLower(name), "world") == 0 {
		layer = LayerWorld
	} else if strings.Compare(strings.ToLower(name), "zone") == 0 {
		layer = LayerZone
	}
	return layer
}

// GetLayerName get layer name by id
func GetLayerName(layer int) string {
	switch layer {
	case LayerCluster:
		return "cluster"
	case LayerWorld:
		return "world"
	case LayerZone:
		return "zone"
	}
	return "unknown"
}
