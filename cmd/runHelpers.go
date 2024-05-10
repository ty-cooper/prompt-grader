package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/3JoB/anthropic-sdk-go/v2"
	"github.com/3JoB/anthropic-sdk-go/v2/data"
	"github.com/3JoB/anthropic-sdk-go/v2/resp"
	"github.com/sashabaranov/go-openai"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type OpenAi struct {
	client *openai.Client
	llm    string
}

type Anthropic struct {
	client *anthropic.Client
	llm    string
}

type DataEntry struct {
	passed    bool
	diffDelta string
}

type Result struct {
	data     *DataEntry
	llm      string
	response string
	passed   bool
}

type LLMs struct {
	anthropic *Anthropic
	openAi    *OpenAi
}

type FinalResult struct {
	seconds                   time.Duration
	total                     int
	passed                    int
	failed                    int
	inconclusive              int
	percentage                float64
	percentageNoInconclusives float64
	results                   []*Result
}

func (fr FinalResult) String() string {
	return fmt.Sprintf("Results achieved in %v\n\nThere were a total of %d tests ran.\n\tPassed: %d\n\tFailed: %d\n\tInconclusive: "+
		"%d\n\nScore: %.2f%%\nScore excluding inconclusive tests: %.2f%%\n",
		fr.seconds, fr.total, fr.passed, fr.failed, fr.inconclusive, fr.percentage, fr.percentageNoInconclusives)
}

type RateLimitError struct {
	retryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit hit, retry after %s", e.retryAfter)
}

const UsageMsg = "Usage: score run [-p, --prompt] <prompt> [-l, --llms] <llms> || score run [-f, --prompt-file] <prompt.txt> [-l, --llms] <llms>\n"

func initOpenAi(llm string) *OpenAi {
	var openAiObj OpenAi

	OPENAI_API_KEY := os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		cobra.CompError("To use Openai services you need to have the `OPENAI_API_KEY` environment variable set.\n")
		os.Exit(1)
	}

	openAiObj.client = openai.NewClient(OPENAI_API_KEY)
	openAiObj.llm = llm

	return &openAiObj
}

func initAnthropic(llm string) *Anthropic {
	var anthropicObj Anthropic

	ANTHROPIC_API_KEY := os.Getenv("ANTHROPIC_API_KEY")
	if ANTHROPIC_API_KEY == "" {
		cobra.CompError("To use Anthropic (Claude) services you need to have the `ANTHROPIC_API_KEY` environment variable set.\n")
		os.Exit(1)
	}

	client, err := anthropic.New(&anthropic.Config{Key: ANTHROPIC_API_KEY, DefaultModel: data.ModelFullInstant})
	cobra.CheckErr(err)

	anthropicObj.client = client
	anthropicObj.llm = llm

	return &anthropicObj
}

