package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runCmd.AddCommand(pseudoCmd)
}

var (
	pseudoCmd = &cobra.Command{
		Use:   "pseudo",
		Short: "Run pseudocode tests",
		Long:  "Use the provided pseudocode data and run tests against it.",
		Run:   onRunPseudo,
	}
)

func onRunPseudo(cmd *cobra.Command, args []string) {
	if CheckRunEmpty(args) {
		cmd.Help()
		os.Exit(1)
	}

	if viper.GetString("output") != "" {
		SetOutputConfig()
	}

	if viper.GetBool("listTestOptions") {
		testOptions := GetTestOptions()
		fmt.Println("\nCompatible test frameworks: \n")
		for _, framework := range testOptions {
			fmt.Println(framework)
		}
		os.Exit(0)
	}

	// check config file for available LLMs
	if viper.GetBool("listLlms") {
		availableLlms := GetLLMs()
		fmt.Println("\nAvailable LLMs loaded into config: \n")
		for _, llm := range availableLlms {
			fmt.Println(llm)
		}
		os.Exit(0)
	}

	if viper.GetBool("concurrent") {
		// load csv file if a data-set is provided and get llm responses
		results, seconds := SubmitPseudoDataAsync(10)

		// visualize the results and output to HTML file
		LoadResults(results, seconds)
		return
	}

	// load csv file if a data-set is provided and get llm responses
	results, seconds := SubmitPseudoData()

	// visualize the results and output to HTML file
	LoadResults(results, seconds)
}
