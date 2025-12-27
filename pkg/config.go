package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration values for Dockwright.
type Config struct {
	ArtifactName      string
	HelmFlavour       string
	DockerNamespace   string
	DockerHost        string
	KubernetesConfig  string
	KubernetesContext string
	Env               []string
	DryRun            bool
	RunDockerBuild    bool
	AutoApprove       bool
}

// ConfigField defines metadata for a single configuration option.
type ConfigField struct {
	Name        string
	ConfigPath  string // path in .dockwright/config.yaml
	Flag        string // CLI flag name
	Description string
	Required    bool
	Default     string
}

// ConfigFields returns all available configuration field definitions.
func ConfigFields() []ConfigField {
	return []ConfigField{
		{
			Name:        "artifactName",
			ConfigPath:  "artifactName",
			Flag:        "artifact-name",
			Description: "Name of the artifact",
			Required:    true,
			Default:     currentDirName(),
		},
		{
			Name:        "helmFlavour",
			ConfigPath:  "helm.flavour",
			Flag:        "helm-flavour",
			Description: "Helm chart flavour (stateful or stateless)",
			Required:    true,
		},
		{
			Name:        "dockerNamespace",
			ConfigPath:  "docker.namespace",
			Flag:        "docker-namespace",
			Description: "Docker registry namespace",
			Required:    false,
		},
		{
			Name:        "dockerHost",
			ConfigPath:  "docker.host",
			Flag:        "docker-host",
			Description: "Docker registry host",
			Required:    false,
			Default:     os.Getenv("REGISTRY_HOST"),
		},
		{
			Name:        "kubernetesConfig",
			ConfigPath:  "kubernetes.config",
			Flag:        "kubernetes-config",
			Description: "Path to kubernetes config file",
			Required:    true,
			Default:     defaultKubeConfigPath(),
		},
		{
			Name:        "kubernetesContext",
			ConfigPath:  "kubernetes.context",
			Flag:        "kubernetes-context",
			Description: "Kubernetes context to use",
			Required:    true,
			Default:     currentKubeContext(),
		},
		{
			Name:        "env",
			ConfigPath:  "env",
			Flag:        "env",
			Description: "Comma-separated list of environments (e.g., staging,production)",
			Required:    false,
		},
		{
			Name:        "dryRun",
			ConfigPath:  "dry-run",
			Flag:        "dry-run",
			Description: "Exercise the deployment pipeline without mutating resources",
			Required:    false,
			Default:     "false",
		},
		{
			Name:        "runDockerBuild",
			ConfigPath:  "docker.build",
			Flag:        "docker-build",
			Description: "Whether to run Docker build",
			Required:    false,
			Default:     "true",
		},
		{
			Name:        "autoApprove",
			ConfigPath:  "auto-approve",
			Flag:        "auto-approve",
			Description: "Skip confirmation prompts and proceed automatically",
			Required:    false,
			Default:     "false",
		},
	}
}

// LoadConfig loads configuration following precedence: CLI Flags > config.yaml > Defaults.
func LoadConfig(cmd *cobra.Command) (*Config, error) {
	cfg := &Config{}

	// Load from config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".dockwright")
	_ = viper.ReadInConfig() // Ignore error if file doesn't exist

	fields := ConfigFields()

	for _, field := range fields {
		value := resolveFieldValue(cmd, field)
		if err := setConfigField(cfg, field, value); err != nil {
			return nil, fmt.Errorf("failed to set config field %s: %w", field.Name, err)
		}
	}

	return cfg, nil
}

// resolveFieldValue determines the value for a field based on precedence.
func resolveFieldValue(cmd *cobra.Command, field ConfigField) string {
	// Priority 1: CLI flags
	if cmd != nil && cmd.Flags().Changed(field.Flag) {
		if val, err := cmd.Flags().GetString(field.Flag); err == nil {
			return val
		}
	}

	// Priority 2: Config file
	if viper.IsSet(field.ConfigPath) {
		return viper.GetString(field.ConfigPath)
	}

	// Priority 3: Default
	return field.Default
}