func GetGPTResponse(c *openai.Client, prompt, model string) (string, error) {
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

	if err != nil {
		if strings.Contains(err.Error(), "Rate limit reached") {
			retryAfter := ParseRateLimitError(err.Error())
			return "", &RateLimitError{retryAfter}
		}
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func GetClaudeResponse(c *anthropic.Client, prompt, model string) (string, error) {
	d, err := c.Send(&anthropic.Sender{
		Message: data.MessageModule{
			Human: prompt,
		},
		Sender: &resp.Sender{MaxToken: 1200},
	})

	if err != nil {
		if strings.Contains(err.Error(), "rate limit") {
			return "", &RateLimitError{}
		}
		return "", err
	}

	return d.Response.Completion[1:], nil
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

func createPrompt(appendedData string) string {
	var prompt string
	argPrompt := viper.GetString("prompt")
	argPromptFile := viper.GetString("promptFile")
	if argPrompt == "" && argPromptFile == "" {
		cobra.CompError(UsageMsg)
		os.Exit(1)
	}

	if argPromptFile != "" {
		dat, err := os.ReadFile(argPromptFile)
		cobra.CheckErr(err)
		prompt = string(dat)
	} else {
		prompt = argPrompt
	}

	return prompt + appendedData
}

func InitLLMs() *LLMs {
	var llmsObj LLMs

	// add additional llm support here
	var openAiObj *OpenAi
	var anthropicObj *Anthropic

	llms := viper.GetStringSlice("llms")
	if len(llms) == 0 {
		cobra.CompError(UsageMsg)
		os.Exit(1)
	}

	for _, llm := range llms {
		llm = strings.TrimSpace(llm)

		if llm[:3] == "gpt" {
			openAiObj = initOpenAi(llm)
		} else if llm[:6] == "claude" {
			anthropicObj = initAnthropic(llm)
		}
	}

	llmsObj.anthropic = anthropicObj
	llmsObj.openAi = openAiObj

	return &llmsObj
}

func createResults(llmsObj *LLMs, e DataEntry, constructedPrompt string, bar *progressbar.ProgressBar) []*Result {
	var results []*Result

	// add additional llm support here
	if llmsObj.openAi != nil {
		var res Result
		res.data = &e

		res.llm = llmsObj.openAi.llm

		response, err := processWithRetries(func() (string, error) {
			return GetGPTResponse(llmsObj.openAi.client, constructedPrompt, llmsObj.openAi.llm)
		})
		cobra.CheckErr(err)

		if strings.ToLower(response) == "true" {
			res.passed = true
		} else if strings.ToLower(response) == "false" {
			res.passed = false
		} else {
			res.response = response
		}

		results = append(results, &res)
		bar.Add(1)
	}

	if llmsObj.anthropic != nil {
		var res Result
		res.data = &e

		res.llm = llmsObj.anthropic.llm

		response, err := processWithRetries(func() (string, error) {
			return GetClaudeResponse(llmsObj.anthropic.client, constructedPrompt, llmsObj.anthropic.llm)
		})
		cobra.CheckErr(err)

		if strings.ToLower(response) == "true" {
			res.passed = true
		} else if strings.ToLower(response) == "false" {
			res.passed = false
		} else {
			res.response = response
		}

		results = append(results, &res)
		bar.Add(1)
	}

	return results
}

func ParseRateLimitError(message string) time.Duration {
	// hard-coded for now
	return 5 * time.Second
}

func processWithRetries(request func() (string, error)) (string, error) {
	for {
		response, err := request()
		if err != nil {
			if rateLimitError, ok := err.(*RateLimitError); ok {
				if viper.GetBool("verbose") {
					fmt.Printf("\nRate limit hit... trying again in %s", rateLimitError.retryAfter)
				}
				time.Sleep(rateLimitError.retryAfter)
				continue
			} else {
				return "", err
			}
		}
		return response, nil
	}
}

func SubmitData() ([]*Result, time.Duration) {
	start := time.Now()

	llmsObj := InitLLMs()
	var results []*Result

	dataFile := viper.GetString("dataFile")
	if dataFile == "" {
		return nil, time.Since(start)
	}

	f, err := os.Open(dataFile)
	cobra.CheckErr(err)
	defer f.Close()

	r, err := csv.NewReader(f).ReadAll()
	cobra.CheckErr(err)
	bar := progressbar.Default(int64(len(r))-1, "running tests")

	for _, record := range r {
		var e DataEntry

		if record[0] == "passed" {
			continue
		}

		e.passed = strings.ToLower(record[0]) == "true"
		e.diffDelta, err = Base64Decode(record[1])
		cobra.CheckErr(err)

		constructedPrompt := createPrompt(e.diffDelta)

		rowResults := createResults(llmsObj, e, constructedPrompt, bar)

		results = append(results, rowResults...)
	}

	seconds := time.Since(start)
	return results, seconds
}

func LoadResults(results []*Result, seconds time.Duration) {
	var finalResult FinalResult
	total := len(results)
	var passed, failed, inconclusive int = 0, 0, 0

	for k, v := range results {
		if v.response == "" {
			if viper.GetBool("verbose") {
				fmt.Println(k, v.data.passed, v)
			}

			if v.passed == v.data.passed {
				passed++
			} else {
				failed++
			}
		} else {
			inconclusive++
		}
	}

	if viper.GetBool("verbose") {
		fmt.Println()
	}

	percentage := Round((float64(passed)/float64(total))*100, 0.05)
	percentageNoInconclusives := Round((float64(passed)/float64(total-inconclusive))*100, 0.05)

	finalResult.failed = failed
	finalResult.total = total
	finalResult.passed = passed
	finalResult.inconclusive = inconclusive
	finalResult.percentage = percentage
	finalResult.percentageNoInconclusives = percentageNoInconclusives
	finalResult.seconds = seconds
	finalResult.results = results

	fmt.Println(finalResult)

	if !viper.GetBool("noOutput") {
		GenerateBarChart(&finalResult)
	}
}
