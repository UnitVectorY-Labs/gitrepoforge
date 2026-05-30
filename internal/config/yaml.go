package config

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func unmarshalYAMLKnownFields(data []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(out)
}
