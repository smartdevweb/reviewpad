// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package aladino_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-github/v49/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	gh "github.com/reviewpad/reviewpad/v3/codehost/github"
	"github.com/reviewpad/reviewpad/v3/codehost/github/target"
	"github.com/reviewpad/reviewpad/v3/handler"
	"github.com/reviewpad/reviewpad/v3/lang/aladino"
	"github.com/stretchr/testify/assert"
)

func TestNewTypeEnv_WithDefaultEnv(t *testing.T) {
	mockedEnv := aladino.MockDefaultEnv(t, nil, nil, aladino.MockBuiltIns(), nil)

	wantTypeEnv := aladino.TypeEnv(map[string]aladino.Type{
		"emptyFunction": aladino.BuildFunctionType([]aladino.Type{}, nil),
		"emptyAction":   aladino.BuildFunctionType([]aladino.Type{}, nil),
		"zeroConst":     aladino.BuildFunctionType([]aladino.Type{}, aladino.BuildIntType()),
		"returnStr":     aladino.BuildFunctionType([]aladino.Type{aladino.BuildStringType()}, aladino.BuildStringType()),
	})

	gotTypeEnv := aladino.NewTypeEnv(mockedEnv)

	assert.Equal(t, wantTypeEnv, gotTypeEnv)
}

func TestNewEvalEnv_WhenGetPullRequestFilesFails(t *testing.T) {
	failMessage := "GetPullRequestFilesFail"
	mockedGithubClientREST := github.NewClient(mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			aladino.GetDefaultMockPullRequestDetails(),
		),
		mock.WithRequestMatchHandler(
			mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mock.WriteError(
					w,
					http.StatusInternalServerError,
					failMessage,
				)
			}),
		),
	))

	mockedGithubClient := gh.NewGithubClient(mockedGithubClientREST, nil)
	ctx := context.Background()

	targetEntity := &handler.TargetEntity{
		Owner:  aladino.DefaultMockPrOwner,
		Repo:   aladino.DefaultMockPrRepoName,
		Number: aladino.DefaultMockPrNum,
		Kind:   handler.PullRequest,
	}

	env, err := aladino.NewEvalEnv(
		ctx,
		aladino.DefaultMockLogger,
		false,
		mockedGithubClient,
		aladino.DefaultMockCollector,
		targetEntity,
		nil,
		aladino.MockBuiltIns(),
	)

	assert.Nil(t, env)
	assert.Equal(t, err.(*github.ErrorResponse).Message, failMessage)
}

func TestNewEvalEnv_WhenNewFileFails(t *testing.T) {
	mockedGithubClientREST := github.NewClient(mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			aladino.GetDefaultMockPullRequestDetails(),
		),
		mock.WithRequestMatch(
			mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
			&[]*github.CommitFile{
				{Patch: github.String("@@a")},
			},
		),
	))

	mockedGithubClient := gh.NewGithubClient(mockedGithubClientREST, nil)
	ctx := context.Background()
	targetEntity := &handler.TargetEntity{
		Owner:  aladino.DefaultMockPrOwner,
		Repo:   aladino.DefaultMockPrRepoName,
		Number: aladino.DefaultMockPrNum,
		Kind:   handler.PullRequest,
	}

	env, err := aladino.NewEvalEnv(
		ctx,
		aladino.DefaultMockLogger,
		false,
		mockedGithubClient,
		aladino.DefaultMockCollector,
		targetEntity,
		nil,
		aladino.MockBuiltIns(),
	)

	assert.Nil(t, env)
	assert.EqualError(t, err, "error in file patch : error in chunk lines parsing (1): missing lines info: @@a\npatch: @@a")
}

