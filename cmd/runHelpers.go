package cmd

import (
	"context"
	"os"

	"github.com/3JoB/anthropic-sdk-go/v2"
	"github.com/3JoB/anthropic-sdk-go/v2/data"
	"github.com/3JoB/anthropic-sdk-go/v2/resp"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const UsageMsg = "Usage: score run [-p, --prompt] <prompt> [-l, --llms] <llms> || score run [-f, --prompt-file] <prompt.txt> [-l, --llms] <llms>\n"

func InitOpenAi() *openai.Client {
	OPENAI_API_KEY := os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		cobra.CompError("To use Openai services you need to have the `OPENAI_API_KEY` environment variable set.\n")
		os.Exit(1)
	}

	client := openai.NewClient(OPENAI_API_KEY)
	return client
}

func InitAnthropic() *anthropic.Client {
	ANTHROPIC_API_KEY := os.Getenv("ANTHROPIC_API_KEY")
	if ANTHROPIC_API_KEY == "" {
		cobra.CompError("To use Anthropic (Claude) services you need to have the `ANTHROPIC_API_KEY` environment variable set.\n")
		os.Exit(1)
	}

	client, err := anthropic.New(&anthropic.Config{Key: ANTHROPIC_API_KEY, DefaultModel: data.ModelFullInstant})
	cobra.CheckErr(err)

	return client

}

// Create go routine and channel for results
func GetGPTResponse(c *openai.Client, prompt, model string) string {
	resp, err := c.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	cobra.CheckErr(err)
	return resp.Choices[0].Message.Content
}

func GetClaudeResponse(c *anthropic.Client, prompt string) string {
	d, err := c.Send(&anthropic.Sender{
		Message: data.MessageModule{
			Human: prompt,
		},
		Sender: &resp.Sender{MaxToken: 1200},
	})

	cobra.CheckErr(err)
	return d.Response.Completion[1:] // TODO: String slice just for presentation reasons.. can remove later- though could be helpful for report generation
}

func GetLLMs() []string {
	return viper.GetStringSlice("supportedLLMs")
}

func GetTestOptions() []string {
	return viper.GetStringSlice("supportedTestFrameworks")
}

func SetOutputConfig() {
	if outputFile != "" {
		viper.Set("outputFile", outputFile)
	}
}

func CheckRunEmpty(args []string) bool {
	return len(args) == 0 && !viper.GetBool("listLlms") && !viper.GetBool("listTestOptions") &&
		len(viper.GetStringSlice("llms")) == 0 && viper.GetString("output") == "" &&
		viper.GetString("prompt") == "" && viper.GetString("promptFile") == "" &&
		viper.GetString("tests") == ""
}
