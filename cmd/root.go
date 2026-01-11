package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "gowatch",
	Short: "A self-hosted movie tracking application",
	Long:  `Gowatch is a web application for tracking movies, built with Go, HTMX, and Templ.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "./config.yaml", "config file (default is ./config.yaml)")
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}
