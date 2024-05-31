package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func SubmitPseudoDataAsync(workerCount int) ([]GlobalResult, time.Duration) {
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
			var e PseudoDataEntry

			if record[0] == strings.ToLower("lesson") {
				continue
			}

			e.passed = strings.ToLower(record[3]) == "true"
			e.lesson = strings.ToLower(record[0])
			e.external = record[1]
			e.patch = record[2]
			e.reason = record[4]
			e.vuln = record[5]

			prompt := fmt.Sprintf("Vulnerable code: %s\nPatched Code: %s\nRequirements for passed test: %s", e.external, e.patch, e.vuln)

			constructedPrompt := createPrompt(prompt)

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
