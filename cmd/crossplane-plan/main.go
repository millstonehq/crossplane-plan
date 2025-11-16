package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/millstonehq/crossplane-plan/pkg/config"
	"github.com/millstonehq/crossplane-plan/pkg/detector"
	"github.com/millstonehq/crossplane-plan/pkg/differ"
	"github.com/millstonehq/crossplane-plan/pkg/formatter"
	"github.com/millstonehq/crossplane-plan/pkg/vcs/github"
	"github.com/millstonehq/crossplane-plan/pkg/watcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	kubeconfig              string
	detectionStrategy       string
	namePattern             string
	githubRepo              string
	githubToken             string
	githubCredentials       string
	githubAppID             string
	githubInstallID         string
	githubAppKeyPath        string
	dryRun                  bool
	reconciliationInterval  int
	configPath              string
	noStripDefaults         bool
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not specified)")
	flag.StringVar(&detectionStrategy, "detection-strategy", "name", "PR detection strategy: name, label, or annotation")
	flag.StringVar(&namePattern, "name-pattern", "pr-{number}-*", "Name pattern for PR detection (when strategy=name)")
	flag.StringVar(&githubRepo, "github-repo", "", "GitHub repository (format: owner/repo)")
	flag.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (can also use GITHUB_TOKEN env var)")
	flag.StringVar(&githubCredentials, "github-credentials", os.Getenv("GITHUB_CREDENTIALS"), "GitHub credentials in crossplane-provider-github format (base64-encoded JSON)")
	flag.StringVar(&githubAppID, "github-app-id", os.Getenv("GITHUB_APP_ID"), "GitHub App ID (can also use GITHUB_APP_ID env var)")
	flag.StringVar(&githubInstallID, "github-installation-id", os.Getenv("GITHUB_INSTALLATION_ID"), "GitHub Installation ID (can also use GITHUB_INSTALLATION_ID env var)")
	flag.StringVar(&githubAppKeyPath, "github-app-key-path", os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"), "Path to GitHub App private key file (can also use GITHUB_APP_PRIVATE_KEY_PATH env var)")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run mode - calculate diffs but don't post to GitHub")
	flag.IntVar(&reconciliationInterval, "reconciliation-interval", 5, "Periodic reconciliation interval in minutes (0 to disable)")
	flag.StringVar(&configPath, "config", "/etc/crossplane-plan/config.yaml", "Path to config file for field stripping rules")
	flag.BoolVar(&noStripDefaults, "no-strip-defaults", false, "Disable default field stripping rules")
}

