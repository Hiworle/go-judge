package types

import (
	"time"

	"github.com/criyle/go-judge/file"
	"github.com/criyle/go-sandbox/types"
)

// RunTask is used to send task into RunQueue,
// if taskqueue is a remote queue, taskqueue need to store / retrive files
type RunTask struct {
	Type string // compile / standard / interactive / answer_submit

	// Used for compile task
	Compile *CompileTask

	// Used for exec tasks
	Exec *ExecTask
}

// CompileTask defines compile task
type CompileTask SourceCode

// ExecTask defines run tasks
type ExecTask struct {
	// Executable
	Exec *CompiledExec

	// Run limits
	TimeLimit   time.Duration
	MemoryLimit types.Size

	// Input / Output
	InputFile  file.File
	AnswerFile file.File

	// File I/O file names if not empty
	InputFileName  *string
	OutputFileName *string

	// Special Judge
	SPJ *CompiledExec

	// Interactor for interactive type
	Interactor *CompiledExec

	// UserAnswers for answer submission run task
	UserAnswer []file.File
}
