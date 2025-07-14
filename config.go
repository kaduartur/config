// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"
)

// Config represents a configuration with convenient access methods.
type Config struct {
	Root    any
	lastErr error
}

// Error return last error
func (c *Config) Error() error {
	return c.lastErr
}

// Get returns a nested config according to a dotted path.
func (c *Config) Get(path string) (*Config, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return nil, err
	}
	return &Config{Root: n}, nil
}

// Set a nested config according to a dotted path.
func (c *Config) Set(path string, val any) error {
	return Set(c.Root, path, val)
}

// Env fetch data from system env, based on existing config keys.
func (c *Config) Env() *Config {
	return c.EnvPrefix("")
}

// EnvPrefix fetch data from system env using prefix, based on existing config keys.
func (c *Config) EnvPrefix(prefix string) *Config {
	if prefix != "" {
		prefix = strings.ToUpper(prefix) + "_"
	}

	keys := getKeys(c.Root)
	for _, key := range keys {
		k := strings.ToUpper(strings.Join(key, "_"))
		if val, exist := syscall.Getenv(prefix + k); exist {
			_ = c.Set(strings.Join(key, "."), val)
		}
	}
	return c
}

// Flag parse command line arguments, based on existing config keys.
func (c *Config) Flag() *Config {
	keys := getKeys(c.Root)
	hash := map[string]*string{}
	for _, key := range keys {
		k := strings.Join(key, "-")
		hash[k] = new(string)
		val, _ := c.String(k)
		flag.StringVar(hash[k], k, val, "")
	}

	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		name := strings.ReplaceAll(f.Name, "-", ".")
		_ = c.Set(name, f.Value.String())
	})

	return c
}

// Args command line arguments, based on existing config keys.
func (c *Config) Args(args ...string) *Config {
	if len(args) <= 1 {
		return c
	}

	keys := getKeys(c.Root)
	hash := map[string]*string{}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var err bytes.Buffer
	f.SetOutput(&err)
	for _, key := range keys {
		k := strings.Join(key, "-")
		hash[k] = new(string)
		val, _ := c.String(k)
		f.StringVar(hash[k], k, val, "")
	}

	c.lastErr = f.Parse(args[1:])

	f.Visit(func(f *flag.Flag) {
		name := strings.ReplaceAll(f.Name, "-", ".")
		_ = c.Set(name, f.Value.String())
	})

	return c
}

// Get all keys for given interface
func getKeys(source any, base ...string) [][]string {
	var acc [][]string

	// Copy "base" so that underlying slice array is not
	// modified in recursive calls
	nextBase := make([]string, len(base))
	copy(nextBase, base)

	switch c := source.(type) {
	case map[string]any:
		for k, v := range c {
			keys := getKeys(v, append(nextBase, k)...)
			acc = append(acc, keys...)
		}
	case []any:
		for i, v := range c {
			k := strconv.Itoa(i)
			keys := getKeys(v, append(nextBase, k)...)
			acc = append(acc, keys...)
		}
	default:
		acc = append(acc, nextBase)
		return acc
	}
	return acc
}

// Bool returns a bool according to a dotted path.
func (c *Config) Bool(path string) (bool, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return false, err
	}
	switch n := n.(type) {
	case bool:
		return n, nil
	case string:
		return strconv.ParseBool(n)
	}
	return false, typeMismatch("bool or string", n)
}

// UBool returns a bool according to a dotted path or default value or false.
func (c *Config) UBool(path string, defaults ...bool) bool {
	value, err := c.Bool(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return false
}

// Float64 returns a float64 according to a dotted path.
func (c *Config) Float64(path string) (float64, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return 0, err
	}
	switch n := n.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	}
	return 0, typeMismatch("float64, int or string", n)
}

