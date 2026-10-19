package cmd

import (
	"github.com/marcosalvi-01/gowatch/internal/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gowatch server",
	Long:  `Start the HTTP server for the gowatch movie tracking application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := server.Config{
			Port:                 viper.GetString("port"),
			Timeout:              viper.GetDuration("request_timeout"),
			DBPath:               viper.GetString("db_path"),
			DBName:               viper.GetString("db_name"),
			SeerrBaseURL:         viper.GetString("seerr_base_url"),
			SeerrAPIKey:          viper.GetString("seerr_api_key"),
			CacheTTL:             viper.GetDuration("cache_ttl"),
			ImageCacheTTL:        viper.GetDuration("image_cache_ttl"),
			ImageCleanupInterval: viper.GetDuration("image_cleanup_interval"),
			SessionExpiry:        viper.GetDuration("session_expiry"),
			ShutdownTimeout:      viper.GetDuration("shutdown_timeout"),
			HTTPS:                viper.GetBool("https"),
			AdminDefaultPassword: viper.GetString("admin_default_password"),
		}
		if err := cfg.Validate(); err != nil {
			panic(err)
		}
		server.RunServer(cfg)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Flags for start command
	startCmd.Flags().String("port", "8080", "Port to run the server on")
	startCmd.Flags().String("db-path", "/var/lib/gowatch", "Path to the database directory")
	startCmd.Flags().String("db-name", "db.db", "Name of the database file")

	// Bind flags to viper
	if err := viper.BindPFlag("port", startCmd.Flags().Lookup("port")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("db_path", startCmd.Flags().Lookup("db-path")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("db_name", startCmd.Flags().Lookup("db-name")); err != nil {
		panic(err)
	}

	// Set defaults
	viper.SetDefault("request_timeout", "30s")
	viper.SetDefault("cache_ttl", "168h")
	viper.SetDefault("image_cache_ttl", "720h")
	viper.SetDefault("image_cleanup_interval", "48h")
	viper.SetDefault("session_expiry", "24h")
	viper.SetDefault("shutdown_timeout", "30s")
	viper.SetDefault("https", false)
	viper.SetDefault("admin_default_password", "Welcome123!")
}
