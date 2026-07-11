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
}

module.exports = { site };