// UFloat64 returns a float64 according to a dotted path or default value or 0.
func (c *Config) UFloat64(path string, defaults ...float64) float64 {
	value, err := c.Float64(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return 0
}

// Int returns an int according to a dotted path.
func (c *Config) Int(path string) (int, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return 0, err
	}
	switch n := n.(type) {
	case float64:
		// encoding/json unmarshal numbers into floats, so we compare
		// the string representation to see if we can return an int.
		if i := int(n); fmt.Sprint(i) == fmt.Sprint(n) {
			return i, nil
		} else {
			return 0, fmt.Errorf("value can't be converted to int: %v", n)
		}
	case int:
		return n, nil
	case string:
		if v, err := strconv.ParseInt(n, 10, 0); err == nil {
			return int(v), nil
		} else {
			return 0, err
		}
	}
	return 0, typeMismatch("float64, int or string", n)
}

// UInt returns an int according to a dotted path or default value or 0.
func (c *Config) UInt(path string, defaults ...int) int {
	value, err := c.Int(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return 0
}

// List returns a []any according to a dotted path.
func (c *Config) List(path string) ([]any, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return nil, err
	}
	if value, ok := n.([]any); ok {
		return value, nil
	}
	return nil, typeMismatch("[]any", n)
}

// UList returns a []any according to a dotted path or defaults or []any.
func (c *Config) UList(path string, defaults ...[]any) []any {
	value, err := c.List(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return make([]any, 0)
}

// Map returns a map[string]any according to a dotted path.
func (c *Config) Map(path string) (map[string]any, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return nil, err
	}
	if value, ok := n.(map[string]any); ok {
		return value, nil
	}
	return nil, typeMismatch("map[string]any", n)
}

