package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type PolicyConfig struct {
	EnforcementMode string `yaml:"enforcement_mode"`
	Network         struct {
		Allow []string `yaml:"allow"`
		Deny  []string `yaml:"deny"`
	} `yaml:"network"`
}

type Policy struct {
	Version int          `yaml:"version"`
	Config  PolicyConfig `yaml:"config"`
	Policy  PolicyConfig `yaml:"policy"` // backwards compatibility with legacy "policy:" key
}

// ParsePolicy securely unmarshals a YAML policy and validates its structure
func ParsePolicy(r io.Reader) (*Policy, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	if p.Version == 0 {
		return nil, fmt.Errorf("invalid policy: missing or invalid version")
	}

	// Fallback to legacy policy section if config section is empty
	if p.Config.EnforcementMode == "" && p.Policy.EnforcementMode != "" {
		p.Config = p.Policy
	}

	return &p, nil
}

// LoadPolicyFile searches for an agentsh/stuntdouble policy file in the given workspace directory
func LoadPolicyFile(workspaceDir string) (*Policy, string, error) {
	candidates := []string{
		".agentsh.yaml",
		".agentsh.yml",
		".stuntdouble.yaml",
		".stuntdouble.yml",
	}

	for _, name := range candidates {
		fullPath := filepath.Join(workspaceDir, name)
		if _, err := os.Stat(fullPath); err == nil {
			f, err := os.Open(fullPath)
			if err != nil {
				return nil, fullPath, fmt.Errorf("opening policy %s: %w", name, err)
			}
			defer f.Close()

			pol, err := ParsePolicy(f)
			if err != nil {
				return nil, fullPath, fmt.Errorf("parsing policy %s: %w", name, err)
			}
			return pol, fullPath, nil
		}
	}

	return nil, "", nil
}
