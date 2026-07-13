__package__({ name: "tinyidp-xapp" });
__verb__("site", {
  name: "site",
  short: "Register the self-contained identity and private-object application routes",
  output: "text"
});

function site() {
  const express = require("express");
  const assets = require("fs:assets");
  const objects = require("durableobjects");
  const app = express.app();

  const actorDisplayName = (actor) => {
    const claims = actor.claims || {};
    const candidate = claims.name || claims.preferredUsername || "Member";
    const normalized = String(candidate).trim();
    return (normalized || "Member").slice(0, 80);
  };

  const fetchBoard = (ctx, request) => objects.fetch("BBS", "community", {
    method: request.method,
    path: request.path,
    body: {
      ...(request.body || {}),
      actorId: ctx.actor.id,
      actorName: actorDisplayName(ctx.actor)
    }
  });

  app.staticFromAssetsModule("/static", assets, "/app");

  app.get("/")
    .public()
    .handle((_ctx, res) =>
      res.type("text/html").send(assets.readFileSync("/app/index.html", "utf8")));

  app.get("/api/me")
    .auth(express.user().required())
    .allow("user.self.read")
    .audit("user.self.read")
    .handle((ctx, res) => res.json({
      id: ctx.actor.id,
      kind: ctx.actor.kind,
      claims: ctx.actor.claims || {}
    }));

  app.get("/api/object")
    .auth(express.user().required())
    .allow("user.self.read")
    .audit("user.object.read")
    .handle((_ctx, res) => {
      const result = objects.fetchForActor("USER_STATE", {
        method: "GET",
        path: "/state"
      });
      res.status(result.status).json(result.body);
    });

  app.post("/api/object")
    .auth(express.user().required())
    .csrf()
    .allow("user.self.update")
    .audit("user.object.updated")
    .handle((ctx, res) => {
      const result = objects.fetchForActor("USER_STATE", {
        method: "POST",
        path: "/state",
        body: ctx.body || {}
      });
      res.status(result.status).json(result.body);
    });

  app.get("/api/bbs")
    .auth(express.user().required())
    .allow("bbs.read")
    .audit("bbs.read")
    .handle((ctx, res) => {
      const result = fetchBoard(ctx, { method: "GET", path: "/board" });
      res.status(result.status).json(result.body);
    });

  app.post("/api/bbs/posts")
    .auth(express.user().required())
    .csrf()
    .allow("bbs.post.create")
    .audit("bbs.post.created")
    .handle((ctx, res) => {
      const result = fetchBoard(ctx, {
        method: "POST",
        path: "/posts",
        body: {
          title: ctx.body && ctx.body.title,
          body: ctx.body && ctx.body.body,
          category: ctx.body && ctx.body.category
        }
      });
      res.status(result.status).json(result.body);
    });

  app.post("/api/bbs/posts/:postId/replies")
    .auth(express.user().required())
    .csrf()
    .allow("bbs.reply.create")
    .audit("bbs.reply.created")
    .handle((ctx, res) => {
      const result = fetchBoard(ctx, {
        method: "POST",
        path: `/posts/${ctx.params.postId}/replies`,
        body: { body: ctx.body && ctx.body.body }
      });
      res.status(result.status).json(result.body);
    });

  app.delete("/api/bbs/posts/:postId")
    .auth(express.user().required())
    .csrf()
    .allow("bbs.post.delete")
    .audit("bbs.post.deleted")
    .handle((ctx, res) => {
      const result = fetchBoard(ctx, {
        method: "DELETE",
        path: `/posts/${ctx.params.postId}`
      });
      res.status(result.status).json(result.body);
    });
}

module.exports = { site };
