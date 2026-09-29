import os

import requests
from langchain_anthropic import ChatAnthropic


API_KEY = os.environ["API_KEY"]
BASE_URL = os.getenv("GATEWAY_CLAUDE_BASE_URL", "http://127.0.0.1:4981/claude")


def discovered_model() -> str:
    response = requests.get(
        f"{BASE_URL}/v1/models",
        headers={"x-api-key": API_KEY},
        timeout=30,
    )
    response.raise_for_status()
    models = response.json().get("data", [])
    if not models:
        raise RuntimeError("The gateway returned no models; check Gemini session readiness")
    first = models[0]
    return os.getenv("GEMINI_MODEL", first.get("id") or first.get("name"))


llm = ChatAnthropic(
    base_url=BASE_URL,
    model=discovered_model(),
    temperature=0.7,
    api_key=API_KEY,
)
response = llm.invoke("Can you introduce yourself?")
print(response.content)