// UMap returns a map[string]any according to a dotted path or default or map[string]any.
func (c *Config) UMap(path string, defaults ...map[string]any) map[string]any {
	value, err := c.Map(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return map[string]any{}
}

// String returns a string according to a dotted path.
func (c *Config) String(path string) (string, error) {
	n, err := Get(c.Root, path)
	if err != nil {
		return "", err
	}
	switch n := n.(type) {
	case bool, float64, int:
		return fmt.Sprint(n), nil
	case string:
		return n, nil
	}
	return "", typeMismatch("bool, float64, int or string", n)
}

// UString returns a string according to a dotted path or default or "".
func (c *Config) UString(path string, defaults ...string) string {
	value, err := c.String(path)

	if err == nil {
		return value
	}

	for _, def := range defaults {
		return def
	}
	return ""
}

// Copy returns a deep copy with given path or without.
func (c *Config) Copy(dottedPath ...string) (*Config, error) {
	var toJoin []string
	for _, part := range dottedPath {
		if len(part) != 0 {
			toJoin = append(toJoin, part)
		}
	}

	var err error
	var path = strings.Join(toJoin, ".")
	var cfg = c
	var root string

	if len(path) > 0 {
		if cfg, err = c.Get(path); err != nil {
			return nil, err
		}
	}

	if root, err = RenderYaml(cfg.Root); err != nil {
		return nil, err
	}
	return ParseYaml(root)
}

// Extend extends the current config with the given config.
//
// Extend will merge arrays in the source config into arrays in the target config.
// If a key in the source config is not present in the target config, it will be
// added. If a key is present in both the source and target config and is not an
// array, the value from the source config will be used.
//
// This is useful for extending a base configuration with additional configuration
// options.
func (c *Config) Extend(cfg *Config) (*Config, error) {
	// First create a deep copy of the current config
	n, err := c.Copy()
	if err != nil {
		return nil, err
	}

	// Find all arrays in the source config
	arrayPaths := findArrayPaths(cfg.Root)
	processedPaths := make(map[string]bool)

	// Process arrays first to ensure they are properly merged
	for _, path := range arrayPaths {
		if path == "" {
			continue // Skip the root path
		}

		// Get the array from the source config
		sourceArr, err := cfg.List(path)
		if err != nil {
			return nil, err
		}

		// Try to get the array from the target config
		targetArr, err := n.List(path)
		if err == nil {
			// We have arrays in both source and target, merge them
			mergedArr := make([]any, len(targetArr))
			copy(mergedArr, targetArr)

			// Override existing elements and append new ones
			for i, item := range sourceArr {
				if i < len(mergedArr) {
					// Override existing element
					mergedArr[i] = item
				} else {
					// Append new element
					mergedArr = append(mergedArr, item)
				}
			}

			// Set the merged array in the target config
			if err := n.Set(path, mergedArr); err != nil {
				return nil, err
			}
		} else {
			// Target doesn't have an array at this path, just set the source array
			if err := n.Set(path, sourceArr); err != nil {
				return nil, err
			}
		}

		// Mark this path as processed
		processedPaths[path] = true
	}

	// Process all other keys from the source config
	keys := getKeys(cfg.Root)
	for _, key := range keys {
		k := strings.Join(key, ".")

		// Skip paths that are arrays or elements of arrays we've already processed
		skipPath := false
		for path := range processedPaths {
			if k == path || strings.HasPrefix(k, path+".") {
				skipPath = true
				break
			}
		}

		if skipPath {
			continue
		}

		// Get the value from the source config
		i, err := Get(cfg.Root, k)
		if err != nil {
			return nil, err
		}

		// Set the value in the target config
		if err := n.Set(k, i); err != nil {
			return nil, err
		}
	}

	return n, nil
}

// findArrayPaths finds all paths in the config that are arrays
func findArrayPaths(root any) []string {
	var paths []string
	findArrayPathsRecursive(root, "", &paths)
	return paths
}

// findArrayPathsRecursive is a helper function for findArrayPaths
func findArrayPathsRecursive(value any, path string, paths *[]string) {
	switch v := value.(type) {
	case []any:
		*paths = append(*paths, path)
	case map[string]any:
		for k, val := range v {
			newPath := path
			if newPath != "" {
				newPath += "."
			}
			newPath += k
			findArrayPathsRecursive(val, newPath, paths)
		}
	}
}

// typeMismatch returns an error for an expected type.
func typeMismatch(expected string, got any) error {
	return fmt.Errorf("type mismatch: expected %s; got %T", expected, got)
}

// Get returns a value according to a dotted path.
//
// The path is split into parts on the dot character. If a part is an empty
// string, it is removed from the path. If the first part is an empty string,
// it is treated as a noop.
//
// The value is then retrieved from the configuration using the resulting
// path. If the path is invalid (i.e. a nonexistent map key or an out-of-range
// list index), an error is returned.
//
// The function handles the following types:
//
// - map[string]any
// - []any
//
// If the value at the given path is not one of the above types, an
// error is returned.
//
// Examples:
//
//	config := map[string]interface{}{
//	    "database": map[string]interface{}{
//	        "host": "localhost",
//	    },
//	}
//
// value, err := Get(config, "database.host")
// // value is "localhost"
//
// value, err := Get(config, "database.ports.0")
// // err is a type mismatch error, because config["database"]["ports"] is
// // not a list
func Get(cfg any, path string) (any, error) {
	parts := splitKeyOnParts(path)
	// Normalize path.
	for k, v := range parts {
		if v == "" {
			if k == 0 {
				parts = parts[1:]
			} else {
				return nil, fmt.Errorf("invalid path %q", path)
			}
		}
	}
	// Get the value.
	for pos, part := range parts {
		switch c := cfg.(type) {
		case []any:
			if i, err := strconv.ParseInt(part, 10, 0); err == nil {
				if int(i) < len(c) {
					cfg = c[i]
				} else {
					return nil, fmt.Errorf(
						"index out of range at %q: list has only %v items",
						strings.Join(parts[:pos+1], "."), len(c))
				}
			} else {
				return nil, fmt.Errorf("invalid list index at %q",
					strings.Join(parts[:pos+1], "."))
			}
		case map[string]any:
			if value, ok := c[part]; ok {
				cfg = value
			} else {
				return nil, fmt.Errorf("nonexistent map key at %q",
					strings.Join(parts[:pos+1], "."))
			}
		default:
			return nil, fmt.Errorf(
				"invalid type at %q: expected []any or map[string]any; got %T",
				strings.Join(parts[:pos+1], "."), cfg)
		}
	}

	return cfg, nil
}

func splitKeyOnParts(key string) []string {
	var parts []string

	bracketOpened := false
	var buffer bytes.Buffer
	for _, char := range key {
		if char == 91 || char == 93 { // [ ]
			bracketOpened = char == 91
			continue
		}
		if char == 46 && !bracketOpened { // point
			parts = append(parts, buffer.String())
			buffer.Reset()
			continue
		}

		buffer.WriteRune(char)
	}

	if buffer.String() != "" {
		parts = append(parts, buffer.String())
		buffer.Reset()
	}

	return parts
}

// Set sets a value in a nested configuration structure
// according to a path specified in dotted notation. The
// `cfg` parameter can be a map or a slice, and the `path`
// is a string representing keys or indices separated by dots.
// If the path leads to a nonexistent key or index, the
// necessary maps or slices are created. This function
// returns an error if the path is invalid or if a type
// mismatch occurs at any part of the path.
func Set(cfg any, path string, value any) error {
	parts := strings.Split(path, ".")
	// Normalize path.
	for k, v := range parts {
		if v == "" {
			if k != 0 {
				return fmt.Errorf("invalid path %q", path)
			}

			parts = parts[1:]
		}
	}

	if len(parts) == 0 {
		return nil
	}

	// Handle the root case
	switch c := cfg.(type) {
	case map[string]any:
		if len(parts) == 1 {
			c[parts[0]] = value
			return nil
		}
		if v, ok := c[parts[0]]; ok {
			return Set(v, strings.Join(parts[1:], "."), value)
		}
		// If the path doesn't exist, create it
		if i, err := strconv.Atoi(parts[1]); err == nil {
			// Next part is a numeric index, create a slice
			newSlice := make([]any, i+1)
			c[parts[0]] = newSlice
			return Set(newSlice, strings.Join(parts[1:], "."), value)
		}
		// Next part is a string key, create a map
		newMap := make(map[string]any)
		c[parts[0]] = newMap
		return Set(newMap, strings.Join(parts[1:], "."), value)
	case []any:
		// First part must be a numeric index for slices
		i, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid list index at %q", parts[0])
		}
		// Ensure the slice is large enough
		for len(c) <= i {
			c = append(c, nil)
		}
		if len(parts) == 1 {
			c[i] = value
			return nil
		}
		// If the path doesn't exist or is nil, create it
		if c[i] == nil {
			if j, err := strconv.Atoi(parts[1]); err == nil {
				// Next part is a numeric index, create a slice
				newSlice := make([]any, j+1)
				c[i] = newSlice
			} else {
				// Next part is a string key, create a map
				newMap := make(map[string]any)
				c[i] = newMap
			}
		}
		return Set(c[i], strings.Join(parts[1:], "."), value)
	default:
		return fmt.Errorf("invalid type at root: expected []any or map[string]any; got %T", cfg)
	}
}

