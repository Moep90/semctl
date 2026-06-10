# Error classes

`semctl` errors use stable error codes in the format `SEMDDDNNN`: `SEM` product prefix, `DDD` domain, `NNN` ordinal within the domain. The HTTP status (when any) is carried as metadata, not encoded in the code.

In `--output json`/`yaml` the `error` field is a structured object (`code`, `name`, `title`, `message`, `hint`, `retryable`, `exit_code`, `http_status`, `metadata`). The process currently exits `1` for all failures; the per-class `exit_code` is informational.

> This file is generated from the registry. Do not edit by hand; run `go generate ./internal/semerr/...`.

## Generic

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM000001 | UNKNOWN_ERROR | Unknown error | No | 1 |
| SEM000002 | INTERNAL_INVARIANT_VIOLATION | Internal invariant violation | No | 1 |
| SEM000003 | UNIMPLEMENTED | Not implemented | No | 1 |

## CLI usage

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM100001 | INVALID_ARGUMENT | Invalid argument | No | 2 |
| SEM100002 | MISSING_ARGUMENT | Missing argument | No | 2 |
| SEM100003 | INVALID_FLAG | Invalid flag | No | 2 |
| SEM100004 | UNSUPPORTED_OUTPUT_FORMAT | Unsupported output format | No | 2 |
| SEM100005 | COMMAND_USAGE_ERROR | Command usage error | No | 2 |

## Config

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM200001 | CONFIG_NOT_FOUND | Config not found. Run `semctl auth login` or set --host/SEMAPHORE_HOST. | No | 3 |
| SEM200002 | CONFIG_INVALID | Config invalid | No | 3 |
| SEM200003 | PROFILE_NOT_FOUND | Profile not found | No | 3 |
| SEM200004 | CONFIG_PERMISSION_DENIED | Config permission denied | No | 3 |
| SEM200005 | ENVIRONMENT_INVALID | Environment invalid | No | 3 |

## Auth

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM300001 | AUTH_TOKEN_MISSING | Authentication token missing. Run `semctl auth login` or set SEMAPHORE_TOKEN. | No | 4 |
| SEM300002 | AUTH_TOKEN_EXPIRED | Authentication token expired. Re-authenticate with `semctl auth login`. | No | 4 |
| SEM300003 | AUTH_TOKEN_INVALID | Authentication token invalid | No | 4 |
| SEM300004 | AUTH_SCOPE_INSUFFICIENT | Insufficient scope | No | 4 |
| SEM300005 | AUTH_INTERACTIVE_REQUIRED | Interactive authentication required | No | 4 |

## Network / transport

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM400001 | NETWORK_DNS_FAILURE | DNS resolution failed. Check the host name and your network/DNS. | Yes | 5 |
| SEM400002 | NETWORK_TIMEOUT | Network timeout | Yes | 5 |
| SEM400003 | TLS_HANDSHAKE_FAILED | TLS handshake failed | No | 5 |
| SEM400004 | PROXY_ERROR | Proxy error | No | 5 |
| SEM400005 | CONNECTION_REFUSED | Connection refused. Check that the host is reachable and the port is correct. | Yes | 5 |
| SEM400006 | CONNECTION_RESET | Connection reset | Yes | 5 |

## API response

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM500001 | API_BAD_REQUEST | API request rejected | No | 6 |
| SEM500002 | API_UNAUTHENTICATED | API authentication required. Run `semctl auth login` or check SEMAPHORE_TOKEN. | No | 6 |
| SEM500003 | API_FORBIDDEN | API access forbidden. Check that your token has access to this resource. | No | 6 |
| SEM500004 | API_RESOURCE_NOT_FOUND | API resource not found. Check the resource ID, endpoint path, and that your token has access to it. | No | 44 |
| SEM500005 | API_CONFLICT | API conflict | No | 6 |
| SEM500006 | API_RATE_LIMITED | API rate limited. Wait and retry; see retry_after if present. | Yes | 75 |
| SEM500007 | API_SERVER_ERROR | API server error | Yes | 75 |
| SEM500008 | API_BAD_GATEWAY | API bad gateway | Yes | 75 |
| SEM500009 | API_SERVICE_UNAVAILABLE | API service unavailable | Yes | 75 |
| SEM500010 | API_GATEWAY_TIMEOUT | API gateway timeout | Yes | 75 |
| SEM500011 | API_UNEXPECTED_STATUS | Unexpected API status | No | 6 |

## Serialization

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM600001 | RESPONSE_DECODE_FAILED | Response decode failed | No | 7 |
| SEM600002 | RESPONSE_SCHEMA_INVALID | Response schema invalid | No | 7 |
| SEM600003 | UNSUPPORTED_CONTENT_TYPE | Unsupported content type | No | 7 |
| SEM600004 | EMPTY_RESPONSE | Empty response | No | 7 |

## Local runtime

| Code | Name | Meaning | Retryable | Exit |
|---|---|---|---:|---:|
| SEM700001 | FILE_NOT_FOUND | File not found | No | 8 |
| SEM700002 | FILE_PERMISSION_DENIED | File permission denied | No | 8 |
| SEM700003 | COMMAND_EXECUTION_FAILED | Command execution failed | No | 8 |
| SEM700004 | LOCAL_STATE_LOCKED | Local state locked | Yes | 8 |