func main() {
	flag.Parse()

	// Set up logging
	zapLogger := zap.New(zap.UseDevMode(true))
	logrLogger := zapLogger.WithName("crossplane-plan")
	logger := logging.NewLogrLogger(logrLogger)

	logger.Info("Starting crossplane-plan",
		"detectionStrategy", detectionStrategy,
		"namePattern", namePattern,
		"githubRepo", githubRepo,
		"dryRun", dryRun,
	)

	// Validate required flags
	if githubRepo == "" {
		logrLogger.Error(fmt.Errorf("github-repo is required"), "missing required flag")
		os.Exit(1)
	}

	// Validate authentication config (unless dry-run)
	if !dryRun {
		hasToken := githubToken != ""
		hasCredentials := githubCredentials != ""
		hasAppCreds := githubAppID != "" && githubInstallID != "" && githubAppKeyPath != ""

		if !hasToken && !hasCredentials && !hasAppCreds {
			logrLogger.Error(
				fmt.Errorf("authentication required"),
				"missing authentication",
				"hint", "provide GITHUB_TOKEN, GITHUB_CREDENTIALS, or GitHub App credentials (GITHUB_APP_ID, GITHUB_INSTALLATION_ID, GITHUB_APP_PRIVATE_KEY_PATH)",
			)
			os.Exit(1)
		}
	}

	// Build Kubernetes config
	cfg, err := buildKubeConfig()
	if err != nil {
		logrLogger.Error(err, "failed to build kubernetes config")
		os.Exit(1)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrLogger.Error(err, "failed to create kubernetes clientset")
		os.Exit(1)
	}

	// Load config file
	appConfig, err := config.LoadConfig(configPath)
	if err != nil {
		logrLogger.Error(err, "failed to load config")
		os.Exit(1)
	}

	// Set CLI-only fields (not in config file)
	appConfig.DetectionStrategy = detectionStrategy
	appConfig.NamePattern = namePattern
	appConfig.GitHubRepo = githubRepo
	appConfig.DryRun = dryRun

	// Create PR detector
	prDetector, err := createDetector(appConfig)
	if err != nil {
		logrLogger.Error(err, "failed to create PR detector")
		os.Exit(1)
	}

	// Create differ
	diffCalculator := differ.NewCalculator(cfg, logger)

	// Override stripDefaults if CLI flag is set
	if noStripDefaults {
		appConfig.Diff.StripDefaults = false
	}

	// Create and configure sanitizer
	stripRules := appConfig.GetAllStripRules()
	if len(stripRules) > 0 {
		sanitizer := differ.NewSanitizer(stripRules)
		diffCalculator.SetSanitizer(sanitizer)
		logger.Info("Field stripping enabled", "ruleCount", len(stripRules))
	} else {
		logger.Info("Field stripping disabled")
	}

	// Create formatter
	diffFormatter := formatter.NewGitHubFormatter()

	// Create VCS client (if not dry-run)
	var vcsClient *github.Client
	if !dryRun {
		vcsClient, err = createGitHubClient()
		if err != nil {
			logrLogger.Error(err, "failed to create GitHub client")
			os.Exit(1)
		}
		logger.Info("GitHub client created successfully",
			"authMethod", getAuthMethod(),
			"repo", githubRepo,
		)
	}

	// Create and start watcher
	xrWatcher := watcher.NewXRWatcher(
		clientset,
		prDetector,
		diffCalculator,
		diffFormatter,
		vcsClient,
		logrLogger,
		reconciliationInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start watching
	if err := xrWatcher.Start(ctx); err != nil {
		logrLogger.Error(err, "watcher failed")
		os.Exit(1)
	}

	logger.Info("Shutting down gracefully")
}

func buildKubeConfig() (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func createDetector(cfg *config.Config) (detector.Detector, error) {
	switch cfg.DetectionStrategy {
	case "name":
		return detector.NewNameDetector(cfg.NamePattern), nil
	case "label":
		return detector.NewLabelDetector(), nil
	case "annotation":
		return detector.NewAnnotationDetector(), nil
	default:
		return nil, fmt.Errorf("unknown detection strategy: %s", cfg.DetectionStrategy)
	}
}

func createGitHubClient() (*github.Client, error) {
	// Build client config
	config := &github.ClientConfig{
		Repository: githubRepo,
	}

	// Priority: token > credentials > direct GitHub App
	if githubToken != "" {
		config.Token = githubToken
		return github.NewClientFromConfig(config)
	}

	// Crossplane provider credentials format (used in production)
	if githubCredentials != "" {
		config.Credentials = githubCredentials
		return github.NewClientFromConfig(config)
	}

	// Direct GitHub App authentication (for local dev/testing)
	if githubAppID != "" && githubInstallID != "" && githubAppKeyPath != "" {
		// Read private key from file
		privateKey, err := os.ReadFile(githubAppKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read GitHub App private key: %w", err)
		}

		config.AppID = githubAppID
		config.InstallationID = githubInstallID
		config.PrivateKey = privateKey

		return github.NewClientFromConfig(config)
	}

	return nil, fmt.Errorf("no valid authentication configured")
}

func getAuthMethod() string {
	if githubToken != "" {
		return "token"
	}
	if githubCredentials != "" {
		return "crossplane-credentials"
	}
	if githubAppID != "" && githubInstallID != "" && githubAppKeyPath != "" {
		return "github-app"
	}
	return "none"
}