// Must is a helper that wraps a call to a function returning (*Config, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//
//	var cfg = config.Must(config.ParseYaml(yamlString))
func Must(cfg *Config, err error) *Config {
	if err != nil {
		panic(err)
	}
	return cfg
}

// normalizeValue takes a value and recursively normalizes it, converting
// any map[any]any to map[string]any, and any []any to []any, and returning
// an error if any value is of an unsupported type.
//
// This function is intended for use when the contents of the value are
// unknown, and the value needs to be converted to a form that can be
// safely used as a configuration. For example, if the value is a JSON
// object received from an untrusted source, this function can be used to
// convert it to a form that can be safely used as a configuration.
func normalizeValue(value any) (any, error) {
	switch value := value.(type) {
	case map[any]any:
		node := make(map[string]any, len(value))
		for k, v := range value {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("unsupported map key: %#v", k)
			}
			item, err := normalizeValue(v)
			if err != nil {
				return nil, fmt.Errorf("unsupported map value: %#v", v)
			}
			node[key] = item
		}
		return node, nil
	case map[string]any:
		node := make(map[string]any, len(value))
		for key, v := range value {
			item, err := normalizeValue(v)
			if err != nil {
				return nil, fmt.Errorf("unsupported map value: %#v", v)
			}
			node[key] = item
		}
		return node, nil
	case []any:
		node := make([]any, len(value))
		for key, v := range value {
			item, err := normalizeValue(v)
			if err != nil {
				return nil, fmt.Errorf("unsupported list item: %#v", v)
			}
			node[key] = item
		}
		return node, nil
	case bool, float64, int, string, nil:
		return value, nil
	}
	return nil, fmt.Errorf("unsupported type: %T", value)
}

