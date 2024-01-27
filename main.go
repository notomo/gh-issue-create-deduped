package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cli/go-gh"
)

func main() {
	args, err := ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err := Run(
		gh.Exec,
		args.RepoArgs,
		args.Title,
		args.IssueState,
		args.IssueCreateArgs,
		args.DryRun,
		os.Stdout,
		os.Stderr,
	); err != nil {
		log.Fatal(err)
	}
}

type Execute func(...string) (stdout, stderr bytes.Buffer, err error)

type Arguments struct {
	Title           string
	RepoArgs        []string
	DryRun          bool
	IssueState      string // all|open|closed
	IssueCreateArgs []string
}

func ParseArgs(rawArgs []string) (*Arguments, error) {
	args := &Arguments{
		IssueState: "open",
	}

	defaultSetter := func(arg string) {
		args.IssueCreateArgs = append(args.IssueCreateArgs, arg)
	}
	setter := defaultSetter

	for _, arg := range rawArgs {
		switch arg {
		case "-R", "--repo":
			setter = func(arg string) {
				args.RepoArgs = []string{"--repo", arg}
			}
		case "-t", "--title":
			setter = func(arg string) {
				args.Title = arg
				args.IssueCreateArgs = append(args.IssueCreateArgs, arg)
			}
			args.IssueCreateArgs = append(args.IssueCreateArgs, arg)
		case "-dry-run", "--dry-run":
			args.DryRun = true
		case "-state", "--state":
			setter = func(arg string) {
				args.IssueState = arg
			}
		default:
			setter(arg)
			setter = defaultSetter
		}
	}

	if args.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	return args, nil
}

type Issue struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func Run(
	execute Execute,
	repoArgs []string,
	title string,
	issueState string,
	issueCreateArgs []string,
	dryRun bool,
	stdout io.Writer,
	stderr io.Writer,
) error {
	{
		existed, err := findIssue(execute, repoArgs, title, issueState)
		if err != nil {
			return err
		}
		if existed != nil {
			stderr.Write([]byte("Already exists!\n"))
			stdout.Write([]byte(existed.URL + "\n"))
			return nil
		}
	}

	output, err := createIssue(
		execute,
		repoArgs,
		issueCreateArgs,
		dryRun,
	)
	if err != nil {
		return err
	}
	if output != nil {
		stdout.Write(output.Bytes())
		return nil
	}
	return nil
}

func findIssue(
	execute Execute,
	repoArgs []string,
	title string,
	issueState string,
) (*Issue, error) {
	args := []string{"issue"}
	args = append(args, repoArgs...)
	args = append(args, []string{
		"list",
		"--state", issueState,
		"--limit", "1000",
		"--json", "title,url",
		"--search", title,
	}...)
	stdout, _, err := execute(args...)
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
		return nil, fmt.Errorf("unmarshal issues: %w", err)
	}

	for _, issue := range issues {
		if issue.Title == title {
			return &issue, nil
		}
	}
	return nil, nil
}

func createIssue(
	execute Execute,
	repoArgs []string,
	issueCreateArgs []string,
	dryRun bool,
) (*bytes.Buffer, error) {
	args := []string{"issue"}
	args = append(args, repoArgs...)
	args = append(args, "create")
	args = append(args, issueCreateArgs...)

	if dryRun {
		return nil, nil
	}

	stdout, _, err := execute(args...)
	if err != nil {
		return nil, fmt.Errorf("gh issue create: %w", err)
	}
	return &stdout, nil
}
