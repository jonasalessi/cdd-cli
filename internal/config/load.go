package config

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and parses the file at path. A missing file wraps
// os.ErrNotExist so callers can tell it apart from a malformed one.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	defer f.Close()
	cfg, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("config: %s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes a cdd.config.yaml document. Unknown keys are an error and
// pattern lists keep their document order.
func Parse(r io.Reader) (*Config, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("empty document")
		}
		return nil, err
	}
	return &cfg, nil
}

// UnmarshalYAML keeps the pattern order of the document, which a Go map
// would lose.
func (p *PatternWeights) UnmarshalYAML(n *yaml.Node) error {
	var out PatternWeights
	err := eachPair(n, func(key string, value *yaml.Node) error {
		var w map[MetricID]float64
		if err := value.Decode(&w); err != nil {
			return err
		}
		out = append(out, PatternWeight{Pattern: key, Weights: w})
		return nil
	})
	if err != nil {
		return err
	}
	*p = out
	return nil
}

// UnmarshalYAML keeps the pattern order of the document.
func (p *PatternLimits) UnmarshalYAML(n *yaml.Node) error {
	var out PatternLimits
	err := eachPair(n, func(key string, value *yaml.Node) error {
		var limit int
		if err := value.Decode(&limit); err != nil {
			return err
		}
		out = append(out, PatternLimit{Pattern: key, Limit: limit})
		return nil
	})
	if err != nil {
		return err
	}
	*p = out
	return nil
}

// eachPair walks the key/value pairs of a mapping node in document order.
// A null node is an empty mapping; anything else is an error naming the line.
func eachPair(n *yaml.Node, fn func(key string, value *yaml.Node) error) error {
	if n.Tag == "!!null" {
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a mapping of patterns", n.Line)
	}
	seen := make(map[string]int, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if first, dup := seen[k.Value]; dup {
			return fmt.Errorf("line %d: pattern %q already defined at line %d", k.Line, k.Value, first)
		}
		seen[k.Value] = k.Line
		if err := fn(k.Value, v); err != nil {
			return err
		}
	}
	return nil
}
