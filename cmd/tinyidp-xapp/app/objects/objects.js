const MAX_DOCUMENT_BYTES = 32 * 1024;
const MAX_KEYS = 64;
const MAX_DEPTH = 8;
const BBS_MAX_POSTS = 200;
const BBS_MAX_REPLIES = 100;
const BBS_CATEGORIES = new Set(["general", "projects", "questions", "notes"]);

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

function requiredTrustedText(value, field, maxLength) {
  if (typeof value !== "string") throw new Error(`${field} must be text`);
  const normalized = value.trim();
  if (!normalized) throw new Error(`${field} is required`);
  if (normalized.length > maxLength) throw new Error(`${field} exceeds ${maxLength} characters`);
  return normalized;
}

function publicInputText(value, field, maxLength) {
  if (typeof value !== "string") {
    return { error: `${field}_must_be_text` };
  }
  const normalized = value.trim();
  if (!normalized) return { error: `${field}_required` };
  if (normalized.length > maxLength) return { error: `${field}_too_long` };
  return { value: normalized };
}

function boardState(storage) {
  const current = storage.get("board");
  if (!current) {
    return { version: 1, nextPostId: 1, nextReplyId: 1, posts: [] };
  }
  if (
    current.version !== 1 ||
    !Number.isSafeInteger(current.nextPostId) || current.nextPostId < 1 ||
    !Number.isSafeInteger(current.nextReplyId) || current.nextReplyId < 1 ||
    !Array.isArray(current.posts)
  ) {
    throw new Error("stored board schema is invalid");
  }
  return current;
}

function sequenceID(prefix, sequence) {
  return `${prefix}_${String(sequence).padStart(12, "0")}`;
}

function publicBoard(board, viewerId) {
  const posts = board.posts.slice().reverse().map((post) => ({
    id: post.id,
    title: post.title,
    body: post.body,
    category: post.category,
    author: post.authorName,
    createdAt: post.createdAt,
    canDelete: post.authorId === viewerId,
    replies: post.replies.map((reply) => ({
      id: reply.id,
      body: reply.body,
      author: reply.authorName,
      createdAt: reply.createdAt
    }))
  }));
  return {
    name: "Local Loop",
    description: "A small persistent board for notes, questions, and project dispatches.",
    posts,
    stats: {
      posts: posts.length,
      replies: posts.reduce((total, post) => total + post.replies.length, 0)
    }
  };
}

class BBS {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  fetch(request) {
    const body = request.body || {};
    const board = boardState(this.state.storage);

    if (request.method === "GET" && request.path === "/board") {
      const actorId = requiredTrustedText(body.actorId, "actorId", 256);
      return { status: 200, body: publicBoard(board, actorId) };
    }

    if (request.method === "POST" && request.path === "/posts") {
      if (board.posts.length >= BBS_MAX_POSTS) {
        return { status: 409, body: { error: "board_capacity_reached" } };
      }
      const title = publicInputText(body.title, "title", 100);
      if (title.error) return { status: 400, body: { error: title.error } };
      const postBody = publicInputText(body.body, "body", 4000);
      if (postBody.error) return { status: 400, body: { error: postBody.error } };
      const categoryInput = publicInputText(body.category, "category", 24);
      if (categoryInput.error) return { status: 400, body: { error: categoryInput.error } };
      const category = categoryInput.value.toLowerCase();
      if (!BBS_CATEGORIES.has(category)) {
        return { status: 400, body: { error: "invalid_category" } };
      }
      const actorId = requiredTrustedText(body.actorId, "actorId", 256);
      const post = {
        id: sequenceID("post", board.nextPostId),
        title: title.value,
        body: postBody.value,
        category,
        authorId: actorId,
        authorName: requiredTrustedText(body.actorName, "actorName", 80),
        createdAt: new Date().toISOString(),
        replies: []
      };
      const nextBoard = {
        version: 1,
        nextPostId: board.nextPostId + 1,
        nextReplyId: board.nextReplyId,
        posts: board.posts.concat([post])
      };
      this.state.storage.put("board", nextBoard);
      return { status: 201, body: publicBoard(nextBoard, actorId) };
    }

    const replyMatch = /^\/posts\/(post_[0-9]{12})\/replies$/.exec(request.path);
    if (request.method === "POST" && replyMatch) {
      const postIndex = board.posts.findIndex((candidate) => candidate.id === replyMatch[1]);
      if (postIndex < 0) return { status: 404, body: { error: "post_not_found" } };
      const post = board.posts[postIndex];
      if (post.replies.length >= BBS_MAX_REPLIES) {
        return { status: 409, body: { error: "reply_capacity_reached" } };
      }
      const replyBody = publicInputText(body.body, "body", 2000);
      if (replyBody.error) return { status: 400, body: { error: replyBody.error } };
      const actorId = requiredTrustedText(body.actorId, "actorId", 256);
      const reply = {
        id: sequenceID("reply", board.nextReplyId),
        body: replyBody.value,
        authorId: actorId,
        authorName: requiredTrustedText(body.actorName, "actorName", 80),
        createdAt: new Date().toISOString()
      };
      const nextPosts = board.posts.slice();
      nextPosts[postIndex] = {
        id: post.id,
        title: post.title,
        body: post.body,
        category: post.category,
        authorId: post.authorId,
        authorName: post.authorName,
        createdAt: post.createdAt,
        replies: post.replies.concat([reply])
      };
      const nextBoard = {
        version: 1,
        nextPostId: board.nextPostId,
        nextReplyId: board.nextReplyId + 1,
        posts: nextPosts
      };
      this.state.storage.put("board", nextBoard);
      return { status: 201, body: publicBoard(nextBoard, actorId) };
    }

    const deleteMatch = /^\/posts\/(post_[0-9]{12})$/.exec(request.path);
    if (request.method === "DELETE" && deleteMatch) {
      const index = board.posts.findIndex((candidate) => candidate.id === deleteMatch[1]);
      if (index < 0) return { status: 404, body: { error: "post_not_found" } };
      const actorId = requiredTrustedText(body.actorId, "actorId", 256);
      if (board.posts[index].authorId !== actorId) {
        return { status: 403, body: { error: "not_post_author" } };
      }
      const nextBoard = {
        version: 1,
        nextPostId: board.nextPostId,
        nextReplyId: board.nextReplyId,
        posts: board.posts.filter((_post, postIndex) => postIndex !== index)
      };
      this.state.storage.put("board", nextBoard);
      return { status: 200, body: publicBoard(nextBoard, actorId) };
    }

    return { status: 404, body: { error: "not_found" } };
  }
}

exports.objects = { UserState, BBS };
