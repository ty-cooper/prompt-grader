package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	scoreCmd = &cobra.Command{
		Use:   "score",
		Short: "Score is a fast and easy way to test prompt accuracy for LLMs",
		Long:  `A Fast and Scalable way to test LLM prompt accuracy against a set of tests provided`,
		Run:   onRootRun,
	}
)

func onRootRun(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}
}

func Execute() error {
	// TODO: delete this
	// err := doc.GenMarkdownTree(scoreCmd, "./docs")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	return scoreCmd.Execute()
}

func init() {
	cobra.OnInitialize(InitConfig)

	scoreCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml).")

	viper.SetDefault("license", "apache")
	viper.SetDefault("useViper", true)
}
