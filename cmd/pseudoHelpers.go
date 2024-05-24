package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type PseudoDataEntry struct {
	external    string
	lesson      string
	passed      bool
	patch       string
	pseudoPatch string
	vuln        string
	reason      string
}

type PseudoResult struct {
	data     *PseudoDataEntry
	llm      string
	response string
	passed   bool
}

func (p *PseudoResult) GetData() interface{} {
	return p.data
}

func (p *PseudoResult) SetData(data interface{}) {
	p.data = data.(*PseudoDataEntry)
}

func (p *PseudoResult) GetPassed() bool {
	return p.passed
}

func (p *PseudoResult) SetPassed(passed bool) {
	p.passed = passed
}

func (p *PseudoResult) GetResponse() string {
	return p.response
}

func (p *PseudoResult) SetResponse(response string) {
	p.response = response
}

func (p *PseudoResult) GetLLM() string {
	return p.llm
}

func (p *PseudoResult) SetLLM(llm string) {
	p.llm = llm
}

func SubmitPseudoData() ([]GlobalResult, time.Duration) {
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
		var e PseudoDataEntry

		if record[0] == "Lesson" {
			continue
		}

		e.passed = strings.ToLower(record[4]) == "true"
		e.lesson = strings.ToLower(record[0])
		e.external = record[1]
		e.patch = record[2]
		e.pseudoPatch = record[3]
		e.reason = record[5]
		e.vuln = record[6]

		prompt := fmt.Sprintf("Expected solution: %s\nPseudocode solution: %s\nvulnerability being tested: %s", e.patch, e.pseudoPatch, e.passed)

		constructedPrompt := createPrompt(prompt)

		rowResults := createResults(llmsObj, e, constructedPrompt, bar)

		results = append(results, rowResults...)
	}

	seconds := time.Since(start)
	return results, seconds
}
