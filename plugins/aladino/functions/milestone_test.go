// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package plugins_aladino_functions_test

import (
	"net/http"
	"testing"

	"github.com/google/go-github/v49/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/reviewpad/reviewpad/v3/lang/aladino"
	plugins_aladino "github.com/reviewpad/reviewpad/v3/plugins/aladino"
	"github.com/reviewpad/reviewpad/v3/utils"
	"github.com/stretchr/testify/assert"
)

var milestone = plugins_aladino.PluginBuiltIns().Functions["milestone"].Code

func TestMilestone(t *testing.T) {
	milestoneTitle := "v1.0"
	mockedPullRequest := aladino.GetDefaultMockPullRequestDetailsWith(&github.PullRequest{
		Milestone: &github.Milestone{
			Title: github.String(milestoneTitle),
		},
	})
	mockedEnv := aladino.MockDefaultEnv(
		t,
		[]mock.MockBackendOption{
			mock.WithRequestMatchHandler(
				mock.GetReposPullsByOwnerByRepoByPullNumber,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					utils.MustWriteBytes(w, mock.MustMarshal(mockedPullRequest))
				}),
			),
		},
		nil,
		aladino.MockBuiltIns(),
		nil,
	)

	args := []aladino.Value{}
	gotMilestoneTitle, err := milestone(mockedEnv, args)

	wantMilestoneTitle := aladino.BuildStringValue(milestoneTitle)

	assert.Nil(t, err)
	assert.Equal(t, wantMilestoneTitle, gotMilestoneTitle)
}
