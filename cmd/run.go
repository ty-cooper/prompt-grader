package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	scoreCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().BoolP("concurrent", "C", false, "Run tests concurrently. (WARNING: may trigger rate limits quicker)")
	runCmd.PersistentFlags().StringVarP(&dataFile, "dataFile", "d", "", "directory location for csv data set.")
	runCmd.PersistentFlags().BoolP("listLlms", "L", false, "show available LLMs for use.")
	runCmd.PersistentFlags().BoolP("listTestOptions", "T", false, "show compatible test frameworks.")
	runCmd.PersistentFlags().StringSliceP("llms", "l", llms, "llms to use (ensure the relevant API keys are set).")
	runCmd.PersistentFlags().BoolP("noOutput", "N", false, "turn off HTML report generation.")
	runCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "directory location for HTML report output. (default is $HOME/.score/reports)")
	runCmd.PersistentFlags().StringVarP(&prompt, "prompt", "p", "", "prompt to test.")
	runCmd.PersistentFlags().StringVarP(&promptFile, "promptFile", "f", "", "directory location of a txt file with a prompt.")
	runCmd.PersistentFlags().StringVarP(&tests, "tests", "t", "", "directory location of a test file.")
	runCmd.PersistentFlags().BoolP("verbose", "V", false, "show all debug messages.")

	viper.BindPFlag("concurrent", runCmd.PersistentFlags().Lookup("concurrent"))
	viper.BindPFlag("dataFile", runCmd.PersistentFlags().Lookup("dataFile"))
	viper.BindPFlag("listTestOptions", runCmd.PersistentFlags().Lookup("listTestOptions"))
	viper.BindPFlag("listLlms", runCmd.PersistentFlags().Lookup("listLlms"))
	viper.BindPFlag("llms", runCmd.PersistentFlags().Lookup("llms"))
	viper.BindPFlag("noOutput", runCmd.PersistentFlags().Lookup("noOutput"))
	viper.BindPFlag("output", runCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("prompt", runCmd.PersistentFlags().Lookup("prompt"))
	viper.BindPFlag("promptFile", runCmd.PersistentFlags().Lookup("promptFile"))
	viper.BindPFlag("verbose", runCmd.PersistentFlags().Lookup("verbose"))
}

var (
	dataFile   string
	llms       []string
	outputFile string
	prompt     string
	promptFile string
	tests      string

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launch tests with provided prompt.",
		Long:  "Use the provided tests and prompt and begin scoring.",
		Run:   onRun,
	}
)

func onRun(cmd *cobra.Command, args []string) {
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
		results, seconds := SubmitDataAsync(10)

		// visualize the results and output to HTML file
		LoadResults(results, seconds)
		return
	}

	// load csv file if a data-set is provided and get llm responses
	results, seconds := SubmitData()

	// visualize the results and output to HTML file
	LoadResults(results, seconds)
}
