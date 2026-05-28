# Customized E2B SDK patch

This Python library patches the E2B client, converting the native E2B protocol to the OpenKruise Agents private
protocol, thereby simplifying sandbox-manager deployment.

## Problem Statement

The E2B SDK requests the backend using the following protocol:

| Protocol                    | Description          | Example                                |
|-----------------------------|----------------------|----------------------------------------|
| api.E2B_DOMAIN              | Management interface | api.e2b.dev                            |
| \<port\>-\<sid\>.E2B_DOMAIN | Sandbox interface    | 49999-i37sc83s52e2cv85h636jjgs.e2b.dev |

Meanwhile, E2B SDK forces the use of HTTPS.

In our practice, we found that in K8s scenarios, this protocol has the following issues:

1. Requires configuring wildcard domain resolution to the management service (sandbox-manager), unable to use methods
   like hosts for resolution.
2. Requires using expensive wildcard certificates.

The above issues simultaneously make deploying a backend service compatible with E2B have a high threshold: not only
increasing user costs, but also making it difficult to automate the setup of an E2E test environment.

## Usage

Requirements:

- e2b >= 2.8.1

```python
from kruise_agents.patch_e2b import patch_e2b
from e2b_code_interpreter import Sandbox

patch_e2b(https=False)  # patch sdk

if __name__ == "__main__":
    with Sandbox.create() as sbx:
        sbx.run_code("print('hello world')")
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `https` | bool | `True` | Whether to use HTTPS for the API URL. |
| `bypass_key_validation` | bool | `False` | Set to `True` when using legacy keys that don't match the `e2b_<hex>` format. |

### API Key Format

Starting from E2B SDK v2.25.0, the SDK validates API keys client-side. Keys must match the pattern `e2b_` followed by
hex characters.

OpenKruise Agents sandbox-manager now generates keys in this format by default. If you have legacy API keys (e.g. UUID
format), use:

```python
patch_e2b(https=False, bypass_key_validation=True)
```
