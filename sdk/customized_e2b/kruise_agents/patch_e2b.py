import os

from e2b import ConnectionConfig
from e2b.sandbox.main import SandboxBase
from e2b_code_interpreter.code_interpreter_sync import Sandbox as SandboxSync
from e2b_code_interpreter.code_interpreter_sync import JUPYTER_PORT


def __sandbox_get_host(self, port: int) -> str:
    return f"{self.sandbox_domain}/kruise/{self.sandbox_id}/{port}"


def __connection_config_get_host(_, sandbox_id: str, sandbox_domain: str, port: int) -> str:
    return f"{sandbox_domain}/kruise/{sandbox_id}/{port}"


def __get_api_url(https: bool):
    return f"{'https' if https else 'http'}://{os.environ['E2B_DOMAIN']}/kruise/api"


def __connection_config_get_sandbox_url_http(self, sandbox_id: str, sandbox_domain: str) -> str:
    return f"http://{__connection_config_get_host(self, sandbox_id, sandbox_domain, ConnectionConfig.envd_port)}"


def __jupyter_url_http(self) -> str:
    return f"http://{__sandbox_get_host(self, JUPYTER_PORT)}"

def patch_e2b(https: bool = True, bypass_key_validation: bool = False):
    """Patch E2B SDK to work with OpenKruise Agents.

    Args:
        https: Whether to use HTTPS for the API URL.
        bypass_key_validation: If True, disable client-side API key format
            validation added in newer E2B SDK versions. This allows using
            keys that don't match the ``e2b_<hex>`` format.
    """
    os.environ["E2B_API_URL"] = __get_api_url(https)
    SandboxBase.get_host = __sandbox_get_host
    ConnectionConfig.get_host = __connection_config_get_host
    if not https:
        ConnectionConfig.get_sandbox_url = __connection_config_get_sandbox_url_http
        setattr(SandboxSync, '_jupyter_url', property(__jupyter_url_http))
    if bypass_key_validation:
        try:
            import e2b.api as _e2b_api
            _e2b_api.validate_api_key = lambda _key: None
        except (ImportError, AttributeError):
            pass