// setConfigField sets a field on the Config struct by name.
func setConfigField(cfg *Config, field ConfigField, value string) error {
	v := reflect.ValueOf(cfg).Elem()
	f := v.FieldByName(cases.Title(language.Und, cases.NoLower).String(field.Name))
	if !f.IsValid() {
		return fmt.Errorf("field %s not found in Config struct", field.Name)
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(value)
	case reflect.Bool:
		parsed := parseBool(value)
		f.SetBool(parsed)
	case reflect.Slice:
		if f.Type().Elem().Kind() == reflect.String {
			parsed := parseList(value, ",")
			slice := reflect.MakeSlice(f.Type(), len(parsed), len(parsed))
			for i, item := range parsed {
				slice.Index(i).SetString(item)
			}
			f.Set(slice)
		} else {
			return fmt.Errorf("unsupported slice element type: %s", f.Type().Elem().Kind())
		}
	default:
		return fmt.Errorf("unsupported field type: %s", f.Kind())
	}
	return nil
}

// parseBool parses a string to bool.
func parseBool(value string) bool {
	return value == "true" || value == "1" || strings.ToLower(value) == "yes"
}

// parseList parses a separator-separated string into a slice of strings.
func parseList(value, sep string) []string {
	if value == "" {
		return nil
	}
	var items []string
	for _, item := range strings.Split(value, sep) {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func defaultKubeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// ImageRepository returns the full Docker image repository path.
func (c *Config) ImageRepository() (string, error) {
	if c.DockerHost == "" || c.DockerNamespace == "" || c.ArtifactName == "" {
		return "", fmt.Errorf("dockerHost, dockerNamespace, and artifactName must all be set to generate image repository")
	}
	return fmt.Sprintf("%s/%s/%s", c.DockerHost, c.DockerNamespace, c.ArtifactName), nil
}

// ImageTag returns the full Docker image tag.
func (c *Config) ImageTag() (string, error) {
	repo, err := c.ImageRepository()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:latest", repo), nil
}

// ShouldRunDockerBuild returns true if Docker build should be run.
// It returns false if either Dockerfile is not found or runDockerBuild is set to false.
func (c *Config) ShouldRunDockerBuild() bool {
	_, err := os.Stat("Dockerfile")
	hasDockerfile := err == nil

	return hasDockerfile && c.RunDockerBuild
}

// ChartPath returns the path to the Helm chart based on flavour.
func (c *Config) ChartPath() string {
	return filepath.Join("/usr/local/share/dockwright/charts", c.HelmFlavour)
}

// Log prints the configuration in a tabular format.
func (c *Config) LogSummary() {
	log.Info("🛠️  Configuration loaded:")
	log.Info("   Field                | Value")
	log.Info("   ---------------------|----------------")

	v := reflect.ValueOf(c).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		fieldName := field.Name
		var coloredValue string

		switch value.Kind() {
		case reflect.String:
			coloredValue = fmt.Sprintf("\033[32m%s\033[0m", value.String())
		case reflect.Bool:
			coloredValue = fmt.Sprintf("\033[33m%t\033[0m", value.Bool())
		case reflect.Slice:
			coloredValue = fmt.Sprintf("\033[36m%v\033[0m", value.Interface())
		default:
			coloredValue = fmt.Sprintf("%v", value.Interface())
		}

		log.Infof("   %-20s | %s", fieldName, coloredValue)
	}
}

func currentKubeContext() string {
	path := defaultKubeConfigPath()
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var kubeconfig struct {
		CurrentContext string `yaml:"current-context"`
	}
	if err := yaml.Unmarshal(content, &kubeconfig); err != nil {
		return ""
	}
	return kubeconfig.CurrentContext
}

func currentDirName() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(dir)
}
