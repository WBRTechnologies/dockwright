package pkg

import (
	"bufio"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "dockwright",
		Short: "Dockwright is a modular CLI for Docker & Helm orchestration",
	}

	deployCmd = &cobra.Command{
		Use:          "deploy",
		Short:        "Deploy the application",
		SilenceUsage: true,
		RunE:         runDeploy,
	}
)

func init() {
	rootCmd.AddCommand(deployCmd)

	// Dynamically register flags from ConfigFields
	for _, field := range ConfigFields() {
		deployCmd.Flags().String(field.Flag, field.Default, field.Description)
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Configure logger
	log.SetTimeFormat("")

	// Step 1: Configuration
	logSection(1, "CONFIGURATION", "⚙️")

	cfg, err := LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("❌ failed to load configuration: %w", err)
	}
	cfg.LogSummary()

	// User confirmation
	if !cfg.AutoApprove && !cfg.DryRun {
		fmt.Print("Please confirm the configuration above. Press Enter to proceed with deployment: ")
		reader := bufio.NewReader(os.Stdin)
		_, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}
	}

	// Step 2: Validation
	logSection(2, "VALIDATION", "✓")

	validator := NewValidator(cfg)
	results, err := validator.ValidateAll()
	for _, r := range results {
		if r.Err != nil {
			log.Errorf("❌ Validation error in %s", r.Name)
			return err
		} else {
			log.Info(r.Message)
		}
	}
	if err != nil {
		return err
	}

	// Step 3: Docker Workflow
	logSection(3, "DOCKER WORKFLOW", "🐳")

	dockerRunner := NewDockerRunner(cfg)
	if err := dockerRunner.Run(); err != nil {
		return fmt.Errorf("❌ docker workflow failed: %w", err)
	}

	// Step 4: Helm Workflow
	logSection(4, "HELM WORKFLOW", "⎈")

	helmRunner := NewHelmRunner(cfg)
	if err := helmRunner.Run(); err != nil {
		return fmt.Errorf("❌ helm workflow failed: %w", err)
	}

	// Complete
	logSection(0, "DEPLOYMENT COMPLETE", "🎉")

	return nil
}

func logSection(num int, title, icon string) {
	log.Info("")
	log.Info("═══════════════════════════════════════════════════════════════")
	if num > 0 {
		log.Infof("%s  %d. %s", icon, num, title)
	} else {
		log.Infof("%s %s", icon, title)
	}
	log.Info("═══════════════════════════════════════════════════════════════")
}
