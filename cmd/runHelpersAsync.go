package cmd

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Job struct {
	llmsObj           *LLMs
	dataEntry         interface{}
	constructedPrompt string
}

func worker(jobs <-chan Job, results chan<- *GlobalResult, bar *progressbar.ProgressBar) {
	for job := range jobs {
		for {
			res, err := processJob(job)
			if err != nil {
				var rateLimitError *RateLimitError
				if errors.As(err, &rateLimitError) {
					if viper.GetBool("verbose") {
						fmt.Printf("\nRate limit hit... trying again in %s", rateLimitError.retryAfter)
					}
					time.Sleep(rateLimitError.retryAfter)
					continue
				} else {
					cobra.CompErrorln(err.Error())
				}
			} else {
				results <- res
				bar.Add(1)
				break
			}
		}
	}
}

func processJob(job Job) (*GlobalResult, error) {
	var res GlobalResult

	switch v := job.dataEntry.(type) {
	case DataEntry:
		res = &Result{}
		res.SetData(&v)
	case PseudoDataEntry:
		res = &PseudoResult{}
		res.SetData(&v)
	default:
		return nil, fmt.Errorf("Invalid type passed to processJob()")
	}

	if job.llmsObj.openAi != nil {
		res.SetLLM(job.llmsObj.openAi.llm)
		response, err := GetGPTResponse(job.llmsObj.openAi.client, job.constructedPrompt, job.llmsObj.openAi.llm)
		if err != nil {
			return nil, err
		}

		if strings.ToLower(response) == "true" {
			res.SetPassed(true)
		} else if strings.ToLower(response) == "false" {
			res.SetPassed(false)
		} else {
			res.SetResponse(response)
		}

		return &res, nil
	}

	if job.llmsObj.anthropic != nil {
		res.SetLLM(job.llmsObj.anthropic.llm)
		response, err := GetClaudeResponse(job.llmsObj.anthropic.client, job.constructedPrompt, job.llmsObj.anthropic.llm)
		if err != nil {
			return nil, err
		}

		if strings.ToLower(response) == "true" {
			res.SetPassed(true)
		} else if strings.ToLower(response) == "false" {
			res.SetPassed(false)
		} else {
			res.SetResponse(response)
		}

		return &res, nil
	}

	return nil, nil
}

func SubmitDataAsync(workerCount int) ([]GlobalResult, time.Duration) {
	start := time.Now()

	llmsObj := InitLLMs()
	var resultsList []GlobalResult

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
	jobs := make(chan Job, workerCount)
	results := make(chan *GlobalResult)
	var wg sync.WaitGroup

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(jobs, results, bar)
		}()
	}

	go func() {
		for _, record := range r {
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
			resultsList = append(resultsList, *res)
		}
	}()

	wg.Wait()
	close(results)

	seconds := time.Since(start)
	return resultsList, seconds
}
