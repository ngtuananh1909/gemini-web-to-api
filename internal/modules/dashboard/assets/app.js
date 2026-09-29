const baseUrl = `${window.location.origin}/openai/v1`;
const state = { key: "", authEnabled: true, models: [], status: "connecting", lastModelsAt: 0 };
const byId = (id) => document.getElementById(id);

byId("base-url").textContent = baseUrl;
byId("claude-url").textContent = `${window.location.origin}/claude/v1`;
byId("gemini-url").textContent = `${window.location.origin}/gemini/v1beta`;

function notice(message, isError = false) {
  const element = byId("notice");
  element.textContent = message;
  element.classList.toggle("error", isError);
}

function apiHeaders() {
  return state.authEnabled ? { Authorization: `Bearer ${state.key}` } : {};
}

async function jsonRequest(path, options = {}) {
  const response = await fetch(path, { cache: "no-store", ...options });
  let body = {};
  try {
    body = await response.json();
  } catch {
    body = {};
  }
  if (!response.ok) {
    const message = typeof body.error === "string" ? body.error : body.error?.message;
    throw new Error(message || `Request failed (${response.status}).`);
  }
  return body;
}

async function loadConnection() {
  const connection = await jsonRequest("/dashboard/api/connection");
  state.key = connection.apiKey || "";
  state.authEnabled = Boolean(connection.authEnabled);
  byId("api-key").textContent = state.authEnabled ? state.key : "Authentication disabled";
  byId("rotate-key").disabled = !connection.rotationAllowed;
  document.querySelector('[data-copy="api-key"]').disabled = !state.authEnabled;
}

function preferredModel() {
  try {
    return window.localStorage.getItem("gatewayPreferredModel") || "";
  } catch {
    return "";
  }
}

function rememberModel(model) {
  try {
    window.localStorage.setItem("gatewayPreferredModel", model);
  } catch {
  }
}

function showModels(models) {
  const select = byId("model-select");
  const previous = select.value || preferredModel();
  select.replaceChildren();
  state.models = models;
  if (models.length === 0) {
    select.add(new Option("Waiting for models…", ""));
    select.disabled = true;
    byId("send-button").disabled = true;
    byId("copy-config").disabled = true;
    document.querySelector('[data-copy="model-select"]').disabled = true;
    return;
  }
  for (const model of models) {
    select.add(new Option(model.id, model.id));
  }
  select.value = models.some((model) => model.id === previous) ? previous : models[0].id;
  select.disabled = false;
  byId("copy-config").disabled = false;
  document.querySelector('[data-copy="model-select"]').disabled = false;
  byId("send-button").disabled = state.status !== "connected";
}

async function loadModels() {
  const result = await jsonRequest(`${baseUrl}/models`, { headers: apiHeaders() });
  const models = Array.isArray(result.data) ? result.data.filter((model) => typeof model.id === "string") : [];
  showModels(models);
  state.lastModelsAt = Date.now();
}

async function loadStatus() {
  try {
    const result = await jsonRequest("/dashboard/api/status");
    state.status = result.status || "error";
    const badge = byId("status-badge");
    badge.textContent = state.status.charAt(0).toUpperCase() + state.status.slice(1);
    badge.className = `status-badge ${state.status}`;
    byId("status-message").textContent = result.message || "Gemini Web status is unavailable.";
    byId("send-button").disabled = state.status !== "connected" || state.models.length === 0;
    if (state.status === "connected" && Date.now() - state.lastModelsAt > 30000) {
      await loadModels();
    }
  } catch (error) {
    state.status = "error";
    byId("status-badge").textContent = "Error";
    byId("status-badge").className = "status-badge error";
    byId("status-message").textContent = error.message;
    byId("send-button").disabled = true;
  }
}

async function copyText(value) {
  if (!value) {
    notice("Nothing to copy yet.", true);
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    notice("Copied to clipboard.");
  } catch {
    notice("Clipboard unavailable. Select and copy the value manually.", true);
  }
}

document.querySelectorAll("[data-copy]").forEach((button) => {
  button.addEventListener("click", () => {
    const target = byId(button.dataset.copy);
    copyText(target.tagName === "SELECT" ? target.value : target.textContent);
  });
});

byId("copy-config").addEventListener("click", () => {
  copyText(`Base URL: ${baseUrl}\nAPI Key: ${state.authEnabled ? state.key : "Authentication disabled"}\nModel: ${byId("model-select").value}`);
});

byId("model-select").addEventListener("change", (event) => rememberModel(event.target.value));

byId("test-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const prompt = byId("prompt").value.trim();
  const model = byId("model-select").value;
  if (!prompt || !model) return;
  const button = byId("send-button");
  button.disabled = true;
  button.textContent = "Sending…";
  byId("test-output").textContent = "Waiting for Gemini…";
  byId("response-model").textContent = "";
  try {
    const result = await jsonRequest(`${baseUrl}/chat/completions`, {
      method: "POST",
      headers: { ...apiHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ model, messages: [{ role: "user", content: prompt }] }),
    });
    const content = result.choices?.[0]?.message?.content;
    byId("test-output").textContent = typeof content === "string" ? content : JSON.stringify(content ?? "No text returned.");
    byId("response-model").textContent = result.model || model;
  } catch (error) {
    byId("test-output").textContent = error.message;
    notice("The test request failed.", true);
  } finally {
    button.disabled = state.status !== "connected" || state.models.length === 0;
    button.textContent = "Send prompt ↗";
  }
});

byId("rotate-key").addEventListener("click", async () => {
  if (!window.confirm("Rotate the gateway API key? Existing clients using the old key will stop working.")) return;
  try {
    await jsonRequest("/dashboard/api/api-key/rotate", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Dashboard-Action": "rotate" },
      body: "{}",
    });
    await loadConnection();
    notice("Gateway API key rotated. Update your clients with the new key.");
  } catch (error) {
    notice(error.message, true);
  }
});

(async () => {
  try {
    await loadConnection();
    await loadStatus();
  } catch (error) {
    notice(error.message, true);
    byId("status-message").textContent = "The local dashboard could not load gateway settings.";
  }
  window.setInterval(loadStatus, 8000);
})();
