package judger

import (
	"fmt"
	"sync/atomic"

	"github.com/criyle/go-judge/client"
	"github.com/criyle/go-judge/file"
	"github.com/criyle/go-judge/types"
)

// Loop fetch judge task from client and report results
// in a infinite loop
func (j *Judger) Loop(done <-chan struct{}) {
	c := j.Client.C()
loop:
	for {
		select {
		case t := <-c:
			rt := j.run(done, t)
			t.Finish(rt)
			select {
			case <-done:
				break loop
			default:
			}
		case <-done:
			break loop
		}
	}
}

func (j *Judger) run(done <-chan struct{}, t client.Task) types.JudgeResult {
	var result types.JudgeResult
	errResult := func(err error) types.JudgeResult {
		result.Error = err.Error()
		return result
	}

	p := t.Param()
	pconf, err := j.Build(p.TestData)
	if err != nil {
		return errResult(err)
	}

	// compile
	compileRet := make(chan types.RunTaskResult)
	err = j.Enqueue(types.RunTask{
		Type:       "compile",
		Language:   p.Language,
		Code:       p.Code,
		ExtraFiles: pconf.ExtraFiles,
	}, compileRet)
	if err != nil {
		return errResult(err)
	}
	compileTaskResult := <-compileRet
	if compileTaskResult.Error != "" {
		return errResult(err)
	}
	execFiles := compileTaskResult.ExecFiles

	// run
	subTaskResult := make(chan types.JudgeSubTaskResult, len(pconf.Subtasks))
	pj := problemJudger{
		Judger:        j,
		ProblemConfig: &pconf,
		Task:          t,
		JudgeTask:     p,
		total:         count(&pconf),
	}
	for _, s := range pconf.Subtasks {
		s := &s
		go func() {
			subTaskResult <- pj.runSubtask(done, execFiles, s)
		}()
	}
	for range pconf.Subtasks {
		result.SubTasks = append(result.SubTasks, <-subTaskResult)
	}
	return result
}

type problemJudger struct {
	*Judger
	*types.ProblemConfig
	*types.JudgeTask
	client.Task
	count int32
	total int32
}

func (pj *problemJudger) runSubtask(done <-chan struct{}, exec []file.File, s *types.SubTask) types.JudgeSubTaskResult {
	var result types.JudgeSubTaskResult
	caseResult := make(chan types.RunTaskResult, len(s.Cases))
	for _, c := range s.Cases {
		pj.Enqueue(types.RunTask{
			Type:        pj.ProblemConfig.Type,
			Language:    pj.Language,
			TimeLimit:   pj.TileLimit,
			MemoryLimit: pj.MemoryLimit,
			Executables: exec,
			InputFile:   c.Input,
			AnswerFile:  c.Answer,
		}, caseResult)
	}
	for range s.Cases {
		rt := <-caseResult
		result.Cases = append(result.Cases, types.JudgeCaseResult{
			Status:     rt.Status,
			ScoreRate:  rt.ScoringRate,
			Error:      rt.Error,
			Time:       rt.Time,
			Memory:     rt.Memory,
			Input:      rt.Input,
			Answer:     rt.Answer,
			UserOutput: rt.UserOutput,
			UserError:  rt.UserError,
			SpjOutput:  rt.SpjOutput,
		})
		result.Score += rt.ScoringRate
		// report prograss
		atomic.AddInt32(&pj.count, 1)
		pj.Progress(types.JudgeProgress{
			Message: fmt.Sprintf("%d/%d", pj.count, pj.total),
		})
	}
	return result
}

// count counts total number of cases
func count(pconf *types.ProblemConfig) int32 {
	var count int32
	for _, s := range pconf.Subtasks {
		count += int32(len(s.Cases))
	}
	return count
}