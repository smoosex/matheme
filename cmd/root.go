package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	curTheme string
	jsonOut  bool
)

var rootCmd = &cobra.Command{
	Use:   "matheme",
	Short: "A theme manager for macOS",
	Long: `Switch a base16 theme across terminals, editors, the window manager,
the system appearance and the desktop wallpaper.

Every command accepts --json so that scripts and GUIs can consume the result.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}

	var exitErr *exitError
	if errors.As(err, &exitErr) {
		if exitErr.err != nil {
			fmt.Fprintln(os.Stderr, exitErr.err)
		}
		os.Exit(exitErr.code)
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/matheme/config.toml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "print machine readable JSON")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(path.Join(home, ".config", "matheme"))
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config %s: %v\n", viper.ConfigFileUsed(), err)
	}
}
