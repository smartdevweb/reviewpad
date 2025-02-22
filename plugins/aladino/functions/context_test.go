// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package plugins_aladino_functions_test

import (
	"testing"

	"github.com/reviewpad/reviewpad/v3/handler"
	"github.com/reviewpad/reviewpad/v3/lang/aladino"
	plugins_aladino "github.com/reviewpad/reviewpad/v3/plugins/aladino"
	"github.com/reviewpad/reviewpad/v3/utils"
	"github.com/stretchr/testify/assert"
)

var context = plugins_aladino.PluginBuiltIns().Functions["context"].Code

func TestContext(t *testing.T) {
	mockPRJSON, err := utils.CompactJSONString(`{
        "id": 1234,
        "number": 6,
        "state": "open",
        "title": "Amazing new feature",
        "body": "Please pull these awesome changes in!",
        "created_at": "2009-11-17T20:34:58.651387237Z",
        "labels": [
            {
                "id": 1,
                "name": "enhancement"
            },
            {
                "id": 2,
                "name": "large"
            }
        ],
        "user": {
            "login": "john"
        },
        "merged": true,
        "comments": 6,
        "commits": 5,
        "url": "https://foo.bar",
        "assignees": [
            {
                "login": "jane"
            }
        ],
        "milestone": {
            "title":"v1.0"
        },
        "node_id": "test",
        "requested_reviewers": [
            {
                "login": "jane"
            }
        ],
        "head": {
            "ref": "new-topic",
            "repo": {
                "owner": {
                    "login": "foobar"
                },
                "name": "default-mock-repo",
                "url": "https://api.github.com/repos/foobar/default-mock-repo/pulls/6"
            }
        },
        "base":{
            "ref": "master",
            "repo": {
                "owner": {
                    "login": "foobar"
                },
                "name": "default-mock-repo",
                "url" :"https://api.github.com/repos/foobar/default-mock-repo/pulls/6"
            }
        }
    }`)
	assert.Nil(t, err)

	mockIssueJSON, err := utils.CompactJSONString(`{
        "id": 1234,
        "number": 6,
        "title": "Found a bug",
        "body": "I'm having a problem with this",
        "user": {
            "login": "john"
        },
        "labels": [
            {
                "id": 1,
                "name": "bug"
            }
        ],
        "comments": 6,
        "created_at": "2009-11-17T20:34:58.651387237Z",
        "url": "https://foo.bar",
        "milestone": {
            "title": "v1.0"
        },
        "assignees": [
            {
                "login": "jane"
            }
        ]
    }`)
	assert.Nil(t, err)

	tests := map[string]struct {
		targetEntity *handler.TargetEntity
		wantErr      error
		wantRes      aladino.Value
	}{
		"when pull request": {
			targetEntity: aladino.DefaultMockTargetEntity,
			wantRes:      aladino.BuildStringValue(mockPRJSON),
		},
		"when issue": {
			targetEntity: &handler.TargetEntity{
				Kind:   handler.Issue,
				Owner:  "test",
				Repo:   "test",
				Number: 1,
			},
			wantRes: aladino.BuildStringValue(mockIssueJSON),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			env := aladino.MockDefaultEnvWithTargetEntity(
				t,
				nil,
				nil,
				aladino.MockBuiltIns(),
				nil,
				test.targetEntity,
			)

			res, err := context(env, []aladino.Value{})

			assert.Equal(t, test.wantErr, err)
			assert.Equal(t, test.wantRes, res)
		})
	}
}