// ParseJson parses a JSON configuration from the given string.
//
// The contents of the string should be a valid JSON object. The function
// will return an error if the JSON is invalid.
//
// The resulting configuration is returned as a *Config, which can be used
// to access the configuration values.
func ParseJson(cfg string) (*Config, error) {
	return parseJson([]byte(cfg))
}

// ParseJsonFile reads a JSON configuration from the given filename.
//
// The contents of the file should be a valid JSON object. The function
// will return an error if the JSON is invalid.
//
// The resulting configuration is returned as a *Config, which can be used
// to access the configuration values.
func ParseJsonFile(filename string) (*Config, error) {
	cfg, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	return parseJson(cfg)
}

// parseJson performs the real JSON parsing.
func parseJson(cfg []byte) (*Config, error) {
	var out any
	var err error
	if err = json.Unmarshal(cfg, &out); err != nil {
		return nil, err
	}
	if out, err = normalizeValue(out); err != nil {
		return nil, err
	}
	return &Config{Root: out}, nil
}

// RenderJson renders a JSON configuration.
//
// The given configuration is marshaled into a JSON object, and the
// resulting JSON is returned as a string.
//
// If the configuration cannot be marshaled, the function returns an error.
func RenderJson(cfg any) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ParseYamlBytes parses a YAML configuration from the given byte slice.
//
// The contents of the byte slice should be a valid YAML object. The function
// will return an error if the YAML is invalid.
//
// The resulting configuration is returned as a *Config, which can be used
// to access the configuration values.
func ParseYamlBytes(cfg []byte) (*Config, error) {
	return parseYaml(cfg)
}

// ParseYaml parses a YAML configuration from the given string.
//
// The contents of the string should be a valid YAML object. The function
// will return an error if the YAML is invalid.
//
// The resulting configuration is returned as a *Config, which can be used
// to access the configuration values.
func ParseYaml(cfg string) (*Config, error) {
	return parseYaml([]byte(cfg))
}

// ParseYamlFile reads a YAML configuration from the given filename.
//
// The contents of the file should be a valid YAML object. The function
// will return an error if the YAML is invalid.
//
// The resulting configuration is returned as a *Config, which can be used
// to access the configuration values.
func ParseYamlFile(filename string) (*Config, error) {
	cfg, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	return parseYaml(cfg)
}

// RenderYaml marshals the given configuration into a YAML formatted string.
//
// The cfg parameter can be any data structure that is compatible with YAML
// marshaling. If the configuration cannot be marshaled, the function returns
// an error.
//
// Returns:
//   - string: A YAML formatted string representing the configuration.
//   - error: An error object if marshaling fails, otherwise nil.
func RenderYaml(cfg any) (string, error) {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseYaml performs the real YAML parsing.
func parseYaml(cfg []byte) (*Config, error) {
	var out any
	var err error
	if err = yaml.Unmarshal(cfg, &out); err != nil {
		return nil, err
	}
	if out, err = normalizeValue(out); err != nil {
		return nil, err
	}
	return &Config{Root: out}, nil
}
