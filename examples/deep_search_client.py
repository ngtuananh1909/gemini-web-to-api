import json
import os

import requests


BASE = os.getenv("GATEWAY_GEMINI_BASE_URL", "http://127.0.0.1:4981/gemini/v1beta")
API_KEY = os.environ["API_KEY"]
HEADERS = {"Authorization": f"Bearer {API_KEY}"}


def discovered_model() -> str:
    response = requests.get(f"{BASE}/models", headers=HEADERS, timeout=30)
    response.raise_for_status()
    payload = response.json()
    models = payload.get("models", payload.get("data", []))
    names = [model.get("name") or model.get("id") for model in models]
    names = [name.rsplit("/", 1)[-1] for name in names if name]
    if not names:
        raise RuntimeError("The gateway returned no models; check Gemini session readiness")
    return os.getenv("GEMINI_RESEARCH_MODEL") or next(
        (name for name in names if "research" in name.lower()), names[0]
    )


with requests.post(
    f"{BASE}/deepresearch/stream",
    json={
        "query": "Compare Elasticsearch vs OpenSearch architecture and performance",
        "model": discovered_model(),
    },
    stream=True,
    headers={**HEADERS, "Accept": "text/event-stream"},
    timeout=360,
) as response:
    response.raise_for_status()
    for line in response.iter_lines(decode_unicode=True):
        if not line or not line.startswith("data:"):
            continue
        event = json.loads(line[5:].strip())

        status = event.get("event", "")

        if status == "progress":
            print("[...] Researching...", end="\r", flush=True)

        elif status == "result":
            result = event.get("result") or {}
            print("\n" + result.get("summary", "No summary returned"))
            break

        elif status == "error":
            print("Error:", event.get("error"))
            break
