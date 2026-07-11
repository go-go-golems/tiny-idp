const MAX_DOCUMENT_BYTES = 32 * 1024;
const MAX_KEYS = 64;
const MAX_DEPTH = 8;

function validateDocument(value) {
  const encoded = JSON.stringify(value);
  if (encoded.length > MAX_DOCUMENT_BYTES) {
    throw new Error("document exceeds encoded size limit");
  }
  let keys = 0;
  const visit = (current, depth) => {
    if (depth > MAX_DEPTH) throw new Error("document exceeds nesting limit");
    if (current === null || typeof current !== "object") return;
    if (Array.isArray(current)) {
      current.forEach((item) => visit(item, depth + 1));
      return;
    }
    for (const key of Object.keys(current)) {
      keys += 1;
      if (keys > MAX_KEYS) throw new Error("document exceeds key limit");
      visit(current[key], depth + 1);
    }
  };
  visit(value, 0);
  return JSON.parse(encoded);
}

class UserState {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  fetch(request) {
    if (request.method === "GET" && request.path === "/state") {
      return { status: 200, body: this.state.storage.get("document") || {} };
    }
    if (request.method === "POST" && request.path === "/state") {
      const document = validateDocument(request.body || {});
      this.state.storage.put("document", document);
      return { status: 200, body: document };
    }
    return { status: 404, body: { error: "not_found" } };
  }
}

exports.objects = { UserState };
