package api

import (
	"fmt"
	"math"

	"robusta-web/backend/internal/models"
)

// toUint64 converts various numeric/string representations to uint64 for controller-level binding.
func toUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value")
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value")
		}
		return uint64(v), nil
	case float64:
		if v < 0 || v != math.Trunc(v) {
			return 0, fmt.Errorf("invalid float value")
		}
		return uint64(v), nil
	case string:
		return models.ParseID(v)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}
