# terraform-provider-rt

A Terraform provider for [Request Tracker], generated from the reverse
engineered OpenAPI description of its REST2 API.

```bash
nix develop
bin/generate        # one RT document in, this repo out
```

Everything under `internal/` is generated and **committed**: the Dockerfile
compiles what is in the tree, not what a regeneration would produce. Run
`bin/generate`, read the diff, commit it.

## Where it comes from

[n-at-han-k/openapi-schema-rt], a fork of [rt/request-tracker-openapi] — the
same document [crossplane-provider-rt] is generated from, pinned as a
**submodule** at `reference/openapi-schema-rt` so that `git submodule status`
says which spec this tree came from without anyone writing it down.

```bash
git submodule update --init                                 # the spec
git submodule update --remote reference/openapi-schema-rt   # take a newer one
```

## What it covers

Twenty-one resources and twenty-one data sources, one per thing RT's REST2 API
can create, read, update or delete.

| resource | what |
|---|---|
| `rt_ticket` | tickets |
| `rt_queue` | queues |
| `rt_user`, `rt_group`, `rt_group_member`, `rt_user_group` | accounts and membership |
| `rt_customfield`, `rt_customfield_value`, `rt_customfield_appliesto` | custom fields |
| `rt_catalog`, `rt_class`, `rt_asset`, `rt_article` | assets and articles |
| `rt_lifecycle`, `rt_lifecycle_map` | lifecycles |
| `rt_queue_right`, `rt_group_right`, `rt_class_right`, `rt_catalog_right`, `rt_customfield_right`, `rt_global_right` | rights |

The rest of the document is lists, searches and verb paths, which a resource
never calls — see [crossplane-provider-rt]'s README for the accounting; the
grouping is the same.

## Credentials

RT's base path is relative in the document, so the endpoint is the one thing
that always has to be set:

```hcl
provider "rt" {
  endpoint = "https://rt.example.com/REST/2.0"
  token    = "1-14-0123456789abcdef"
}
```

The token goes out as `Authorization: token <token>`, which is RT's own
scheme. `username`/`password` are accepted instead, and `api_key` is sent as
the whole header verbatim for a scheme this client does not spell.

## The generator

`-g rt-terraform` is upstream's `terraform-provider` generator with the hooks
that decide which operations are one resource replaced, built by
`nix build .#openapi-generator-rt` — `javac` against the packaged CLI's own
jar and an SPI entry, no Maven and no checkout of the generator.

It is [terraform-provider-wso2]'s generator with the parts RT's API shape
breaks replaced, and those replacements are [crossplane-provider-rt]'s: both
targets ask the document the same question, which operations are one resource
and which of them is the create, the read, the update and the delete.

- **201 decides a create, not the spelling.** RT POSTs both to create and to
  search. `POST /ticket` creates, `POST /tickets` searches; but `POST
  /lifecycles` creates on a plural path and `POST /customfields` searches on
  one, so no rule about singular and plural survives the whole document. A
  create answers 201 and a search answers 200, which also drops the action
  endpoints (`/lifecycle/{name}/validate`) for free.

- **Two paths, one resource.** RT creates a lifecycle at `POST /lifecycles`
  and addresses it at `/lifecycle/{name}` ever after. Left alone they are two
  groups writing one set of files, the second overwriting the first with half
  a resource.

- **Verbs that look like resources.** `DELETE /queue/{id}/rights/{right}/group/{id}`
  is a member path by shape, so it would become a resource that can only be
  deleted. A collection with neither a create nor a read anywhere in the
  document is a verb, and is dropped.

- **Identifiers are whatever RT feels like.** `"id": 1` in one response and
  `"id": "5"` in the next, both in one array. Every identifier is a
  `client.RTID` — a string with a tolerant unmarshaller — because a create
  whose response will not parse is a resource that exists in RT with nothing
  in state recording it.

- **3.1 validation `anyOf`.** `EmailAddress` is `type: string` with an `anyOf`
  of `{format: email, maxLength: 0}` — an address, or empty. The generator
  emits that composition as a Go type named `AnyOf`, which does not compile.
  The declared type is still on the property and is used; a genuine union
  travels as JSON.

- **A nested create interpolated nothing.** Upstream converts the read, update
  and delete paths to `%v` and leaves the create's `{idOrName}` literal, which
  `fmt.Sprintf` compiles and the server answers 404 for.

`Dockerfile`, `.dockerignore` and `.github/workflows/build-image.yml` are
generated too, from `generators/rt/resources/terraform-provider/`. A provider
that cannot be deployed is not finished.

## How it is deployed

There is no provider registry involved. The image carries the binary and
exists only to be copied out of: `provider-opentofu` runs it as an
initContainer and copies the binary into a filesystem mirror that OpenTofu
resolves the provider from.

```
<mirror>/ghcr.io/n-at-han-k/rt/<version>/linux_amd64/terraform-provider-rt_v<version>
```

## What it does not do yet

- Nested objects and arrays become a `schema.StringAttribute` holding JSON.
  `jsonSupersetOf` keeps a server that merely filled in its own defaults from
  proposing an update forever.
- Nothing is `Computed` because the document says `readOnly` — RT's never
  does. What the create body takes is what a person may write, and what only
  the response answers is computed; that inference is the whole rule.
- `examples/provider/provider.tf` is upstream's and repeats the document's
  relative base path. Set a real endpoint, as above.
- A right cannot be revoked. Granting is `POST …/rights`; revoking needs
  `DELETE …/rights/{right}/group/{id}`, whose path the generator cannot
  derive from the grant it was given.

[Request Tracker]: https://bestpractical.com/request-tracker
[n-at-han-k/openapi-schema-rt]: https://github.com/n-at-han-k/openapi-schema-rt
[rt/request-tracker-openapi]: https://github.com/bestpractical/request-tracker-openapi
[crossplane-provider-rt]: https://github.com/n-at-han-k/crossplane-provider-rt
[terraform-provider-wso2]: https://github.com/n-at-han-k/terraform-provider-wso2
