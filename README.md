# chessdocs-api

Backend for the contribution flow on [chessdocs.org](https://chessdocs.org/). It hands a
reader the markdown source of a docs page, and turns the version they edit into a pull
request against the docs repository — so anyone can suggest a change without a GitHub
account, a fork, or a local checkout.

Written in Go against the standard library; `golang.org/x/time/rate` is the only
dependency the server binary carries.

## API

Both endpoints live at the root, and every failure answers with `{ "error": string }`.
[`openapi.yaml`](openapi.yaml) is the contract, and the validation in `internal/docs` is
tested against it so the two cannot drift.

### `GET /?path=<lang>/<page>.md`

Returns the page's current markdown as `text/plain`.

```console
$ curl 'https://api.chessdocs.org/?path=en/glossary/fork.md'
# Fork

A double attack by a single piece.
```

| Status | Meaning                                                       |
| ------ | ------------------------------------------------------------- |
| `200`  | The source, as it stands on the base branch                    |
| `400`  | `path` is missing or points outside the docs tree              |
| `403`  | The page's frontmatter sets `editLink: false` or `dev: true`   |
| `429`  | More than 60 requests a minute from one address                |
| `502`  | GitHub refused or could not be reached                         |

### `POST /`

Takes the full replacement text of one page and opens a pull request for it.

```console
$ curl -X POST https://api.chessdocs.org/ \
    -H 'Content-Type: application/json' \
    -d '{
          "title": "Clarify the fork example",
          "content": "# Fork\n\nA double attack by a single piece.",
          "sourcePath": "en/glossary/fork.md",
          "author": { "name": "Artem", "contact": "artem@example.com" }
        }'
{"url":"https://github.com/.../pull/42"}
```

`title` (≤ 120 characters) and `content` (≤ 4000) are required and may not be blank;
`author` is optional and `lang` defaults to `en`. The branch, the commit and the pull
request are created in a single GraphQL round trip, off the base commit the page was read
at, and the commit is signed by GitHub rather than attributed to an unverified author.

| Status | Meaning                                                     |
| ------ | ----------------------------------------------------------- |
| `201`  | Pull request opened; the body carries its `url`             |
| `400`  | The submission does not satisfy the contract                 |
| `403`  | The page does not accept edits                               |
| `413`  | The body is larger than 16 KiB                               |
| `429`  | More than 5 submissions per 10 minutes from one address      |
| `502`  | GitHub refused or could not be reached                       |

Editability is decided by the page's frontmatter on the base branch, which is re-read on
every submission — the rule mirrors the site's own, so a page the site will not offer an
edit link for cannot be edited through the API either.

## Configuration

All six variables are required; the server refuses to start without them and names every
one that is missing. See [`.env.example`](.env.example).

| Variable             | Purpose                                          |
| -------------------- | ------------------------------------------------ |
| `GITHUB_TOKEN`       | Token the pull requests are opened with           |
| `GITHUB_OWNER`       | Owner of the docs repository                      |
| `GITHUB_REPO`        | Docs repository name                              |
| `GITHUB_BASE_BRANCH` | Branch submissions are read from and opened onto  |
| `ALLOWED_ORIGIN`     | The single browser origin CORS admits             |
| `PORT`               | Port to listen on                                 |

## Layout

```
cmd/chessdocs-api   Startup, signal handling, graceful shutdown
internal/config     Environment loading
internal/api        HTTP transport: routing, CORS, rate limiting, error shaping
internal/docs       The domain: submissions, their validation, editability rules
internal/github     The docs repository: reading sources, opening pull requests
```

`internal/api` declares the GitHub behaviour it needs as an interface rather than
importing the client, so the handlers are tested end to end without a network, and
`internal/docs` depends on nothing at all.

## Development

```console
$ make run     # serve locally with .env.development
$ make check   # gofmt, go vet, go test -race — what CI runs
$ make cover   # test coverage per function
```

## Deployment

Pushes to `main` deploy over SSH: the VPS checkout is reset to the pushed commit and
`deploy.sh` installs the Caddy site and rebuilds the container. The image is a distroless
static binary running as a non-root user, behind Caddy at `api.chessdocs.org`.
