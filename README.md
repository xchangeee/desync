# desync

Declarative management of [deSEC](https://desec.io) DNS resources via the deSEC REST API.

Define your desired state in a JSON file, then use `plan` to preview changes and `apply` to execute them — similar to Terraform, but purpose-built for deSEC.

## What it manages

| Resource | Behaviour |
|---|---|
| **Token policies** | Fully reconciled per token: creates, updates, and deletes policies to exactly match the file. |
| **RRsets** | Fully reconciled per domain: creates, updates, and deletes records to exactly match the file. All changes for a domain are sent as a single bulk request to minimise rate-limit consumption. |
| Tokens | Not managed — secrets are shown only once at creation time. Use `tokens list` / `tokens create` to provision tokens and discover their IDs. |
| Domains | Not managed — create them via the deSEC web UI. desync only manages the records inside them. |

## Installation

```sh
go install codeberg.org/xchangeee/desync/cmd/desync@latest
```

Or build from source:

```sh
git clone https://codeberg.org/xchangeee/desync
cd desync
make build-bin
```

## Authentication

All commands require a deSEC API token. Pass it with `-token` or set `DESEC_TOKEN` in your environment:

```sh
export DESEC_TOKEN=your-token-here
```

The token needs `perm_manage_tokens` to manage token policies, and write access to the relevant domains for RRset changes.

## Quick start

```sh
# 1. Bootstrap a state file from your current account
desync init

# 2. Edit desync.json to reflect the desired state

# 3. Preview changes
desync plan

# 4. Apply
desync apply
```

## Subcommands

### `init`

Fetches your current account state from the API and writes it to a state file. Safe to run on a fresh checkout — it will not overwrite an existing file unless you pass `-force`.

```sh
desync init                   # writes desync.json
desync init -f prod.json      # custom output path
desync init -force            # overwrite existing file
```

TTL values equal to the default (3600) are omitted from the output to keep the file readable.

### `plan`

Shows what changes would be made without touching anything.

```sh
desync plan
desync plan -f prod.json
desync plan -f -          # read state from stdin
```

Example output:

```
Fetching current state...

RRsets:
  Domain "example.com":
    + create  mail                    MX
              ttl=3600 records=[10 mail.example.com.]
    ~ update  www                     A
        ~ records: [1.2.3.4] -> [5.6.7.8]
    - delete  legacy                  CNAME

Plan: 1 to create, 1 to update, 1 to delete.
```

### `apply`

Runs `plan`, then prompts for confirmation before making any API calls.

```sh
desync apply
desync apply -auto-approve    # skip confirmation (useful in CI)
desync apply -f prod.json
desync apply -f -             # read state from stdin
```

### `tokens list`

Lists all tokens in your account with their UUIDs, then prints the policies for each token that has any. Use this to find the `tokenId` to reference in your state file.

```sh
desync tokens list
```

```
ID                                    NAME       CREATED               LAST USED             VALID  PERM_CREATE  PERM_DELETE  PERM_MGMT
3a6b94b5-d20e-40bd-a7cc-521f5c79fab3  ci-deploy  2024-01-15T10:00:00Z  2024-03-01T08:12:00Z  true   false        false        false

Policies for ci-deploy (3a6b94b5-d20e-40bd-a7cc-521f5c79fab3):
  DOMAIN       SUBNAME  TYPE  PERM_WRITE
  *            *        *     false
  example.com  *        *     true
```

### `tokens create`

Creates a new token and prints the secret **once**. Store it immediately.

```sh
desync tokens create -name ci-deploy
desync tokens create -name admin -perm-create-domain -perm-delete-domain -perm-manage-tokens
```

```
Token created.
  ID:     3a6b94b5-d20e-40bd-a7cc-521f5c79fab3
  Name:   ci-deploy
  SECRET: 4pnk7u-NHvrEkFzrhFDRTjGFyX_S

The secret is shown only once. Store it securely.
Reference this token in your state file with tokenId: "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3"
```

## State file reference

The state file is JSON with two top-level keys. Both are optional — omit a section if you don't want desync to manage that resource type.

```json
{
  "tokenPolicies": [...],
  "domains": [...]
}
```

### `domains`

Declares the desired RRsets for one or more DNS zones. desync fully reconciles each listed domain: RRsets in the file are created or updated, and any RRset present in the API but absent from the file is **deleted**.

Domains not listed here are left completely untouched.

```json
"domains": [
  {
    "name": "example.com",
    "rrsets": [
      {"subname": "",    "type": "A",   "records": ["1.2.3.4"]},
      {"subname": "www", "type": "A",   "records": ["1.2.3.4"]},
      {"subname": "www", "type": "AAAA","records": ["2001:db8::1"]},
      {"subname": "mail","type": "MX",  "records": ["10 mail.example.com."]},
      {"subname": "",    "type": "TXT", "records": ["\"v=spf1 mx ~all\""]}
    ]
  },
  {
    "name": "example.org",
    "rrsets": [
      {"subname": "", "type": "A", "records": ["1.2.3.4"]}
    ]
  }
]
```

#### RRset fields

| Field | Type | Required | Description |
|---|---|---|---|
| `subname` | string | yes | DNS label relative to the domain apex. Empty string `""` refers to the apex itself. |
| `type` | string | yes | Record type in uppercase: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `CAA`, `SRV`, `TLSA`, etc. |
| `ttl` | integer | no | Time-to-live in seconds. Defaults to `3600` when omitted. |
| `records` | array of strings | yes | Record values in zone-file presentation format. |

#### Common record examples

```json
{"subname": "",         "type": "A",    "records": ["1.2.3.4"]}
{"subname": "",         "type": "AAAA", "records": ["2001:db8::1"]}
{"subname": "www",      "type": "CNAME","records": ["example.com."]}
{"subname": "",         "type": "MX",   "records": ["10 mail.example.com.", "20 mail2.example.com."]}
{"subname": "",         "type": "TXT",  "records": ["\"v=spf1 include:example.net ~all\""]}
{"subname": "_dmarc",   "type": "TXT",  "records": ["\"v=DMARC1; p=reject; rua=mailto:dmarc@example.com\""]}
{"subname": "",         "type": "CAA",  "records": ["0 issue \"letsencrypt.org\""]}
{"subname": "_acme-challenge", "type": "TXT", "ttl": 60, "records": ["\"some-challenge-value\""]}
```

> Record values that include a trailing dot (CNAME, MX, NS targets) **must** include the dot. TXT record values **must** be wrapped in escaped double quotes.

### `tokenPolicies`

Declares the complete desired policy set for one or more existing tokens. desync will create, update, and delete policies to exactly match what is listed. Tokens with no entry here are left untouched.

Each entry references a token by its UUID (`tokenId`), which you can find with `desync tokens list`.

```json
"tokenPolicies": [
  {
    "tokenId": "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3",
    "policies": [
      {"domain": null, "subname": null, "type": null, "permWrite": false},
      {"domain": "example.com", "subname": null, "type": null, "permWrite": true}
    ]
  }
]
```

#### Policy fields

| Field | Type | Description |
|---|---|---|
| `domain` | string or `null` | Domain the policy applies to. `null` = default (wildcard). |
| `subname` | string or `null` | Subname the policy applies to. `null` = all subnames. |
| `type` | string or `null` | Record type the policy applies to. `null` = all types. |
| `permWrite` | bool | Whether this token may write matching RRsets. |

#### Policy matching

The deSEC API uses longest-prefix matching on the `(domain, subname, type)` triple, where `null` acts as a wildcard at each position.

The **default policy** is the entry where `domain`, `subname`, and `type` are all `null`. It acts as a catch-all for anything not matched by a more specific rule. The API requires it to be present whenever any other policies exist — desync handles the required creation/deletion ordering automatically.

Set the default policy's `permWrite` to `false` for a restrictive baseline (deny all, then allow specific domains), or `true` if the token should have broad write access and you only need to carve out exceptions.

Restrictive baseline — deny everywhere, allow one domain:

```json
"policies": [
  {"domain": null,         "subname": null, "type": null, "permWrite": false},
  {"domain": "example.com","subname": null, "type": null, "permWrite": true}
]
```

Permissive baseline — allow everywhere, restrict one domain:

```json
"policies": [
  {"domain": null,         "subname": null, "type": null, "permWrite": true},
  {"domain": "example.com","subname": null, "type": null, "permWrite": false}
]
```

To allow only specific record types (e.g. for a dynDNS token):

```json
"policies": [
  {"domain": null,          "subname": null, "type": null, "permWrite": false},
  {"domain": "example.com", "subname": null, "type": "A",  "permWrite": true},
  {"domain": "example.com", "subname": null, "type": "AAAA","permWrite": true}
]
```

## Rate limiting

The deSEC API applies rate limits per account. desync handles `429 Too Many Requests` responses automatically by reading the `Retry-After` header and sleeping before retrying. RRset changes are batched per domain into a single bulk request to minimise the number of write operations counted against the per-domain rate limit.

## Full example

```json
{
  "tokenPolicies": [
    {
      "tokenId": "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3",
      "policies": [
        {"domain": null,          "subname": null, "type": null, "permWrite": false},
        {"domain": "example.com", "subname": null, "type": null, "permWrite": true},
        {"domain": "example.org", "subname": null, "type": null, "permWrite": true}
      ]
    }
  ],
  "domains": [
    {
      "name": "example.com",
      "rrsets": [
        {"subname": "",       "type": "A",   "records": ["1.2.3.4"]},
        {"subname": "",       "type": "AAAA","records": ["2001:db8::1"]},
        {"subname": "www",    "type": "A",   "records": ["1.2.3.4"]},
        {"subname": "www",    "type": "AAAA","records": ["2001:db8::1"]},
        {"subname": "",       "type": "MX",  "records": ["10 mail.example.com."]},
        {"subname": "",       "type": "TXT", "records": ["\"v=spf1 mx ~all\""]},
        {"subname": "_dmarc", "type": "TXT", "records": ["\"v=DMARC1; p=reject\""]},
        {"subname": "",       "type": "CAA", "records": ["0 issue \"letsencrypt.org\""]}
      ]
    },
    {
      "name": "example.org",
      "rrsets": [
        {"subname": "", "type": "A", "records": ["1.2.3.4"]}
      ]
    }
  ]
}
```
