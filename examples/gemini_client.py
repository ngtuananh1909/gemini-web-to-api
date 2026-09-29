import os
from pathlib import Path

from google import genai
from google.genai import types


API_KEY = os.environ["API_KEY"]
client = genai.Client(
    api_key=API_KEY,
    http_options={
        "base_url": os.getenv("GATEWAY_GEMINI_BASE_URL", "http://127.0.0.1:4981/gemini"),
        "api_version": "v1beta",
    },
)


models = list(client.models.list())
if not models:
    raise RuntimeError("The gateway returned no models; check Gemini session readiness")

# Model names are discovered from the signed-in Gemini Web account. Set
# GEMINI_MODEL to select a particular ID from the list when needed.
model_name = os.getenv("GEMINI_MODEL", models[0].name)
print("Available models:", [model.name for model in models])

image_path = Path(__file__).with_name("fiber.png")
response = client.models.generate_content(
    model=model_name,
    contents=[
        types.Content(
            role="user",
            parts=[
                types.Part.from_text(text="Describe this image in detail."),
                types.Part.from_bytes(
                    data=image_path.read_bytes(),
                    mime_type="image/png",
                ),
            ],
        )
    ],
)

print(response.text)
