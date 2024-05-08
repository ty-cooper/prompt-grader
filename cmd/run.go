package cmd

import (
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/3JoB/anthropic-sdk-go/v2"
)

func init() {
	scoreCmd.AddCommand(runCmd)

	runCmd.Flags().BoolP("listLlms", "L", false, "show available LLMs for use.")
	runCmd.Flags().BoolP("listTestOptions", "T", false, "show compatible test frameworks.")
	runCmd.Flags().StringSliceP("llms", "l", llms, "llms to use (ensure the relevant API keys are set).")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "directory location for HTML report output. (default is $HOME/.score/reports)")
	runCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "prompt to test.")
	runCmd.Flags().StringVarP(&promptFile, "promptFile", "f", "", "directory location of a txt file with a prompt.")
	runCmd.Flags().StringVarP(&tests, "tests", "t", "", "directory location of a test file.")

	viper.BindPFlag("listTestOptions", runCmd.Flags().Lookup("listTestOptions"))
	viper.BindPFlag("listLlms", runCmd.Flags().Lookup("listLlms"))
	viper.BindPFlag("llms", runCmd.Flags().Lookup("llms"))
	viper.BindPFlag("output", runCmd.Flags().Lookup("output"))
	viper.BindPFlag("prompt", runCmd.Flags().Lookup("prompt"))
	viper.BindPFlag("promptFile", runCmd.Flags().Lookup("promptFile"))
}

var (
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

	// Check config file for available LLMs
	if viper.GetBool("listLlms") {
		llms := GetLLMs()
		fmt.Println("\nAvailable LLMs loaded into config: \n")
		for _, llm := range llms {
			fmt.Println(llm)
		}
		os.Exit(0)
	}

	var prompt string
	argPromptFile := viper.GetString("promptFile")
	if prompt == "" && argPromptFile == "" {
		cobra.CompError(UsageMsg)
		os.Exit(1)
	}

	if argPromptFile != "" {
		dat, err := os.ReadFile(argPromptFile)
		cobra.CheckErr(err)
		prompt = string(dat)
	} else {
		prompt = viper.GetString("prompt")
	}

	// Check if llms passed contain OpenAI and Anthropic and initialize where needed
	var openAiClient *openai.Client
	var anthropicClient *anthropic.Client

	llms := viper.GetStringSlice("llms")
	if len(llms) == 0 {
		cobra.CompError(UsageMsg)
		os.Exit(1)
	}

	for _, llm := range llms {
		llm = strings.TrimSpace(llm)

		if llm[:3] == "gpt" {
			openAiClient = InitOpenAi()
		} else if llm[:6] == "claude" {
			anthropicClient = InitAnthropic()
		}
	}

	if openAiClient != nil {
		response := GetGPTResponse(openAiClient, prompt, "gpt-4")
		fmt.Println(response)
	}

	if anthropicClient != nil {
		responseAnthropic := GetClaudeResponse(anthropicClient, prompt)
		fmt.Println(responseAnthropic)
	}
}
