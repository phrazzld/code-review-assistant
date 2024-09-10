package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/fatih/color"
	"github.com/google/go-github/v39/github"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type Config struct {
	GithubToken string
	OpenAIKey   string
}

type PRMetadata struct {
	Title  string
	Author string
	Files  int
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

	spinner := NewSpinner("Fetching PR data and generating insights...")
	spinner.Start()

	// Initialize GitHub client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Fetch PR metadata
	metadata, err := getPRMetadata(ctx, client, owner, repo, prNumber)
	if err != nil {
		spinner.Stop()
		log.Fatalf("Error fetching PR metadata: %v", err)
	}

	// convert prNumber to int
	prNumberInt, err := strconv.Atoi(prNumber)
	if err != nil {
		spinner.Stop()
		log.Fatalf("Error converting PR number to int: %v", err.Error())
	}

	// Fetch PR diff
	diff, _, err := client.PullRequests.GetRaw(ctx, owner, repo, prNumberInt, github.RawOptions{Type: github.Diff})
	if err != nil {
		spinner.Stop()
		log.Fatalf("Error fetching PR diff: %v", err)
	}

	// Generate review insights using OpenAI
	insights := generateReviewInsights(diff, config)

	spinner.Stop()

	// Print PR metadata and insights
	printReview(metadata, insights)
}

func getPRMetadata(ctx context.Context, client *github.Client, owner, repo, prNumber string) (*PRMetadata, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, atoi(prNumber))
	if err != nil {
		return nil, err
	}

	files, _, err := client.PullRequests.ListFiles(ctx, owner, repo, atoi(prNumber), nil)
	if err != nil {
		return nil, err
	}

	return &PRMetadata{
		Title:  pr.GetTitle(),
		Author: pr.GetUser().GetLogin(),
		Files:  len(files),
	}, nil
}

func generateReviewInsights(diff string, config *Config) []string {
	client := openai.NewClient(config.OpenAIKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `<role>You are an expert software engineer and architect with extensive experience in code review. Your task is to analyze pull request diffs and provide actionable insights to improve code quality, maintainability, and performance.</role>

<guidelines>
- Provide specific, actionable feedback with line numbers when possible
- Consider best practices, design patterns, and potential edge cases
- Highlight both areas for improvement and commendable code practices
- You are not conducting a code review. You are helping another developer conduct a code review.
</guidelines>

<personality>
- No-nonsense
- Concise
- Pragmatic
- Cuts straight to the chase
- Slightly eccentric
- High agency
- Encyclopedic knowledge
- Security-conscious and slightly paranoid
</personality>`,
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
					Analyze the following pull request diff. Focus on the most impactful insights that will help improve the overall quality of the code.

<diff>
%s
</diff>`, diff),
				},
			},
		},
	)

	if err != nil {
		log.Printf("Error generating review insights: %v", err)
		return []string{"Error generating review insights"}
	}

	return strings.Split(resp.Choices[0].Message.Content, "\n")
}

func printReview(metadata *PRMetadata, insights []string) {
	headerColor := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	subHeaderColor := color.New(color.FgCyan).SprintFunc()

	fmt.Println(headerColor("=== Pull Request Metadata ==="))
	fmt.Printf("%s %s\n", subHeaderColor("Title:"), metadata.Title)
	fmt.Printf("%s %s\n", subHeaderColor("Author:"), metadata.Author)
	fmt.Printf("%s %d\n", subHeaderColor("Files Changed:"), metadata.Files)
	fmt.Println()

	fmt.Println(headerColor("=== Code Review Insights ==="))
	for _, insight := range insights {
		printInsight(insight)
	}
}

func printInsight(insight string) {
	parts := strings.SplitN(insight, ":", 2)
	if len(parts) != 2 {
		fmt.Println(insight)
		return
	}

	category := strings.TrimSpace(parts[0])
	content := strings.TrimSpace(parts[1])

	var categoryColor func(a ...interface{}) string
	switch category {
	case "Critical":
		categoryColor = color.New(color.FgHiRed, color.Bold).SprintFunc()
	case "Warning":
		categoryColor = color.New(color.FgHiYellow, color.Bold).SprintFunc()
	case "Suggestion":
		categoryColor = color.New(color.FgHiGreen, color.Bold).SprintFunc()
	default:
		categoryColor = color.New(color.FgHiWhite, color.Bold).SprintFunc()
	}

	fmt.Printf("%s: %s\n", categoryColor(category), content)

	// Check if there's a code snippet and highlight it
	if strings.Contains(content, "```") {
		highlightCodeSnippet(content)
	}
}

func highlightCodeSnippet(content string) {
	parts := strings.SplitN(content, "```", 3)
	if len(parts) != 3 {
		return
	}

	language := strings.TrimSpace(parts[1])
	code := strings.TrimSpace(parts[2])

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println()
	err = formatter.Format(os.Stdout, style, iterator)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println()
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
