import base64
import os
from pathlib import Path

from openai import OpenAI


API_KEY = os.environ["API_KEY"]
BASE_URL = os.getenv("GATEWAY_OPENAI_BASE_URL", "http://127.0.0.1:4981/openai/v1")
client = OpenAI(base_url=BASE_URL, api_key=API_KEY)


def discovered_model() -> str:
    models = client.models.list().data
    if not models:
        raise RuntimeError("The gateway returned no models; check Gemini session readiness")
    return os.getenv("GEMINI_MODEL", models[0].id)


def image_generation_example() -> None:
    # The model list is account-specific. Set GEMINI_MODEL when the account has
    # more than one image-capable model and you want a particular choice.
    response = client.images.generate(
        model=discovered_model(),
        prompt="A cinematic cyberpunk rabbit wearing a yellow raincoat, neon city, high detail",
        n=1,
        size="1024x1024",
    )

    image = response.data[0]
    if image.url:
        print("Generated image URL:", image.url)
        return

    if image.b64_json:
        output_path = Path(__file__).with_name("generated_image.png")
        output_path.write_bytes(base64.b64decode(image.b64_json))
        print("Generated image saved to:", output_path)


if __name__ == "__main__":
    image_generation_example()
