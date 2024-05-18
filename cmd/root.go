/*
Copyright © 2024 weiserchen

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nikolaydubina/go-instrument/processor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-instrument <path>...",
	Short: "A simple instrumentation tool for tracing application data.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filenames, err := listFileNames(args)
		if err != nil {
			return err
		}

		tracePattern := processor.DefaultTracePattern
		config := processor.TraceConfig{
			App:           viper.GetString("app"),
			Overwrite:     viper.GetBool("overwrite"),
			DefaultSelect: viper.GetBool("default-select"),
			SkipGenerated: viper.GetBool("skip-generated"),
		}

		fmt.Println(config)

		p := processor.NewParallelTraceProcessor(viper.GetInt("parallel"), tracePattern)
		if err := p.Process(filenames, config); err != nil {
			return err
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-instrument.yaml)")
	rootCmd.Flags().IntP("parallel", "j", 1, "The number of parallel worker")
	rootCmd.Flags().StringP("app", "n", "app", "Application name")
	rootCmd.Flags().BoolP("overwrite", "w", false, "Overwrite original files")
	rootCmd.Flags().BoolP("default-select", "s", true, "Instrument all by default")
	rootCmd.Flags().BoolP("skip-generated", "k", false, "Skip generated files")

	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("INSTRA")
	viper.BindPFlag("parallel", rootCmd.Flags().Lookup("parallel"))
	viper.BindPFlag("app", rootCmd.Flags().Lookup("app"))
	viper.BindPFlag("overwrite", rootCmd.Flags().Lookup("overwrite"))
	viper.BindPFlag("default-select", rootCmd.Flags().Lookup("default-select"))
	viper.BindPFlag("skip-generated", rootCmd.Flags().Lookup("skip-generated"))
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

		// Search config in home directory with name ".go-instrument" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".go-instrument")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func listFileNames(args []string) ([]string, error) {
	var filenames []string

	for _, f := range args {
		fileInfo, err := os.Stat(f)
		if err != nil {
			return []string{}, err
		}

		if fileInfo.IsDir() {
			err := filepath.WalkDir(f, func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				if filepath.Ext(path) == ".go" {
					filenames = append(filenames, path)
				}
				return nil
			})
			if err != nil {
				return []string{}, err
			}
		} else {
			filenames = append(filenames, f)
		}
	}

	return filenames, nil
}
