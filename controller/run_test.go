package controller

import "testing"

// TestRunShellConfigDisablesDotAndSlashCommands locks in the fix for
// `anyquery run` executing dot/slash commands from remotely-fetched query
// files (URL, S3, or Query Hub ID). Content run this way is not necessarily
// authored by the operator, and a grep across all shipped Query Hub .sql
// files found zero legitimate uses of any dot or slash command, so both
// must stay disabled here.
//
// Note this deliberately does not cover `.read`: that primitive is handled
// in shell.Run before the middlewares run at all and is not governed by
// this flag — see TestReadCommand in shell_test.go for its own gate.
func TestRunShellConfigDisablesDotAndSlashCommands(t *testing.T) {
	t.Parallel()

	cfg := runShellConfig()

	if cfg.GetBool("dot-command", true) {
		t.Error(`anyquery run must set "dot-command": false for remotely-fetched content`)
	}

	if cfg.GetBool("slash-command", true) {
		t.Error(`anyquery run must set "slash-command": false for remotely-fetched content`)
	}
}
