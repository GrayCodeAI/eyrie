"""Eyrie Python SDK — conversation DAG client."""

import json
import httpx
from typing import Generator, Optional


class EyrieClient:
    def __init__(self, base_url: str = "http://localhost:8080", api_key: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.headers = {"Content-Type": "application/json"}
        if api_key:
            self.headers["Authorization"] = f"Bearer {api_key}"

    def prompt(self, message: str, model: str = "", system_prompt: str = "", max_tokens: int = 0) -> dict:
        body = {"message": message}
        if model:
            body["model"] = model
        if system_prompt:
            body["system_prompt"] = system_prompt
        if max_tokens:
            body["max_tokens"] = max_tokens
        resp = httpx.post(f"{self.base_url}/prompt", json=body, headers=self.headers, timeout=120)
        resp.raise_for_status()
        return resp.json()

    def prompt_from(self, node_id: str, message: str, model: str = "") -> dict:
        body = {"message": message}
        if model:
            body["model"] = model
        resp = httpx.post(f"{self.base_url}/nodes/{node_id}/prompt", json=body, headers=self.headers, timeout=120)
        resp.raise_for_status()
        return resp.json()

    def stream_prompt(self, message: str, model: str = "") -> Generator[dict, None, None]:
        body = {"message": message, "stream": True}
        if model:
            body["model"] = model
        with httpx.stream("POST", f"{self.base_url}/prompt", json=body, headers=self.headers, timeout=120) as resp:
            resp.raise_for_status()
            for line in resp.iter_lines():
                if line.startswith("data: "):
                    yield json.loads(line[6:])

    def list_conversations(self) -> list:
        resp = httpx.get(f"{self.base_url}/nodes", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def get_node(self, node_id: str) -> dict:
        resp = httpx.get(f"{self.base_url}/nodes/{node_id}", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def get_tree(self, node_id: str) -> list:
        resp = httpx.get(f"{self.base_url}/nodes/{node_id}/tree", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def delete_node(self, node_id: str) -> dict:
        resp = httpx.delete(f"{self.base_url}/nodes/{node_id}", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def create_alias(self, node_id: str, alias: str) -> dict:
        resp = httpx.put(f"{self.base_url}/nodes/{node_id}/aliases/{alias}", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def delete_alias(self, alias: str) -> dict:
        resp = httpx.delete(f"{self.base_url}/aliases/{alias}", headers=self.headers, timeout=30)
        resp.raise_for_status()
        return resp.json()

    def health(self) -> dict:
        resp = httpx.get(f"{self.base_url}/health", timeout=5)
        resp.raise_for_status()
        return resp.json()
