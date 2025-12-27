package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Validator handles all pre-deployment validation checks.
type Validator struct {
	cfg *Config
}

// NewValidator creates a new Validator with the given configuration.
func NewValidator(cfg *Config) *Validator {
	return &Validator{cfg: cfg}
}

// ValidationResult represents the outcome of a validation check.
type ValidationResult struct {
	Name    string
	Icon    string
	Message string
	Err     error
}

// ValidateAll runs all validation checks and returns the first error encountered.
func (v *Validator) ValidateAll() ([]ValidationResult, error) {
	checks := []struct {
		name string
		icon string
		fn   func() error
	}{
		{"Configuration", "✅", v.validateConfig},
		{"Helm flavour", "⎈ ", v.validateHelmFlavour},
		{"Environment variables", "🔐", v.validateEnvVars},
		{"Environment values files", "📄", v.validateEnvValueFiles},
		{"Kubernetes context", "☸️ ", v.validateKubeContext},
		{"System tools", "🛠️ ", v.validateTools},
	}

	var results []ValidationResult

	for _, check := range checks {
		err := check.fn()
		result := ValidationResult{
			Name:    check.name,
			Icon:    check.icon,
			Message: fmt.Sprintf("%s Validated - %s", check.icon, check.name),
			Err:     err,
		}
		results = append(results, result)

		if err != nil {
			return results, err
		}
	}

	return results, nil
}

func (v *Validator) validateConfig() error {
	fields := ConfigFields()

	for _, field := range fields {
		if !field.Required {
			continue
		}

		value := v.getFieldValue(field.Name)
		if value == "" {
			return fmt.Errorf("configuration error: required field '%s' (flag: --%s) is not set", field.Name, field.Flag)
		}
	}

	return nil
}

func (v *Validator) getFieldValue(name string) string {
	switch name {
	case "artifactName":
		return v.cfg.ArtifactName
	case "helmFlavour":
		return v.cfg.HelmFlavour
	case "dockerNamespace":
		return v.cfg.DockerNamespace
	case "dockerHost":
		return v.cfg.DockerHost
	case "kubernetesConfig":
		return v.cfg.KubernetesConfig
	case "kubernetesContext":
		return v.cfg.KubernetesContext
	case "env":
		if len(v.cfg.Env) > 0 {
			return "set"
		}
		return ""
	default:
		return ""
	}
}

func (v *Validator) validateHelmFlavour() error {
	flavour := v.cfg.HelmFlavour
	if flavour != "stateful" && flavour != "stateless" {
		return fmt.Errorf("invalid helm flavour: expected 'stateful' or 'stateless', but got '%s'", flavour)
	}
	return nil
}

func (v *Validator) validateEnvVars() error {
	required := []string{"REGISTRY_USERNAME", "REGISTRY_PASSWORD"}

	for _, envVar := range required {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("required environment variable '%s' is not set. Please export %s before running Dockwright", envVar, envVar)
		}
	}

	return nil
}

func (v *Validator) validateTools() error {
	tools := []string{"docker", "helm"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("required tool '%s' is not installed or not found in PATH. Please install %s to proceed", tool, tool)
		}
	}

	// Verify Docker daemon is running
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon is not running. Please start Docker Desktop or the Docker daemon and try again")
	}

	return nil
}

func (v *Validator) validateEnvValueFiles() error {
	for _, env := range v.cfg.Env {
		path := filepath.Join(".dockwright", "helm", fmt.Sprintf("%s.values.yaml", env))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("environment values file not found at path: %s. Please ensure the file exists in the .dockwright/helm directory", path)
		}
	}
	return nil
}

func (v *Validator) validateKubeContext() error {
	if v.cfg.KubernetesContext == "" {
		return nil // Optional field
	}

	content, err := os.ReadFile(v.cfg.KubernetesConfig)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig file at path '%s': %w", v.cfg.KubernetesConfig, err)
	}

	var kubeconfig struct {
		Contexts []struct {
			Name string `yaml:"name"`
		} `yaml:"contexts"`
	}

	if err := yaml.Unmarshal(content, &kubeconfig); err != nil {
		return fmt.Errorf("failed to parse kubeconfig file at '%s': %w. The file may be corrupted or not in valid YAML format", v.cfg.KubernetesConfig, err)
	}

	for _, ctx := range kubeconfig.Contexts {
		if ctx.Name == v.cfg.KubernetesContext {
			return nil
		}
	}

	return fmt.Errorf("kubernetes context '%s' not found in kubeconfig at '%s'. Use 'kubectl config get-contexts' to see available contexts", v.cfg.KubernetesContext, v.cfg.KubernetesConfig)
}
