package types

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	specA = `{
  "Instance": {
    "Plugin": "a",
    "Properties": {
      "a": "a",
      "b": "b",
      "c": {
        "d": "d",
        "e": "e"
      }
    }
  },
  "Flavor": {
    "Plugin": "f",
    "Properties": {
	"g": "g"
    }
  }
}`

	reordered = `{
  "Instance": {
    "Plugin": "a",
    "Properties": {
      "a": "a",
      "c": {
        "e": "e",
        "d": "d"
      },
      "b": "b"
    }
  },
  "Flavor": {
    "Plugin": "f",
    "Properties": {
	"g": "g"
    }
  }
}`

	different = `{
  "Instance": {
    "Plugin": "a",
    "Properties": {
      "a": "a",
      "c": {
        "d": "d"
      }
    }
  },
  "Flavor": {
    "Plugin": "f",
    "Properties": {
	"g": "g"
    }
  }
}`
)

func TestInstanceHash(t *testing.T) {
	hash := func(config string) string {
		spec := Spec{}
		err := json.Unmarshal([]byte(config), &spec)
		require.NoError(t, err)
		return spec.InstanceHash()
	}

	require.Equal(t, hash(specA), hash(specA))
	require.Equal(t, hash(specA), hash(reordered))
	require.NotEqual(t, hash(specA), hash(different))
}
