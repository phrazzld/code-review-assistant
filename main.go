package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v39/github"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type Config struct {
	GithubToken string
	OpenAIKey   string
}

func main() {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "review-assistant [owner/repo] [pr_number]",
		Short: "A code review assistant using LLM",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			owner, repo, found := strings.Cut(args[0], "/")
			if !found {
				log.Fatalf("Invalid repository format. Use 'owner/repo'")
			}
			prNumber := args[1]

			runReviewAssistant(owner, repo, prNumber, &config)
		},
	}

	rootCmd.Flags().StringVar(&config.GithubToken, "github-token", "", "GitHub API Token")
	rootCmd.Flags().StringVar(&config.OpenAIKey, "openai-key", "", "OpenAI API Key")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runReviewAssistant(owner, repo, prNumber string, config *Config) {
	ctx := context.Background()

	// Initialize GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// convert prNumber to int
	prNumberInt, err := strconv.Atoi(prNumber)
	if err != nil {
		log.Fatalf("Error converting PR number to int: %v", err)
	}

	// Fetch PR diff
	diff, _, err := client.PullRequests.GetRaw(ctx, owner, repo, prNumberInt, github.RawOptions{Type: github.Diff})
	if err != nil {
		log.Fatalf("Error fetching PR diff: %v", err)
	}

	// Generate review insights using OpenAI
	insights := generateReviewInsights(diff, config.OpenAIKey)

	// Print insights
	fmt.Println("Code Review Insights:")
	fmt.Println(insights)
}

func generateReviewInsights(diff, apiKey string) string {
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `You are a code review assistant. Analyze the given diff and provide insights
					to help a developer conduct a thorough code review. Focus on potential issues,
					suggestions for improvement, and questions that should be asked during the review.`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Please review this diff and provide insights:\n\n%s", diff),
				},
			},
			MaxTokens: 1000,
		},
	)

	if err != nil {
		log.Printf("Error generating review insights: %v", err)
		return "Error generating review insights"
	}

	return resp.Choices[0].Message.Content
}
