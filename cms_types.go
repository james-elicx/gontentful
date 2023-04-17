package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

func GetCMSSchemas(repo string, ct string) (*ContentTypes, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchemasRecursive(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas from github: %s", err.Error())
	}

	schemas := &ContentTypes{
		Total: len(res),
		Limit: 0,
		Skip:  0,
		Items: make([]*ContentType, 0),
	}

	for _, rc := range res {
		ect := extractContentype(*rc.Path)

		ghc, err := rc.GetContent()
		if err != nil {
			return nil, fmt.Errorf("repositoryContent.GetContent failed: %s", err.Error())
		}
		m := &content.Schema{}
		_ = json.Unmarshal([]byte(ghc), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error())
		}
		schemas.Items = append(schemas.Items, formatSchema(m))
	}
	return schemas, nil
}

func GetCMSSchemasExpanded(repo string, ct string) (*ContentTypes, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchemasRecursive(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas from github: %s", err.Error())
	}

	schemas := &ContentTypes{
		Total: len(res),
		Limit: 0,
		Skip:  0,
		Items: make([]*ContentType, 0),
	}

	for _, rc := range res {
		ect := extractContentype(*rc.Path)

		ghc, err := rc.GetContent()
		if err != nil {
			return nil, fmt.Errorf("repositoryContent.GetContent failed: %s", err.Error())
		}
		m := &content.Schema{}
		_ = json.Unmarshal([]byte(ghc), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error())
		}
		schemas.Items = append(schemas.Items, formatSchemaRecursive(m)...)
	}
	return schemas, nil
}