func TestNewEvalEnv(t *testing.T) {
	fileName := "default-mock-repo/file1.ts"
	patch := "@@ -2,9 +2,11 @@ package main\n- func previous1() {\n+ func new1() {\n+\nreturn"
	mockedGithubClientREST := github.NewClient(mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			aladino.GetDefaultMockPullRequestDetails(),
			aladino.GetDefaultMockPullRequestDetails(),
		),
		mock.WithRequestMatch(
			mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
			&[]*github.CommitFile{
				{
					Filename: github.String(fileName),
					Patch:    github.String(patch),
				},
			},
		),
	))

	ctx := context.Background()
	mockedPullRequest, _, err := mockedGithubClientREST.PullRequests.Get(
		ctx,
		aladino.DefaultMockPrOwner,
		aladino.DefaultMockPrRepoName,
		aladino.DefaultMockPrNum,
	)

	if err != nil {
		assert.FailNow(t, "couldn't get pull request", err)
	}

	mockedGithubClient := gh.NewGithubClient(mockedGithubClientREST, nil)

	gotEnv, err := aladino.NewEvalEnv(
		ctx,
		aladino.DefaultMockLogger,
		false,
		mockedGithubClient,
		aladino.DefaultMockCollector,
		aladino.DefaultMockTargetEntity,
		nil,
		aladino.MockBuiltIns(),
	)

	mockedTarget, _ := target.NewPullRequestTarget(ctx, aladino.DefaultMockTargetEntity, mockedGithubClient, mockedPullRequest)

	wantEnv := &aladino.BaseEnv{
		Ctx:          ctx,
		GithubClient: mockedGithubClient,
		Collector:    aladino.DefaultMockCollector,
		RegisterMap:  aladino.RegisterMap(make(map[string]aladino.Value)),
		BuiltIns:     aladino.MockBuiltIns(),
		Report:       &aladino.Report{Actions: make([]string, 0)},
		// TODO: Mock an event
		EventPayload: nil,
		Target:       mockedTarget,
	}

	assert.Nil(t, err)
	assert.Equal(t, wantEnv.Ctx, gotEnv.GetCtx())
	assert.Equal(t, wantEnv.GithubClient, gotEnv.GetGithubClient())
	assert.Equal(t, wantEnv.Collector, gotEnv.GetCollector())
	assert.Equal(t, wantEnv.RegisterMap, gotEnv.GetRegisterMap())
	assert.Equal(t, wantEnv.EventPayload, gotEnv.GetEventPayload())
	// TODO: Compare the targets
	// assert.Equal(t, wantEnv.Target, gotEnv.GetTarget())

	assert.Equal(t, len(wantEnv.BuiltIns.Functions), len(gotEnv.GetBuiltIns().Functions))
	assert.Equal(t, len(wantEnv.BuiltIns.Actions), len(gotEnv.GetBuiltIns().Actions))
	for functionName, functionCode := range wantEnv.BuiltIns.Functions {
		assert.NotNil(t, gotEnv.GetBuiltIns().Functions[functionName])
		assert.Equal(t, functionCode.Type, gotEnv.GetBuiltIns().Functions[functionName].Type)
	}
	for actionName, actionCode := range wantEnv.BuiltIns.Actions {
		assert.NotNil(t, gotEnv.GetBuiltIns().Actions[actionName])
		assert.Equal(t, actionCode.Type, gotEnv.GetBuiltIns().Actions[actionName].Type)
	}

	assert.Equal(t, wantEnv.Report, gotEnv.GetReport())
}

func TestNewEvalEnv_WhenGetPullRequestFails(t *testing.T) {
	failMessage := "GetPullRequestFail"
	mockedGithubClientREST := github.NewClient(mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mock.WriteError(
					w,
					http.StatusInternalServerError,
					failMessage,
				)
			}),
		),
	))

	mockedGithubClient := gh.NewGithubClient(mockedGithubClientREST, nil)
	ctx := context.Background()

	targetEntity := &handler.TargetEntity{
		Owner:  aladino.DefaultMockPrOwner,
		Repo:   aladino.DefaultMockPrRepoName,
		Number: aladino.DefaultMockPrNum,
		Kind:   handler.PullRequest,
	}

	env, err := aladino.NewEvalEnv(
		ctx,
		aladino.DefaultMockLogger,
		false,
		mockedGithubClient,
		aladino.DefaultMockCollector,
		targetEntity,
		nil,
		aladino.MockBuiltIns(),
	)

	assert.Nil(t, env)
	assert.Equal(t, err.(*github.ErrorResponse).Message, failMessage)
}
