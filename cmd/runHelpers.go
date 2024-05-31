package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/3JoB/anthropic-sdk-go/v2"
	"github.com/3JoB/anthropic-sdk-go/v2/data"
	"github.com/3JoB/anthropic-sdk-go/v2/resp"
	"github.com/sashabaranov/go-openai"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	. "github.com/theplant/htmlgo"
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
	results                   []GlobalResult
}

type GlobalResult interface {
	GetData() interface{}
	SetData(data interface{})
	GetPassed() bool
	SetPassed(passed bool)
	GetResponse() string
	SetResponse(response string)
	GetLLM() string
	SetLLM(llm string)
}

func (r *Result) GetData() interface{} {
	return r.data
}

func (r *Result) SetData(data interface{}) {
	r.data = data.(*DataEntry)
}

func (r *Result) GetPassed() bool {
	return r.passed
}

func (r *Result) SetPassed(passed bool) {
	r.passed = passed
}

func (r *Result) GetResponse() string {
	return r.response
}

func (r *Result) SetResponse(response string) {
	r.response = response
}

func (r *Result) GetLLM() string {
	return r.llm
}

func (r *Result) SetLLM(llm string) {
	r.llm = llm
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

func createResults(llmsObj *LLMs, e interface{}, constructedPrompt string, bar *progressbar.ProgressBar) []GlobalResult {
	// add additional llm support here
	var results []GlobalResult
	var res GlobalResult

	switch v := e.(type) {
	case DataEntry:
		res = &Result{}
		res.SetData(&v)
	case PseudoDataEntry:
		res = &PseudoResult{}
		res.SetData(&v)
	default:
		return nil
	}

	if llmsObj.openAi != nil {
		res.SetLLM(llmsObj.openAi.llm)

		response, err := processWithRetries(func() (string, error) {
			return GetGPTResponse(llmsObj.openAi.client, constructedPrompt, llmsObj.openAi.llm)
		})
		cobra.CheckErr(err)

		if strings.ToLower(response) == "true" {
			res.SetPassed(true)
		} else if strings.ToLower(response) == "false" {
			res.SetPassed(false)
		} else {
			res.SetResponse(response)
		}

		results = append(results, res)
		bar.Add(1)
	}

	if llmsObj.anthropic != nil {
		res.SetLLM(llmsObj.anthropic.llm)

		response, err := processWithRetries(func() (string, error) {
			return GetClaudeResponse(llmsObj.anthropic.client, constructedPrompt, llmsObj.anthropic.llm)
		})
		cobra.CheckErr(err)

		if strings.ToLower(response) == "true" {
			res.SetPassed(true)
		} else if strings.ToLower(response) == "false" {
			res.SetPassed(false)
		} else {
			res.SetResponse(response)
		}

		results = append(results, res)
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

func SubmitData() ([]GlobalResult, time.Duration) {
	start := time.Now()

	llmsObj := InitLLMs()
	var results []GlobalResult

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

		if rowResults := createResults(llmsObj, e, constructedPrompt, bar); rowResults == nil {
			cobra.CompError("Invalid type passed to createResults().\n")
		} else {
			results = append(results, rowResults...)
		}
	}

	seconds := time.Since(start)
	return results, seconds
}

func LoadResults(results []GlobalResult, seconds time.Duration) {
	var finalResult FinalResult
	total := len(results)
	var passed, failed, inconclusive int = 0, 0, 0

	for k, v := range results {
		if v.GetResponse() == "" {
			switch d := v.GetData().(type) {
			case *DataEntry:
				if viper.GetBool("verbose") {
					fmt.Println(k, d.passed, v)
				}

				if v.GetPassed() == d.passed {
					passed++
				} else {
					failed++
				}
			case *PseudoDataEntry:
				if viper.GetBool("verbose") {
					fmt.Println(k, d.passed, v)
				}

				if v.GetPassed() == d.passed {
					passed++
				} else {
					failed++
				}
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
		GenerateHTML(&finalResult)
	}
}

func GenerateHTML(f *FinalResult) {
	outputFilePath := os.ExpandEnv(viper.GetString("outputFile"))

	now := time.Now()
	timeSuffix := now.Format("01-02-2006_150405")

	ext := filepath.Ext(outputFilePath)
	base := outputFilePath[:len(outputFilePath)-len(ext)]
	outputFilePath = fmt.Sprintf("%s-%s%s", base, timeSuffix, ext)

	outputDir := filepath.Dir(outputFilePath)
	err := os.MkdirAll(outputDir, os.ModePerm)
	cobra.CheckErr(err)

	outputFile, err := os.Create(outputFilePath)
	cobra.CheckErr(err)
	defer outputFile.Close()

	styles := `
		body {
			font-family: Arial, sans-serif;
			margin: 20px;
		}

		.content {
			padding: 0 18px;
			display: none;
			overflow: hidden;
			background-color: #f1f1f1;
		}

		.active, .collapsible:hover {
			background-color: #ccc;
		}
		  
		.collapsible {
			background-color: #eee;
			color: #444;
			cursor: pointer;
			padding: 18px;
			width: 100%;
			border: none;
			text-align: left;
			outline: none;
			font-size: 15px;
			margin: 1em 0 0 0;
		}
	`

	script := `
		var coll = document.getElementsByClassName("collapsible");
		var i;

		for (i = 0; i < coll.length; i++) {
		coll[i].addEventListener("click", function() {
			this.classList.toggle("active");
			var content = this.nextElementSibling;
			if (content.style.display === "block") {
			content.style.display = "none";
			} else {
			content.style.display = "block";
			}
		});
		}
	`

	detailsDiv := Div(H2("Failed Test Details"))

	for _, result := range f.results {
		switch data := result.GetData().(type) {
		case *DataEntry:
			if result.GetPassed() == data.passed {
				continue
			}

			resultParagraph := P(
				Textf("User-Passed: %t", data.passed),
				Br(),
				Textf("Passed: %t", result.GetPassed()),
				Br(),
				Textf("Diff Delta: %s", data.diffDelta),
				Br(),
				Textf("LLM: %s", result.GetLLM()),
				Br(),
			)
			detailsDiv.AppendChildren(resultParagraph)
		case *PseudoDataEntry:
			if result.GetPassed() == data.passed {
				continue
			}

			resultParagraph := P(
				Textf("LLM: %s said %t", result.GetLLM(), result.GetPassed()),
				Br(),
				Textf("Expected Result: %t", data.passed),
				Br(),
				Iff(len(data.reason) > 0, func() HTMLComponent {
					return Textf("Reason: %s", data.reason)
				}),
				Iff(len(result.GetResponse()) > 0, func() HTMLComponent {
					return Textf("LLM Reason: %s", result.GetResponse())
				}),
			)

			dataDiv := Div(
				Button("External File").Type("button").Class("collapsible"),
				Div(
					Pre(data.external),
				).Class("content"),
				Button("Patch File").Type("button").Class("collapsible"),
				Div(
					Pre(data.patch),
				).Class("content"),
				Hr(),
			)

			detailsDiv.AppendChildren(resultParagraph)
			detailsDiv.AppendChildren(dataDiv)
		}
	}

	comp := HTML(
		Head(
			Meta().Charset("utf8"),
			Title("Score report"),
			Style(styles),
		),
		Body(
			H2("Overview"),
			P(
				Textf("%v", time.Now().Format("01-02-2006, 15:04:05")),
			),
			P(
				Textf("%d tests ran in %v\n", f.total, f.seconds),
			),
			P(
				Textf("Passed: %d", f.passed),
				Br(),
				Textf("Failed: %d", f.failed),
				Br(),
				Textf("Total: %d", f.total),
				Br(),
				Textf("Inconclusive: %d", f.inconclusive),
			),
			P(
				Textf("Score: %.2f%%", f.percentage),
				Br(),
				Textf("Score excluding inconclusives: %.2f%%", f.percentageNoInconclusives),
				Br(),
			),

			detailsDiv,
		),
		Script(script),
	)

	Fprint(outputFile, comp, context.TODO())
	fmt.Println("File written to:", outputFilePath)
}
