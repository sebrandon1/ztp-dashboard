package cmd

import (
	"os"
	"path/filepath"

	"github.com/sebrandon1/go-toolbox/cobrautil"
	"github.com/sebrandon1/go-toolbox/confkit"
	"github.com/sebrandon1/ztp-dashboard/internal/config"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "ztp-dashboard",
	Short: "ZTP Hub/Spoke Manager Dashboard",
	Long: `A purpose-built dashboard for managing ZTP provisioning pipelines.
Provides real-time status, AI-powered diagnostics via ollama, and
interactive controls for hub/spoke cluster management.`,
}

func Execute() error {
	return cobrautil.Execute(rootCmd)
}

func init() {
	defaultKubeconfig := confkit.EnvString("KUBECONFIG", "")
	if defaultKubeconfig == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultKubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	defaultLogFormat := confkit.EnvString("LOG_FORMAT", "text")
	defaultOllamaEndpoint := confkit.EnvString("OLLAMA_ENDPOINT", "http://localhost:11434")
	defaultOllamaModel := confkit.EnvString("OLLAMA_MODEL", "llama3.1")

	rootCmd.PersistentFlags().StringVar(&cfg.KubeConfig, "kubeconfig", defaultKubeconfig,
		"Path to kubeconfig file (env: KUBECONFIG)")
	rootCmd.PersistentFlags().IntVar(&cfg.Port, "port", 8080,
		"HTTP server port")
	rootCmd.PersistentFlags().StringVar(&cfg.LogFormat, "log-format", defaultLogFormat,
		"Log output format: text or json (env: LOG_FORMAT)")
	rootCmd.PersistentFlags().StringVar(&cfg.OllamaEndpoint, "ollama-endpoint", defaultOllamaEndpoint,
		"Ollama API endpoint (env: OLLAMA_ENDPOINT)")
	rootCmd.PersistentFlags().StringVar(&cfg.OllamaModel, "ollama-model", defaultOllamaModel,
		"Default ollama model (env: OLLAMA_MODEL)")
}
