package gontentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	ASSET              = "Asset"
	DELETED_ASSET      = "DeletedAsset"
	ASSET_TABLE_NAME   = "_asset"
	ASSET_DISPLAYFIELD = "title"
	IMAGE_FOLDER_NAME  = "_images"
)

var (
	assetColumns = []string{"title", "description", "file_name", "content_type", "url"}
)

type AssetsService service

func (s *AssetsService) Create(body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathAssets, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	// Set header for content type
	s.client.headers[headerContentType] = "application/vnd.contentful.management.v1+json"
	return s.client.post(path, bytes.NewBuffer(body))
}

func (s *AssetsService) Process(id string, locale string) ([]byte, error) {
	path := fmt.Sprintf(pathAssetsProcess, s.client.Options.SpaceID, s.client.Options.EnvironmentID, id, locale)
	return s.client.put(path, nil)
}

func (s *AssetsService) Publish(id string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathAssetsPublished, s.client.Options.SpaceID, s.client.Options.EnvironmentID, id)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}

func (s *AssetsService) GetSingle(id string) ([]byte, error) {
	path := fmt.Sprintf(pathAssetsEntry, s.client.Options.SpaceID, s.client.Options.EnvironmentID, id)
	return s.client.getCMA(path, nil)
}

func (s *AssetsService) GetEntries(query url.Values) (*Entries, error) {
	path := fmt.Sprintf(pathAssets, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	data, err := s.client.getCMA(path, query)
	if err != nil {
		return nil, err
	}

	res := &Entries{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
