package pkg

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
)

// DockerRunner handles Docker build, login, and push operations.
type DockerRunner struct {
	cfg *Config
}

// NewDockerRunner creates a new DockerRunner with the given configuration.
func NewDockerRunner(cfg *Config) *DockerRunner {
	return &DockerRunner{cfg: cfg}
}

// Run executes the Docker workflow: build, login, and push.
func (d *DockerRunner) Run() error {
	if !d.cfg.ShouldRunDockerBuild() {
		log.Info("⏭️  Skipping Docker workflow. Either docker build (--docker-build) flag is disabled or Dockerfile is missing.")
		return nil
	}

	imageTag, err := d.cfg.ImageTag()
	if err != nil {
		return err
	}

	if err := d.build(imageTag); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	if err := d.login(); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}

	if err := d.push(imageTag); err != nil {
		return fmt.Errorf("docker push failed: %w", err)
	}

	return nil
}

func (d *DockerRunner) build(imageTag string) error {
	log.Infof("🔨 Building Docker image: %s", imageTag)
	log.Infof("   Build context: %s", ".")

	if d.cfg.DryRun {
		log.Infof("   🧪 [DRY-RUN] Would run: docker build -t %s .", imageTag)
		return nil
	}

	cmd := exec.Command("docker", "build", "-t", imageTag, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	log.Infof("✓  Successfully built Docker image: %s", imageTag)
	return nil
}

func (d *DockerRunner) login() error {
	username := os.Getenv("REGISTRY_USERNAME")
	password := os.Getenv("REGISTRY_PASSWORD")

	if username == "" || password == "" {
		return fmt.Errorf("REGISTRY_USERNAME and REGISTRY_PASSWORD environment variables must be set for Docker login")
	}

	log.Infof("🔐 Authenticating with Docker registry: %s", d.cfg.DockerHost)
	log.Infof("   Username: %s", username)

	if d.cfg.DryRun {
		log.Infof("   🧪 [DRY-RUN] Would run: docker login %s -u %s", d.cfg.DockerHost, username)
		return nil
	}

	cmd := exec.Command("docker", "login", d.cfg.DockerHost, "-u", username, "--password-stdin")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker login: %w", err)
	}

	if _, err := fmt.Fprintln(stdin, password); err != nil {
		return fmt.Errorf("failed to write password: %w", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}

	log.Infof("✓  Successfully authenticated with registry: %s", d.cfg.DockerHost)
	return nil
}

func (d *DockerRunner) push(imageTag string) error {
	log.Infof("📤 Pushing Docker image: %s", imageTag)
	log.Infof("   Target registry: %s", d.cfg.DockerHost)

	if d.cfg.DryRun {
		log.Infof("   🧪 [DRY-RUN] Would run: docker push %s", imageTag)
		return nil
	}

	cmd := exec.Command("docker", "push", imageTag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	log.Infof("✓  Successfully pushed image to registry: %s", imageTag)
	return nil
}
