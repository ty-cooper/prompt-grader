package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	

)

var rootCmd = &cobra.Command{
	Use: "prgrade",
	Short: "Prompt grader is a fast and easy way to test prompt accuracy for GPT-4",
	Long: `A Fast and Scalable way to test GPT-4 prompt accuracy against a set of tests provided`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do stuff here
	},
}