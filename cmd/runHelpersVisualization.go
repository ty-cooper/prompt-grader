package cmd

import (
	"fmt"
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/spf13/viper"
)

// TODO: visualization is a WIP
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

func GenerateBarChart(finalResult *FinalResult) {
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
