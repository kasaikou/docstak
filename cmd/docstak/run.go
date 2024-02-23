/*
Copyright 2024 Kasai Kou

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kasaikou/docstak/cli"
	"github.com/kasaikou/docstak/docstak"
	"github.com/kasaikou/docstak/docstak/condition"
	"github.com/kasaikou/docstak/docstak/markdown"
	"github.com/kasaikou/docstak/docstak/model"
	"github.com/kasaikou/docstak/docstak/resolver"
	"github.com/kasaikou/docstak/docstak/srun"
)

func run() int {
	cwWaiter := sync.WaitGroup{}
	defer cwWaiter.Wait()
	cw, _ := cli.NewConsoleWriter(os.Stdout)
	cwWaiter.Add(1)
	go func() {
		defer cwWaiter.Done()
		cw.Route()
	}()
	defer cw.Close()

	logger := slog.New(cw.NewLoggerHandler(nil))
	ctx := docstak.WithLogger(context.Background(), logger)
	wd, _ := os.Getwd()
	po, err := markdown.FromLocalFile(wd, "docstak.md")
	if err != nil {
		logger.Error("cannot open file", slog.Any("error", err))
	}

	parsed, err := markdown.ParseMarkdown(ctx, po)
	if err != nil {
		logger.Error("cannot parse markdown", slog.String("filepath", po.Filename()), slog.Any("error", err))
		return -1
	}

	document, err := model.NewDocument(ctx,
		model.NewDocOptionRootDir(filepath.Dir(po.Filename())),
		resolver.NewDocumentWithPathResolver(
			resolver.ResolveOption{Lang: []string{"sh", "shell"}, Command: "sh", CmdOpt: "-c"},
			resolver.ResolveOption{Lang: []string{"bash"}, Command: "bash", CmdOpt: "-c"},
			resolver.ResolveOption{Lang: []string{"powershell", "posh"}, Command: "powershell", CmdOpt: "-Command"},
			resolver.ResolveOption{Lang: []string{"py", "python"}, Command: "python", CmdOpt: "-c"},
			resolver.ResolveOption{Lang: []string{"js", "javascript"}, Command: "node", CmdOpt: "-e"},
		),
		markdown.NewDocFromMarkdownParsing(parsed),
	)

	if err != nil {
		logger.Error("failed to initialize document", slog.String("error", err.Error()))
		return -1
	}

	chDecoration := make(chan cli.ProcessOutputDecoration, len(cli.ProcessOutputDecorations))
	for i := range cli.ProcessOutputDecorations {
		chDecoration <- cli.ProcessOutputDecorations[i]
	}

	return docstak.ExecuteContext(ctx, document,
		docstak.ExecuteOptCalls(Cmds...),
		docstak.ExecuteOptProcessExec(func(ctx context.Context, task model.DocumentTask, runner *srun.ScriptRunner) (int, error) {
			decoration := <-chDecoration
			defer func() {
				chDecoration <- decoration
			}()

			sufficient := condition.NewRequiresFromDocumentTask(&task).Test(ctx, condition.TestOption{})
			if !sufficient {
				logger.Error("task's require rules are insufficient", slog.String("task", task.Call))
				return -1, nil
			}

			stdOutScanner := cw.NewScanner(decoration.Stdout, "STDOUT", task.Title)
			stdout, _ := runner.Stdout()
			stderrScanner := cw.NewScanner(decoration.Stderr, "ERROUT", task.Title)
			stderr, _ := runner.Stderr()

			wg := sync.WaitGroup{}
			defer wg.Wait()

			wg.Add(1)
			go func() {
				defer wg.Done()
				stdOutScanner.Scan(stdout)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				stderrScanner.Scan(stderr)
			}()

			return runner.RunContext(ctx)
		}),
	)
}
