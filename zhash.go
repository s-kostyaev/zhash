package libdeploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
)

const REQUIRED = "[REQUIRED]"

type Config map[string]interface{}

func NewConfig() Config {
	return Config{}
}

type NotFoundError struct {
	Path []string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("Value for %s not found", strings.Join(e.Path, "."))
}

func NewNotFoundError(path []string) error {
	return NotFoundError{path}
}

type RequiredError struct {
	Path string
}

func (e RequiredError) Error() string {
	return fmt.Sprintf("%s is required, please specify it by adding "+
		"key -k %s:<value>", e.Path, e.Path)
}

func (c *Config) ReadConfig(r io.Reader) error {
	_, err := toml.DecodeReader(r, &c)
	return err
}

func (c Config) WriteConfig(w io.Writer) error {
	return toml.NewEncoder(w).Encode(c)
}

func (c Config) Reader() io.Reader {
	var buff bytes.Buffer
	c.WriteConfig(&buff)
	return &buff
}

func (c Config) SetPath(value interface{}, path string) {
	c.Set(value, strings.Split(path, ".")...)
}

func (c Config) Set(value interface{}, path ...string) {
	key := ""
	ptr := map[string]interface{}(c)
	for i, p := range path {
		if i < len(path)-1 { // middle element
			switch node := ptr[p].(type) {
			case map[string]interface{}:
				ptr = node
			case Config:
				ptr = map[string]interface{}(node)
			default:
				ptr[p] = map[string]interface{}{}
				ptr = ptr[p].(map[string]interface{})
			}
		}
		key = p
	}

	ptr[key] = value
}

func (c Config) GetPath(path ...string) interface{} {
	ptr := c
	for i, p := range path {
		if i == len(path)-1 {
			return ptr[p]
		}

		switch node := ptr[p].(type) {
		case map[string]interface{}:
			ptr = node
		default:
			return nil
		}
	}

	return nil
}

func (c Config) GetMap(path ...string) (map[string]interface{}, error) {
	m := c.GetPath(path...)
	if m == nil {
		return map[string]interface{}{}, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case map[string]interface{}:
		return val, nil
	default:
		return map[string]interface{}{},
			errors.New(fmt.Sprintf("Error converting %s to map",
				strings.Join(path, ".")))
	}
}

func (c Config) GetString(path ...string) (string, error) {
	m := c.GetPath(path...)
	if m == nil {
		return "", NewNotFoundError(path)
	}
	switch val := m.(type) {
	case string:
		return val, nil
	default:
		return "", errors.New(fmt.Sprintf("Error converting %s to string",
			strings.Join(path, ".")))
	}
}

func (c Config) GetSlice(path ...string) ([]interface{}, error) {
	m := c.GetPath(path...)
	if m == nil {
		return []interface{}{}, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case []interface{}:
		return val, nil
	default:
		return []interface{}{},
			errors.New(fmt.Sprintf("Error converting %s to slice",
				strings.Join(path, ".")))
	}
}

func (c Config) GetStringSlice(path ...string) ([]string, error) {
	m := c.GetPath(path...)
	if m == nil {
		return []string{}, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case []interface{}:
		sl := []string{}
		for _, v := range val {
			switch s := v.(type) {
			case string:
				sl = append(sl, s)
			default:
				return []string{}, errors.New(
					fmt.Sprintf("Error converting %s to string slice",
						strings.Join(path, ".")))
			}
		}
		return sl, nil
	default:
		return []string{},
			errors.New(fmt.Sprintf("Error converting %s to slice",
				strings.Join(path, ".")))
	}
}

func (c Config) GetBool(path ...string) (bool, error) {
	m := c.GetPath(path...)
	if m == nil {
		return false, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case bool:
		return val, nil
	default:
		return false, errors.New(fmt.Sprintf("Error converting %s to bool",
			strings.Join(path, ".")))
	}
}

func (c Config) GetInt(path ...string) (int64, error) {
	m := c.GetPath(path...)
	if m == nil {
		return 0, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	default:
		return 0, errors.New(fmt.Sprintf("Error converting %s to int",
			strings.Join(path, ".")))
	}
}

func (c Config) GetFloat(path ...string) (float64, error) {
	m := c.GetPath(path...)
	if m == nil {
		return 0, NewNotFoundError(path)
	}
	switch val := m.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, errors.New(fmt.Sprintf("Error converting %s to float",
			strings.Join(path, ".")))
	}
}

func (c Config) Validate() (errs []error) {
	nodes := []interface{}{}
	paths := []string{}

	for p, v := range c {
		nodes = append(nodes, v)
		paths = append(paths, p)
	}

	for len(nodes) > 0 {
		node := nodes[len(nodes)-1]
		path := paths[len(paths)-1]
		nodes = nodes[:len(nodes)-1]
		paths = paths[:len(paths)-1]

		switch inner := node.(type) {
		case map[string]interface{}:
			for k, n := range inner {
				nodes = append(nodes, n)
				paths = append(paths, path+"."+k)
			}
		case string:
			if inner == REQUIRED {
				errs = append(errs, RequiredError{path})
			}
		}
	}

	return
}

func (c Config) String() string {
	buf, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "Error converting config to json"
	}

	return string(buf)
}