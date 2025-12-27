package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// HelmRunner handles Helm deployment operations.
type HelmRunner struct {
	cfg *Config
}

// NewHelmRunner creates a new HelmRunner with the given configuration.
func NewHelmRunner(cfg *Config) *HelmRunner {
	return &HelmRunner{cfg: cfg}
}

// Run executes the Helm deployment workflow.
func (h *HelmRunner) Run() error {
	chartPath := h.cfg.ChartPath()

	if err := h.validateChartExists(chartPath); err != nil {
		return err
	}

	valuesFiles, err := h.collectValuesFiles()
	if err != nil {
		return fmt.Errorf("failed to collect values files: %w", err)
	}

	args := h.buildArgs(chartPath, valuesFiles)

	imageArgs, err := h.buildImageArgs()
	if err != nil {
		return err
	}
	args = append(args, imageArgs...)

	return h.execute(args)
}

func (h *HelmRunner) validateChartExists(chartPath string) error {
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		return fmt.Errorf("helm chart not found at path: %s. Please ensure the chart directory exists", chartPath)
	}
	log.Infof("✅ Helm chart found at: %s", chartPath)
	return nil
}

func (h *HelmRunner) collectValuesFiles() ([]string, error) {
	var files []string

	// Base values file (optional)
	baseValues := filepath.Join(".dockwright", "helm", "values.yaml")
	if _, err := os.Stat(baseValues); err == nil {
		files = append(files, baseValues)
		log.Infof("📄 Found base values file: %s", baseValues)
	}

	// Environment-specific values files
	for _, env := range h.cfg.Env {
		envValues := filepath.Join(".dockwright", "helm", fmt.Sprintf("%s.values.yaml", env))
		if _, err := os.Stat(envValues); os.IsNotExist(err) {
			return nil, fmt.Errorf("environment values file not found at path: %s. Please ensure the file exists", envValues)
		}
		files = append(files, envValues)
		log.Infof("📄 Found environment values file: %s", envValues)
	}

	log.Infof("✅ Collected %d values file(s) for deployment", len(files))
	return files, nil
}

func (h *HelmRunner) buildArgs(chartPath string, valuesFiles []string) []string {
	args := []string{
		"upgrade", "--install",
		h.cfg.ArtifactName,
		chartPath,
		"--kubeconfig", h.cfg.KubernetesConfig,
	}

	if h.cfg.KubernetesContext != "" {
		args = append(args, "--kube-context", h.cfg.KubernetesContext)
	}

	for _, f := range valuesFiles {
		args = append(args, "--values", f)
	}

	return args
}

func (h *HelmRunner) buildImageArgs() ([]string, error) {
	if h.cfg.ShouldRunDockerBuild() {
		imageRepo, err := h.cfg.ImageRepository()
		if err != nil {
			return nil, err
		}
		log.Infof("💉 Injecting image configuration into Helm deployment")
		log.Infof("   Repository: %s", imageRepo)
		log.Infof("   Tag: latest")
		return []string{
			"--set", fmt.Sprintf("image.repository=%s", imageRepo),
			"--set", "image.tag=latest",
		}, nil
	}

	return nil, nil
}

func (h *HelmRunner) execute(args []string) error {
	if h.cfg.DryRun {
		args = append(args, "--dry-run")
		log.Info("   🧪 [DRY-RUN] Would run: helm")
		h.logArgs(args)
		return nil
	}

	log.Infof("🚀 Executing Helm deployment for artifact: %s", h.cfg.ArtifactName)
	log.Infof("   Kubeconfig: %s", h.cfg.KubernetesConfig)
	if h.cfg.KubernetesContext != "" {
		log.Infof("   Context: %s", h.cfg.KubernetesContext)
	}
	log.Info("   Running: helm")
	h.logArgs(args)

	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm deployment failed: %w", err)
	}

	log.Infof("✓  Successfully deployed %s with Helm", h.cfg.ArtifactName)
	return nil
}

func (h *HelmRunner) logArgs(args []string) {
	log.Info("   Arguments:")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Pair flags with their values on the same line
		if i+1 < len(args) && strings.HasPrefix(arg, "--") && !strings.HasPrefix(args[i+1], "--") {
			log.Infof("     \033[32m%s\033[0m = %s", arg, args[i+1])
			i++ // skip the value
		} else {
			log.Infof("     %s", arg)
		}
	}
}
