// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package aladino

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/reviewpad/reviewpad/v3/codehost/github"
	"github.com/reviewpad/reviewpad/v3/codehost/github/target"
	"github.com/reviewpad/reviewpad/v3/collector"
	"github.com/reviewpad/reviewpad/v3/engine"
	"github.com/reviewpad/reviewpad/v3/handler"
	"github.com/reviewpad/reviewpad/v3/utils"
	"github.com/sirupsen/logrus"
)

type Interpreter struct {
	Env Env
}

func buildGroupAST(typeOf engine.GroupType, expr, paramExpr, whereExpr string) (Expr, error) {
	if typeOf == engine.GroupTypeFilter {
		whereExprAST, err := Parse(whereExpr)
		if err != nil {
			return nil, err
		}

		return BuildFilter(paramExpr, whereExprAST)
	} else {
		return Parse(expr)
	}
}

func evalGroup(env Env, expr Expr) (Value, error) {
	exprType, err := TypeInference(env, expr)
	if err != nil {
		return nil, err
	}

	if exprType.Kind() != ARRAY_TYPE && exprType.Kind() != ARRAY_OF_TYPE {
		return nil, fmt.Errorf("expression is not a valid group")
	}

	return Eval(env, expr)
}

func (i *Interpreter) ProcessGroup(groupName string, kind engine.GroupKind, typeOf engine.GroupType, expr, paramExpr, whereExpr string) error {
	exprAST, err := buildGroupAST(typeOf, expr, paramExpr, whereExpr)
	if err != nil {
		return fmt.Errorf("ProcessGroup:buildGroupAST: %v", err)
	}

	value, err := evalGroup(i.Env, exprAST)
	if err != nil {
		return fmt.Errorf("ProcessGroup:evalGroup %v", err)
	}

	i.Env.GetRegisterMap()[groupName] = value
	return nil
}

func BuildInternalLabelID(id string) string {
	return fmt.Sprintf("@label:%v", id)
}

func (i *Interpreter) ProcessLabel(id, name string) error {
	internalLabelID := BuildInternalLabelID(id)

	i.Env.GetRegisterMap()[internalLabelID] = BuildStringValue(name)
	return nil
}

func BuildInternalRuleName(name string) string {
	return fmt.Sprintf("@rule:%v", name)
}

func (i *Interpreter) ProcessRule(name, spec string) error {
	internalRuleName := BuildInternalRuleName(name)

	i.Env.GetRegisterMap()[internalRuleName] = BuildStringValue(spec)
	return nil
}

func EvalExpr(env Env, kind, expr string) (bool, error) {
	exprAST, err := Parse(expr)
	if err != nil {
		return false, err
	}

	exprType, err := TypeInference(env, exprAST)
	if err != nil {
		return false, err
	}

	if exprType.Kind() != BOOL_TYPE {
		return false, fmt.Errorf("expression %v is not a condition", expr)
	}

	return EvalCondition(env, exprAST)
}

func (i *Interpreter) EvalExpr(kind, expr string) (bool, error) {
	return EvalExpr(i.Env, kind, expr)
}

func (i *Interpreter) ExecProgram(program *engine.Program) (engine.ExitStatus, error) {
	i.Env.GetLogger().Info("executing program")

	for _, statement := range program.GetProgramStatements() {
		err := i.ExecStatement(statement)
		if err != nil {
			return engine.ExitStatusFailure, err
		}

		hasFatalError := len(i.Env.GetBuiltInsReportedMessages()[SEVERITY_FATAL]) > 0
		if hasFatalError {
			i.Env.GetLogger().Info("execution stopped")
			return engine.ExitStatusFailure, nil
		}
	}

	i.Env.GetLogger().Info("execution done")

	return engine.ExitStatusSuccess, nil
}

func (i *Interpreter) ExecStatement(statement *engine.Statement) error {
	statRaw := statement.GetStatementCode()
	statAST, err := Parse(statRaw)
	if err != nil {
		return err
	}

	execStatAST, err := TypeCheckExec(i.Env, statAST)
	if err != nil {
		return err
	}

	if !i.Env.GetDryRun() {
		err = execStatAST.exec(i.Env)
		if err != nil {
			return err
		}
	}

	i.Env.GetReport().addToReport(statement)

	i.Env.GetLogger().Infof("action %v executed", statRaw)
	return nil
}

func (i *Interpreter) Report(mode string, safeMode bool) error {
	i.Env.GetLogger().Info("generating report")

	if mode == "" {
		// By default mode is silent
		mode = engine.SILENT_MODE
	}

	env := i.Env

	var err error

	comment, err := FindReportCommentByAnnotation(env, ReviewpadReportCommentAnnotation)
	if err != nil {
		return err
	}

	reportComments := env.GetBuiltInsReportedMessages()

	if mode == engine.SILENT_MODE && len(reportComments) == 0 && !safeMode {
		if comment != nil {
			return DeleteReportComment(env, *comment.ID)
		}
		return nil
	}

	report := buildReport(mode, safeMode, reportComments, env.GetReport())

	if comment == nil {
		return AddReportComment(env, report)
	}

	return UpdateReportComment(env, *comment.ID, report)

}

func (i *Interpreter) ReportMetrics() error {
	targetEntity := i.Env.GetTarget().GetTargetEntity()
	owner := targetEntity.Owner
	prNum := targetEntity.Number
	repo := targetEntity.Repo
	ctx := i.Env.GetCtx()
	pr := i.Env.GetTarget().(*target.PullRequestTarget).PullRequest

	if !*pr.Merged {
		return nil
	}

	report := strings.Builder{}

	firstCommitDate, firstReviewDate, err := i.Env.GetGithubClient().GetFirstCommitAndReviewDate(ctx, owner, repo, prNum)
	if err != nil {
		return err
	}

	if firstCommitDate != nil {
		report.WriteString(fmt.Sprintf("**💻 Coding Time**: %s", utils.ReadableTimeDiff(*firstCommitDate, *pr.CreatedAt)))
	}

	if firstReviewDate != nil && firstReviewDate.Before(*pr.MergedAt) {
		report.WriteString(fmt.Sprintf("\n**🛻 Pickup Time**: %s", utils.ReadableTimeDiff(*pr.CreatedAt, *firstReviewDate)))

		report.WriteString(fmt.Sprintf("\n**👀 Review Time**: %s", utils.ReadableTimeDiff(*firstReviewDate, *pr.MergedAt)))
	}

	if report.Len() > 0 {
		comment, err := FindReportCommentByAnnotation(i.Env, ReviewpadMetricReportCommentAnnotation)
		if err != nil {
			return err
		}

		r := ReviewpadMetricReportCommentAnnotation + "\n## 📈 Pull Request Metrics\n" + report.String()

		if comment == nil {
			err = AddReportComment(i.Env, r)
			if err != nil {
				return err
			}
			return nil
		}

		err = UpdateReportComment(i.Env, *comment.ID, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewInterpreter(
	ctx context.Context,
	logger *logrus.Entry,
	dryRun bool,
	githubClient *gh.GithubClient,
	collector collector.Collector,
	targetEntity *handler.TargetEntity,
	eventPayload interface{},
	builtIns *BuiltIns,
) (engine.Interpreter, error) {
	evalEnv, err := NewEvalEnv(ctx, logger, dryRun, githubClient, collector, targetEntity, eventPayload, builtIns)
	if err != nil {
		return nil, err
	}

	return &Interpreter{
		Env: evalEnv,
	}, nil
}
