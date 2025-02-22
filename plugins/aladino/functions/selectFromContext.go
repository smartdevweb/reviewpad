// Copyright (C) 2022 Explore.dev, Unipessoal Lda - All Rights Reserved
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package plugins_aladino_functions

import (
	"github.com/reviewpad/reviewpad/v3/handler"
	"github.com/reviewpad/reviewpad/v3/lang/aladino"
)

func SelectFromContext() *aladino.BuiltInFunction {
	return &aladino.BuiltInFunction{
		Type:           aladino.BuildFunctionType([]aladino.Type{aladino.BuildStringType()}, aladino.BuildStringType()),
		Code:           selectFromContext,
		SupportedKinds: []handler.TargetEntityKind{handler.PullRequest, handler.Issue},
	}
}

func selectFromContext(e aladino.Env, args []aladino.Value) (aladino.Value, error) {
	path := args[0].(*aladino.StringValue)

	targetContext, err := contextCode(e, nil)
	if err != nil {
		return nil, err
	}

	contextJSON, err := toJSONCode(e, []aladino.Value{targetContext})
	if err != nil {
		return nil, err
	}

	return selectFromJSONCode(e, []aladino.Value{contextJSON, path})
}
