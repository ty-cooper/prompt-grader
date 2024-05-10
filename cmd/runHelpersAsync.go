package cmd

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Job struct {
	llmsObj           *LLMs
	dataEntry         DataEntry
	constructedPrompt string
}

func worker(jobs <-chan Job, results chan<- *Result) {
	for job := range jobs {
		if job.llmsObj.openAi != nil {
			var res Result
			res.data = &job.dataEntry
			res.llm = job.llmsObj.openAi.llm
			response := GetGPTResponse(job.llmsObj.openAi.client, job.constructedPrompt, job.llmsObj.openAi.llm)

			if strings.ToLower(response) == "true" {
				res.passed = true
			} else if strings.ToLower(response) == "false" {
				res.passed = false
			} else {
				res.response = response
			}

			results <- &res
		}

		if job.llmsObj.anthropic != nil {
			var res Result
			res.data = &job.dataEntry

			res.llm = job.llmsObj.anthropic.llm

			response := GetClaudeResponse(job.llmsObj.anthropic.client, job.constructedPrompt, job.llmsObj.anthropic.llm)
			if strings.ToLower(response) == "true" {
				res.passed = true
			} else if strings.ToLower(response) == "false" {
				res.passed = false
			} else {
				res.response = response
			}

			results <- &res

		}
	}
}

func SubmitDataAsync(workerCount int) ([]*Result, time.Duration) {
	start := time.Now()
	llmsObj := InitLLMs()
	var resultsList []*Result

	dataFile := viper.GetString("dataFile")
	if dataFile == "" {
		return nil, time.Since(start)
	}

	f, err := os.Open(dataFile)
	cobra.CheckErr(err)
	defer f.Close()

	r := csv.NewReader(f)
	jobs := make(chan Job, workerCount)
	results := make(chan *Result)
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(jobs, results)
		}()
	}

	go func() {
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			cobra.CheckErr(err)

			if record[0] == "passed" {
				continue
			}

			var e DataEntry
			e.passed = strings.ToLower(record[0]) == "true"
			e.diffDelta, err = Base64Decode(record[1])
			cobra.CheckErr(err)

			constructedPrompt := createPrompt(e.diffDelta)
			jobs <- Job{llmsObj, e, constructedPrompt}
		}
		close(jobs)
	}()

	go func() {
		for res := range results {
			resultsList = append(resultsList, res)
		}
	}()

	wg.Wait()
	close(results)

	seconds := time.Since(start)
	return resultsList, seconds
}
