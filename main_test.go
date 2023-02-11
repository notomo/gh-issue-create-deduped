package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	testtarget "github.com/notomo/gh-issue-create-deduped"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestParseArgs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
			want testtarget.Arguments
		}{
			{
				name: "title only",
				args: []string{
					"--title", "title",
				},
				want: testtarget.Arguments{
					Title: "title",
					IssueCreateArgs: []string{
						"--title", "title",
					},
				},
			},
			{
				name: "dry run",
				args: []string{
					"--title", "title",
					"--dry-run",
				},
				want: testtarget.Arguments{
					Title: "title",
					IssueCreateArgs: []string{
						"--title", "title",
					},
					DryRun: true,
				},
			},
			{
				name: "repo long",
				args: []string{
					"--repo", "notomo/test",
					"--title", "title",
				},
				want: testtarget.Arguments{
					RepoArgs: []string{"--repo", "notomo/test"},
					Title:    "title",
					IssueCreateArgs: []string{
						"--title", "title",
					},
				},
			},
			{
				name: "repo short",
				args: []string{
					"-R", "notomo/test",
					"--title", "title",
				},
				want: testtarget.Arguments{
					RepoArgs: []string{"--repo", "notomo/test"},
					Title:    "title",
					IssueCreateArgs: []string{
						"--title", "title",
					},
				},
			},
			{
				name: "other option",
				args: []string{
					"--title", "title",
					"--label", "bug",
				},
				want: testtarget.Arguments{
					Title: "title",
					IssueCreateArgs: []string{
						"--title", "title",
						"--label", "bug",
					},
				},
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				got, err := testtarget.ParseArgs(c.args)
				require.NoError(t, err)
				assert.Equal(t, c.want, *got)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
			want error
		}{
			{
				name: "no args",
				args: []string{},
				want: fmt.Errorf("title is required"),
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				_, err := testtarget.ParseArgs(c.args)
				assert.Equal(t, c.want, err)
			})
		}
	})
}

func listOutput(issues ...testtarget.Issue) testtarget.Execute {
	return func(args ...string) (bytes.Buffer, bytes.Buffer, error) {
		b, err := json.Marshal(issues)
		if err != nil {
			return bytes.Buffer{}, bytes.Buffer{}, err
		}
		stdout := bytes.NewBuffer(b)
		return *stdout, bytes.Buffer{}, nil
	}
}

type Mock struct {
	List   testtarget.Execute
	Create testtarget.Execute
}

func (m *Mock) Exec(args ...string) (bytes.Buffer, bytes.Buffer, error) {
	if slices.Contains(args, "list") {
		return m.List(args...)
	}
	if slices.Contains(args, "create") {
		return m.Create(args...)
	}
	return bytes.Buffer{}, bytes.Buffer{}, fmt.Errorf("invalid args: %s", args)
}

func TestRun(t *testing.T) {
	t.Run("does not create issue if the title is duplicated", func(t *testing.T) {
		mock := &Mock{
			List: listOutput(testtarget.Issue{
				Title: "title",
				URL:   "https://github.com/notomo/example/issues/1",
			}),
			Create: func(args ...string) (bytes.Buffer, bytes.Buffer, error) {
				t.Fail()
				return bytes.Buffer{}, bytes.Buffer{}, nil
			},
		}

		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		require.NoError(t, testtarget.Run(
			mock.Exec,
			[]string{"--repo", "notomo/example"},
			"title",
			[]string{"--title", "title"},
			false,
			&stdout,
			&stderr,
		))

		assert.Equal(t, "Already exists!\n", stderr.String())
		assert.Equal(t, "https://github.com/notomo/example/issues/1\n", stdout.String())
	})

	t.Run("create issue if the title is not duplicated", func(t *testing.T) {
		mock := &Mock{
			List: listOutput(),
			Create: func(args ...string) (bytes.Buffer, bytes.Buffer, error) {
				stdout := bytes.NewBufferString("https://github.com/notomo/example/issues/2\n")
				return *stdout, bytes.Buffer{}, nil
			},
		}

		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		require.NoError(t, testtarget.Run(
			mock.Exec,
			[]string{"--repo", "notomo/example"},
			"title",
			[]string{"--title", "title"},
			false,
			&stdout,
			&stderr,
		))

		assert.Equal(t, "", stderr.String())
		assert.Equal(t, "https://github.com/notomo/example/issues/2\n", stdout.String())
	})
}
