package gontentful

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	timeout = 30 * time.Second

	pathSpaces              = "/spaces/%s"
	pathEnvironments        = "/environments/%s"
	pathSpacesCreate        = "/spaces"
	pathEntries             = pathSpaces + pathEnvironments + "/entries"
	pathEntry               = pathEntries + "/%s"
	pathEntriesPublish      = pathEntry + "/published"
	pathEntriesArchive      = pathEntry + "/archived"
	pathSync                = pathSpaces + pathEnvironments + "/sync"
	pathAssets              = pathSpaces + pathEnvironments + "/assets"
	pathAssetsProcess       = pathAssets + "/%s/files/%s/process"
	pathAssetsPublished     = pathAssets + "/%s/published"
	pathAssetsEntry         = pathAssets + "/%s"
	pathUploads             = pathSpaces + "/uploads"
	pathContentTypes        = pathSpaces + pathEnvironments + "/content_types"
	pathContentType         = pathContentTypes + "/%s"
	pathContentTypesPublish = pathContentType + "/published"
	pathLocales             = pathSpaces + "/locales"

	headerContentfulContentType  = "X-Contentful-Content-Type"
	headerContentfulVersion      = "X-Contentful-Version"
	headerContentfulOrganization = "X-Contentful-Organization"
	headerContentType            = "Content-Type"
	headerAuthorization          = "Authorization"
)

type Client struct {
	client       *http.Client
	headers      map[string]string
	Options      *ClientOptions
	AfterRequest func(c *Client, req *http.Request, res *http.Response, elapsed time.Duration)

	common       service
	Entries      *EntriesService
	Spaces       *SpacesService
	Locales      *LocalesService
	Assets       *AssetsService
	Uploads      *UploadsService
	ContentTypes *ContentTypesService
}

type service struct {
	client *Client
}

type ClientOptions struct {
	OrgID         string
	SpaceID       string
	EnvironmentID string
	CdnToken      string
	PreviewToken  string
	CmaToken      string
	CdnURL        string
	PreviewURL    string
	CmaURL        string
	UsePreview    bool
}

func NewClient(options *ClientOptions) *Client {
	httpClient := &http.Client{
		Timeout: timeout,
	}

	client := &Client{
		Options: options,
		client:  httpClient,
		headers: getHeadersMap(options.OrgID),
	}

	client.common.client = client
	client.Entries = (*EntriesService)(&client.common)
	client.Spaces = (*SpacesService)(&client.common)
	client.Locales = (*LocalesService)(&client.common)
	client.Assets = (*AssetsService)(&client.common)
	client.Uploads = (*UploadsService)(&client.common)
	client.ContentTypes = (*ContentTypesService)(&client.common)

	return client
}

func getHeadersMap(orgID string) map[string]string {
	return map[string]string{
		headerContentfulOrganization: orgID,
		headerContentType:            "application/vnd.contentful.delivery.v1+json",
	}
}

func (c *Client) get(path string, query url.Values) ([]byte, error) {
	host := ""
	authToken := ""
	if c.Options.UsePreview {
		host = c.Options.PreviewURL
		authToken = c.Options.PreviewToken
	} else {
		host = c.Options.CdnURL
		authToken = c.Options.CdnToken
	}
	return c.req(http.MethodGet, path, query, nil, host, authToken)
}

func (c *Client) getCMA(path string, query url.Values) ([]byte, error) {
	return c.req(http.MethodGet, path, query, nil, c.Options.CmaURL, c.Options.CmaToken)
}

func (c *Client) post(path string, body io.Reader) ([]byte, error) {
	return c.req(http.MethodPost, path, nil, body, c.Options.CmaURL, c.Options.CmaToken)
}

func (c *Client) put(path string, body io.Reader) ([]byte, error) {
	return c.req(http.MethodPut, path, nil, body, c.Options.CmaURL, c.Options.CmaToken)
}

func (c *Client) delete(path string) ([]byte, error) {
	return c.req(http.MethodDelete, path, nil, nil, c.Options.CmaURL, c.Options.CmaToken)
}

func (c *Client) req(method string, path string, query url.Values, body io.Reader, host string, authToken string) ([]byte, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
	u.RawQuery = query.Encode()

	// fmt.Println(fmt.Sprintf("%s%s?%s", host, path, u.RawQuery))

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	// set headers
	for key, value := range c.headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	// fmt.Println(fmt.Sprintf("%s: Bearer %s", headerAuthorization, authToken))
	// add auth header
	req.Header.Set(headerAuthorization, fmt.Sprintf("Bearer %s", authToken))
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	start := time.Now()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if c.AfterRequest != nil {
		c.AfterRequest(c, req, res, time.Since(start))
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		// clear headers so that they will not infect the next request.
		c.headers = getHeadersMap(c.Options.OrgID)
		// return the response
		return io.ReadAll(res.Body)
	}

	apiError := parseError(req, res)

	// return apiError if it is not rate limit error
	if _, ok := apiError.(RateLimitExceededError); !ok {
		return nil, apiError
	}

	resetHeader := res.Header.Get("x-contentful-ratelimit-reset")

	// return apiError if Ratelimit-Reset header is not presented
	if resetHeader == "" {
		return nil, apiError
	}

	// wait X-Contentful-Ratelimit-Reset amount of seconds
	waitSeconds, err := strconv.Atoi(resetHeader)
	if err != nil {
		return nil, apiError
	}

	// retry on rate limit
	time.Sleep(time.Second * time.Duration(waitSeconds))
	return c.do(req)
}
