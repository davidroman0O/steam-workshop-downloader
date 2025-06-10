package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	configDir   string
	downloadDir string
	steamcmdDir string
	verbose     bool
)

// Build information
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildTime    = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "workshop",
	Short: "A CLI tool to download Steam Workshop items",
	Long: `Workshop is a CLI application built with Cobra and Viper
that allows you to download Steam Workshop items using SteamCMD.

Features:
- Download workshop items by URL or ID
- Install and manage SteamCMD
- Configurable download directories
- Support for different Steam apps`,
	Version: buildVersion,
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(version, commit, buildTimeStr string) {
	buildVersion = version
	buildCommit = commit
	buildTime = buildTimeStr
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildTimeStr)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.workshop.yaml)")
	rootCmd.PersistentFlags().StringVar(&downloadDir, "download-dir", "", "directory to download workshop items to")
	rootCmd.PersistentFlags().StringVar(&steamcmdDir, "steamcmd-dir", "", "directory where SteamCMD is installed")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("download_dir", rootCmd.PersistentFlags().Lookup("download-dir"))
	viper.BindPFlag("steamcmd_dir", rootCmd.PersistentFlags().Lookup("steamcmd-dir"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".workshop" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".workshop")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Set default values
	setDefaults()
}

func setDefaults() {
	home, _ := os.UserHomeDir()

	// Set default download directory
	if viper.GetString("download_dir") == "" {
		defaultDownloadDir := filepath.Join(home, "Downloads", "Steam-Workshop")
		viper.SetDefault("download_dir", defaultDownloadDir)
	}

	// Set default SteamCMD directory
	if viper.GetString("steamcmd_dir") == "" {
		defaultSteamCMDDir := filepath.Join(home, ".workshop", "steamcmd")
		viper.SetDefault("steamcmd_dir", defaultSteamCMDDir)
	}

	// Set default cache directory
	defaultCacheDir := filepath.Join(home, ".workshop", "cache")
	viper.SetDefault("cache_dir", defaultCacheDir)

	// Set default for anonymous login
	viper.SetDefault("anonymous_login", true)
	viper.SetDefault("auto_extract", true)
}
