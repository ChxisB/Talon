// Package util provides utility functions for UI message handling.
package util

import (
	"context"
	"errors"
	"os/exec"
	"time"

	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
	"mvdan.cc/sh/v3/shell"
)

type Cursor interface {
	Cursor() *bubble.Cursor
}

func CmdHandler(msg bubble.Msg) bubble.Cmd {
	return func() bubble.Msg {
		return msg
	}
}

func ReportError(err error) bubble.Cmd {
	return CmdHandler(NewErrorMsg(err))
}

type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarn
	InfoTypeError
	InfoTypeUpdate
)

func NewInfoMsg(info string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	}
}

func NewWarnMsg(warn string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	}
}

func NewErrorMsg(err error) InfoMsg {
	return InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	}
}

func ReportInfo(info string) bubble.Cmd {
	return CmdHandler(NewInfoMsg(info))
}

func ReportWarn(warn string) bubble.Cmd {
	return CmdHandler(NewWarnMsg(warn))
}

type (
	InfoMsg struct {
		Type InfoType
		Msg  string
		TTL  time.Duration
	}
	ClearStatusMsg struct{}
)

// IsEmpty checks if the [InfoMsg] is empty.
func (m InfoMsg) IsEmpty() bool {
	var zero InfoMsg
	return m == zero
}

// ExecShell parses a shell command string and executes it with exec.Command.
// Uses shell.Fields for proper handling of shell syntax like quotes and
// arguments while preserving TTY handling for terminal editors.
func ExecShell(ctx context.Context, cmdStr string, callback bubble.ExecCallback) bubble.Cmd {
	fields, err := shell.Fields(cmdStr, nil)
	if err != nil {
		return ReportError(err)
	}
	if len(fields) == 0 {
		return ReportError(errors.New("empty command"))
	}

	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	return bubble.ExecProcess(cmd, callback)
}
