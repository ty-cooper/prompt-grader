package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	runCmd.AddCommand(pseudoCmd)

	pseudoCmd.Flags().BoolP("concurrent", "C", false, "Run tests concurrently. (WARNING: may trigger rate limits quicker)")
	pseudoCmd.Flags().StringVarP(&dataFilePseudo, "dataFile", "d", "", "directory location for csv data set.")
	pseudoCmd.Flags().BoolP("listLlms", "L", false, "show available LLMs for use.")
	pseudoCmd.Flags().BoolP("listTestOptions", "T", false, "show compatible test frameworks.")
	pseudoCmd.Flags().StringSliceP("llms", "l", llms, "llms to use (ensure the relevant API keys are set).")
	pseudoCmd.Flags().BoolP("noOutput", "N", false, "turn off HTML report generation.")
	pseudoCmd.Flags().StringVarP(&outputFile, "output", "o", "", "directory location for HTML report output. (default is $HOME/.score/reports)")
	pseudoCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "prompt to test.")
	pseudoCmd.Flags().StringVarP(&promptFile, "promptFile", "f", "", "directory location of a txt file with a prompt.")
	pseudoCmd.Flags().StringVarP(&tests, "tests", "t", "", "directory location of a test file.")
	pseudoCmd.Flags().BoolP("verbose", "V", false, "show all debug messages.")

	viper.BindPFlag("concurrent", runCmd.Flags().Lookup("concurrent"))
	viper.BindPFlag("dataFile", runCmd.Flags().Lookup("dataFile"))
	viper.BindPFlag("listTestOptions", runCmd.Flags().Lookup("listTestOptions"))
	viper.BindPFlag("listLlms", runCmd.Flags().Lookup("listLlms"))
	viper.BindPFlag("llms", runCmd.Flags().Lookup("llms"))
	viper.BindPFlag("noOutput", runCmd.Flags().Lookup("noOutput"))
	viper.BindPFlag("output", runCmd.Flags().Lookup("output"))
	viper.BindPFlag("prompt", runCmd.Flags().Lookup("prompt"))
	viper.BindPFlag("promptFile", runCmd.Flags().Lookup("promptFile"))
	viper.BindPFlag("verbose", runCmd.Flags().Lookup("verbose"))
}

var (
	dataFilePseudo string

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
