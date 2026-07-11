(() => {
  const status = document.querySelector("#status");
  const app = document.querySelector("#app");
  const userID = document.querySelector("#user-id");
  const editor = document.querySelector("#document");
  let csrfToken = "";

  async function request(path, options = {}) {
    const headers = new Headers(options.headers || {});
    if (csrfToken && options.method && options.method !== "GET") {
      headers.set("X-CSRF-Token", csrfToken);
    }
    const response = await fetch(path, { ...options, headers });
    if (response.status === 401) {
      window.location.assign("/auth/login?return_to=/");
      throw new Error("authentication required");
    }
    if (!response.ok) throw new Error(`request failed: ${response.status}`);
    return response.json();
  }

  async function loadObject() {
    const value = await request("/api/object");
    editor.value = JSON.stringify(value, null, 2);
  }

  async function bootstrap() {
    try {
      const session = await request("/auth/session");
      csrfToken = session.csrfToken;
      userID.textContent = session.userId;
      await loadObject();
      status.classList.add("d-none");
      app.classList.remove("d-none");
    } catch (error) {
      status.textContent = String(error.message || error);
      status.className = "alert alert-danger";
    }
  }

  document.querySelector("#reload").addEventListener("click", () => loadObject().catch(showError));
  document.querySelector("#save").addEventListener("click", async () => {
    try {
      const documentValue = JSON.parse(editor.value);
      const saved = await request("/api/object", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(documentValue)
      });
      editor.value = JSON.stringify(saved, null, 2);
      status.textContent = "Saved";
      status.className = "alert alert-success";
    } catch (error) {
      showError(error);
    }
  });

  function showError(error) {
    status.textContent = String(error.message || error);
    status.className = "alert alert-danger";
  }

  bootstrap();
})();
