package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/3JoB/anthropic-sdk-go/v2"
	"github.com/3JoB/anthropic-sdk-go/v2/data"
	"github.com/3JoB/anthropic-sdk-go/v2/resp"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/sashabaranov/go-openai"
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

func GetClaudeResponse(c *anthropic.Client, prompt, model string) string {
	d, err := c.Send(&anthropic.Sender{
		Message: data.MessageModule{
			Human: prompt,
		},
		Sender: &resp.Sender{MaxToken: 1200},
	})

	cobra.CheckErr(err)
	return d.Response.Completion[1:]
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

func createResults(llmsObj *LLMs, e DataEntry, constructedPrompt string) []*Result {
	var results []*Result

	// add additional llm support here
	if llmsObj.openAi != nil {
		var res Result
		res.data = &e

		res.llm = llmsObj.openAi.llm

		response := GetGPTResponse(llmsObj.openAi.client, constructedPrompt, llmsObj.openAi.llm)

		if strings.ToLower(response) == "true" {
			res.passed = true
		} else if strings.ToLower(response) == "false" {
			res.passed = false
		} else {
			res.response = response
		}

		results = append(results, &res)
	}

	if llmsObj.anthropic != nil {
		var res Result
		res.data = &e

		res.llm = llmsObj.anthropic.llm

		response := GetClaudeResponse(llmsObj.anthropic.client, constructedPrompt, llmsObj.anthropic.llm)
		if strings.ToLower(response) == "true" {
			res.passed = true
		} else if strings.ToLower(response) == "false" {
			res.passed = false
		} else {
			res.response = response
		}

		results = append(results, &res)
	}

	return results
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

	r := csv.NewReader(f)

	for {
		var e DataEntry

		record, err := r.Read()
		if err == io.EOF {
			break
		}
		cobra.CheckErr(err)

		if record[0] == "passed" {
			continue
		}

		e.passed = strings.ToLower(record[0]) == "true"
		e.diffDelta, err = Base64Decode(record[1])
		cobra.CheckErr(err)

		constructedPrompt := createPrompt(e.diffDelta)

		rowResults := createResults(llmsObj, e, constructedPrompt)

		results = append(results, rowResults...)
	}

	seconds := time.Since(start)
	return results, seconds
}

// TODO: call the visualization function this will need to loop through each result and work on outputting to an HTML file
func LoadResults(results []*Result, seconds time.Duration) {
	var finalResult FinalResult
	total := len(results)
	var passed, failed, inconclusive int = 0, 0, 0

	for k, v := range results {
		if v.response == "" {
			fmt.Println(k, v.data.passed, v) // TODO: remove line
			if v.passed == v.data.passed {
				passed++
			} else {
				failed++
			}
		} else {
			inconclusive++
		}
	}

	fmt.Println() // TODO: remove line

	percentage := Round((float64(passed)/float64(total))*100, 0.05)
	percentageNoInconclusives := Round((float64(passed)/float64(total-inconclusive))*100, 0.05)

	finalResultFmt := fmt.Sprintf("Results achieved in %v\n\nThere were a total of %d tests ran.\n\tPassed: %d\n\tFailed: %d\n\tInconclusive: "+
		"%d\n\nScore: %.2f%%\nScore excluding inconclusive tests: %.2f%%\n",
		seconds, total, passed, failed, inconclusive, percentage, percentageNoInconclusives)

	fmt.Println(finalResultFmt) // TODO: turn this into a method on the object? just have it output this formatted string when you call the obj

	finalResult.failed = failed
	finalResult.total = total
	finalResult.passed = passed
	finalResult.inconclusive = inconclusive
	finalResult.percentage = percentage
	finalResult.percentageNoInconclusives = percentageNoInconclusives
	finalResult.seconds = seconds
	finalResult.results = results

	// TODO: check if no output flag is set and return early if so
	generateBarChart(&finalResult)
}

func generateBarItems(results []*Result, llm string) (int, int, int) {
	passed, failed, inconclusive := 0, 0, 0

	for _, res := range results {
		if res.llm == llm {
			if res.response == "" {
				if res.passed == res.data.passed {
					passed++
				} else {
					failed++
				}
			} else {
				inconclusive++
			}
		}
	}

	return passed, failed, inconclusive
}

func generateBarChart(finalResult *FinalResult) {
	bar := charts.NewBar()

	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "LLM Prompt Scoring",
		Subtitle: fmt.Sprintf("Score: %.2f%%, Score excluding inconclusive: %.2f%%, Time: %vs",
			finalResult.percentage, finalResult.percentageNoInconclusives, finalResult.seconds.Seconds()),
	}))

	llmModels := []string{"gpt-4", "claude-3-opus-20240229"} // TODO: make this dynamic
	passedItems := make([]opts.BarData, 0)
	failedItems := make([]opts.BarData, 0)
	inconclusiveItems := make([]opts.BarData, 0)

	for _, llm := range llmModels {
		passed, failed, inconclusive := generateBarItems(finalResult.results, llm)
		passedItems = append(passedItems, opts.BarData{Value: passed})
		failedItems = append(failedItems, opts.BarData{Value: failed})
		inconclusiveItems = append(inconclusiveItems, opts.BarData{Value: inconclusive})
	}

	bar.SetXAxis(llmModels).
		AddSeries("Passed", passedItems).
		AddSeries("Failed", failedItems).
		AddSeries("Inconclusive", inconclusiveItems)

	outputFile := viper.GetString("outputFile")

	f, _ := os.Create(outputFile)
	bar.Render(f)
}
