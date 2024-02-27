# Gontentful

Contentful client library for Go with a command line interface for schema export and data sync.

## Library

### Install

```sh
$ go get -u github.com/james-elicx/gontentful
```

### Usage

Create client:

```sh
import (
	"github.com/james-elicx/gontentful"
)

client := gontentful.NewClient(&gontentful.ClientOptions{
	CdnURL:        "cdn.contentful.com",
	SpaceID:       <spaceid>,
	EnvironmentID: <environmentid>,
	CdnToken:      <cdntoken>,
})

// get entries
query := url.Values{}
query.Set("content_type", "foo")
query.Set("locale", "en")
entries, err := client.Entries.GetEntries(query)

// get entry
entry, err := client.Entries.GetSingle(<entryid>)
```

## CLI

### Install

```sh
$ go get -u github.com/james-elicx/gontentful/cmd/gfl
```

### Usage

Schema export:

```sh
# generate postgres schema and print to stdout
$ gfl schema pg --space <spaceid> --token <token>

# generate postgres schema and execute on the specified database
$ gfl schema pg --space <spaceid> --token <token> --url postgres://user:pass@host:port/db

# generate graphql schema and print to stdout
$ gfl schema gql --space <spaceid> --token <token>
```

Data sync:

```sh
# sync data to postgres (init sync first then incremental)
$ gfl sync pg --space <spaceid> --token <token> --url postgres://user:pass@host:port/db

# sync data to postgres (init sync always start from scratch)
$ gfl sync pg --init --space <spaceid> --token <token> --url postgres://user:pass@host:port/db
```

## Dependencies

Using Go modules:

```sh
$ go mod vendor
```

## License

Licensed under the [MIT License](LICENSE)

### Acknowledgements

Utilize code from [contentful-go](https://github.com/contentful-labs/contentful-go)
