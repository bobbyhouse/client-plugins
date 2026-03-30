package profile

import (
	"fmt"
	"regexp"
	"strings"
)

// Profile describes the set of MCP servers declared in a profile image.
type Profile struct {
	Name    string   `yaml:"name"`
	Servers []Server `yaml:"servers"`
}

// Server is a single MCP server declared in a profile.
type Server struct {
	Name       string            `yaml:"name"`
	Identifier string            `yaml:"identifier"`
	Config     map[string]string `yaml:"config"`
	Mounts     []string          `yaml:"mounts"`
}

// MissingConfigError is returned when one or more ${KEY} placeholders in a
// profile's config values have not been supplied by the caller.
type MissingConfigError struct {
	Keys []string
}

func (e *MissingConfigError) Error() string {
	return fmt.Sprintf("missing required config keys: %s", strings.Join(e.Keys, ", "))
}

var placeholderRE = regexp.MustCompile(`\$\{([^}]+)\}`)

// ResolvePlaceholders replaces ${KEY} placeholders in each server's config
// values and mount strings with the corresponding entry from userConfig. It
// returns *MissingConfigError if any placeholder has no supplied value.
func ResolvePlaceholders(p *Profile, userConfig map[string]string) error {
	var missing []string
	resolve := func(v string) string {
		return placeholderRE.ReplaceAllStringFunc(v, func(m string) string {
			key := m[2 : len(m)-1]
			if val, ok := userConfig[key]; ok {
				return val
			}
			missing = append(missing, key)
			return m
		})
	}
	for i, srv := range p.Servers {
		for k, v := range srv.Config {
			p.Servers[i].Config[k] = resolve(v)
		}
		for j, mount := range srv.Mounts {
			p.Servers[i].Mounts[j] = resolve(mount)
		}
	}
	if len(missing) > 0 {
		return &MissingConfigError{Keys: missing}
	}
	return nil
}
