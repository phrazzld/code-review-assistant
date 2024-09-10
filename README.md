# Code Review Assistant

A command-line tool that leverages LLMs to provide insightful code review assistance for GitHub pull requests.

## Features

- Fetches PR metadata and diff from GitHub
- Generates code review insights using OpenAI's GPT-4
- Colorized output with syntax highlighting for code snippets
- Progress indicator during API calls

## Installation

1. Ensure you have Go 1.16+ installed
2. Clone this repository
3. Run `go build -o code-review-assistant`

## Usage

```
./code-review-assistant [owner/repo] [pr_number] --github-token YOUR_GITHUB_TOKEN --openai-key YOUR_OPENAI_KEY
```

Example:
```
./code-review-assistant octocat/Hello-World 1 --github-token ghp_xxxxxxxxxxxx --openai-key sk-xxxxxxxxxxxx
```

## Configuration

- `GITHUB_TOKEN`: Your GitHub Personal Access Token
- `OPENAI_API_KEY`: Your OpenAI API Key

You can set these as environment variables or pass them as command-line flags.

## Dependencies

- github.com/google/go-github/v39
- github.com/sashabaranov/go-openai
- github.com/spf13/cobra
- github.com/fatih/color
- github.com/alecthomas/chroma

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
