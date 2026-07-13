import { useEffect, useState, type FormEvent } from "react";
import {
  apiErrorMessage,
  bbsApi,
  isUnauthorized,
  type BBSCategory,
  type BBSPost,
  useCreatePostMutation,
  useCreateReplyMutation,
  useDeletePostMutation,
  useGetBoardQuery,
  useGetMeQuery,
  useGetSessionQuery,
  useLogoutMutation
} from "./api";
import { sessionEnded, sessionEstablished } from "./authSlice";
import { useAppDispatch, useAppSelector } from "./store";

const categories: Array<{ value: BBSCategory; label: string }> = [
  { value: "general", label: "General" },
  { value: "projects", label: "Projects" },
  { value: "questions", label: "Questions" },
  { value: "notes", label: "Notes" }
];

function App() {
  const dispatch = useAppDispatch();
  const auth = useAppSelector((state) => state.auth);
  const session = useGetSessionQuery(undefined, { skip: auth.status === "loggedOut" });
  const sessionReady = Boolean(session.data && auth.csrfToken);
  const me = useGetMeQuery(undefined, { skip: !sessionReady });
  const board = useGetBoardQuery(undefined, { skip: !sessionReady });
  const [logout, logoutState] = useLogoutMutation();

  useEffect(() => {
    if (session.data) {
      dispatch(sessionEstablished({ userId: session.data.userId, csrfToken: session.data.csrfToken }));
    }
  }, [dispatch, session.data]);

  const handleLogout = async () => {
    try {
      await logout().unwrap();
      dispatch(sessionEnded());
      dispatch(bbsApi.util.resetApiState());
    } catch {
      // The visible mutation state below carries the server error.
    }
  };

  if (auth.status === "loggedOut") {
    return <SessionEnded />;
  }

  if (session.isLoading || (session.data && !sessionReady)) {
    return <CenteredNotice label="Checking your application session…" />;
  }

  if (isUnauthorized(session.error)) {
    return <SignInRequired />;
  }

  if (session.error) {
    return <CenteredNotice label={apiErrorMessage(session.error)} tone="error" />;
  }

  const displayName = userDisplayName(me.data, auth.userId);

  return (
    <div className="page-shell">
      <header className="masthead">
        <div>
          <p className="eyebrow accent-blue mb-1">TINY-IDP COMMUNITY SERVICE</p>
          <h1>Local Loop</h1>
          <p className="masthead-copy mb-0">Persistent notes, questions, and project dispatches.</p>
        </div>
        <div className="session-block">
          <span className="session-label">SIGNED IN</span>
          <strong data-testid="current-user">{displayName}</strong>
          <button className="text-action accent-coral" type="button" onClick={handleLogout} disabled={logoutState.isLoading}>
            {logoutState.isLoading ? "Ending session…" : "Log out"}
          </button>
          {logoutState.error ? <span className="inline-error">{apiErrorMessage(logoutState.error)}</span> : null}
        </div>
      </header>

      <main className="container-fluid px-0">
        <div className="row g-4">
          <aside className="col-12 col-lg-4">
            <section className="intro-block" aria-labelledby="board-introduction">
              <p className="section-number">01 / BOARD</p>
              <h2 id="board-introduction">A compact shared record.</h2>
              <p>{board.data?.description || "Read and write through one serialized durable object."}</p>
              <dl className="board-stats">
                <div><dt>POSTS</dt><dd data-testid="post-count">{board.data?.stats.posts ?? "—"}</dd></div>
                <div><dt>REPLIES</dt><dd data-testid="reply-count">{board.data?.stats.replies ?? "—"}</dd></div>
              </dl>
            </section>
            <PostComposer disabled={!board.data || board.isFetching} />
          </aside>

          <section className="col-12 col-lg-8" aria-labelledby="threads-heading">
            <div className="thread-heading">
              <div>
                <p className="section-number">02 / CURRENT THREADS</p>
                <h2 id="threads-heading">Dispatches</h2>
              </div>
              <button className="text-action" type="button" onClick={() => board.refetch()} disabled={board.isFetching}>
                {board.isFetching ? "Refreshing…" : "Refresh"}
              </button>
            </div>
            {board.isLoading ? <Notice label="Loading the board…" /> : null}
            {board.error ? <Notice label={apiErrorMessage(board.error)} tone="error" /> : null}
            {board.data && board.data.posts.length === 0 ? <EmptyBoard /> : null}
            {board.data?.posts.map((post) => <Thread key={post.id} post={post} />)}
          </section>
        </div>
      </main>

      <footer>
        <span>LOCAL LOOP / ONE SHARED DURABLE OBJECT</span>
        <span className="accent-teal">IDENTITY VERIFIED BY TINY-IDP</span>
      </footer>
    </div>
  );
}

function PostComposer({ disabled }: { disabled: boolean }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [category, setCategory] = useState<BBSCategory>("general");
  const [createPost, mutation] = useCreatePostMutation();

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    try {
      await createPost({ title, body, category }).unwrap();
      setTitle("");
      setBody("");
      setCategory("general");
    } catch {
      // RTK Query exposes the visible error state.
    }
  };

  return (
    <section className="composer-block" aria-labelledby="new-post-heading">
      <p className="section-number accent-teal">NEW DISPATCH</p>
      <h2 id="new-post-heading">Post to the loop</h2>
      <form onSubmit={submit}>
        <label className="form-label" htmlFor="post-title">Title</label>
        <input id="post-title" className="form-control" value={title} onChange={(event) => setTitle(event.target.value)} maxLength={100} required disabled={disabled || mutation.isLoading} />

        <label className="form-label" htmlFor="post-category">Category</label>
        <select id="post-category" className="form-select" value={category} onChange={(event) => setCategory(event.target.value as BBSCategory)} disabled={disabled || mutation.isLoading}>
          {categories.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
        </select>

        <label className="form-label" htmlFor="post-body">Message</label>
        <textarea id="post-body" className="form-control" rows={7} value={body} onChange={(event) => setBody(event.target.value)} maxLength={4000} required disabled={disabled || mutation.isLoading} />
        <div className="form-foot">
          <span>{body.length} / 4000</span>
          <button className="primary-action" type="submit" disabled={disabled || mutation.isLoading}>
            {mutation.isLoading ? "Posting…" : "Post dispatch"}
          </button>
        </div>
        {mutation.error ? <p className="form-error" role="alert">{apiErrorMessage(mutation.error)}</p> : null}
      </form>
    </section>
  );
}

function Thread({ post }: { post: BBSPost }) {
  const [deletePost, deletion] = useDeletePostMutation();

  const remove = async () => {
    try {
      await deletePost(post.id).unwrap();
    } catch {
      // RTK Query exposes the visible error state.
    }
  };

  return (
    <article className="thread" data-testid={`thread-${post.id}`}>
      <div className="thread-meta">
        <span className={`category category-${post.category}`}>{post.category.toUpperCase()}</span>
        <span>BY <strong>{post.author}</strong></span>
        <time dateTime={post.createdAt}>{formatTime(post.createdAt)}</time>
      </div>
      <h3>{post.title}</h3>
      <p className="post-body">{post.body}</p>
      {post.canDelete ? (
        <div className="owner-actions">
          <button className="text-action accent-coral" type="button" onClick={remove} disabled={deletion.isLoading}>
            {deletion.isLoading ? "Deleting thread…" : "Delete thread and replies"}
          </button>
          {deletion.error ? <span className="inline-error">{apiErrorMessage(deletion.error)}</span> : null}
        </div>
      ) : null}
      <div className="replies" aria-label={`Replies to ${post.title}`}>
        <p className="reply-count">{post.replies.length === 1 ? "1 REPLY" : `${post.replies.length} REPLIES`}</p>
        {post.replies.map((reply) => (
          <div className="reply" key={reply.id} data-testid={`reply-${reply.id}`}>
            <p>{reply.body}</p>
            <span>— {reply.author}, <time dateTime={reply.createdAt}>{formatTime(reply.createdAt)}</time></span>
          </div>
        ))}
        <ReplyComposer postId={post.id} />
      </div>
    </article>
  );
}

function ReplyComposer({ postId }: { postId: string }) {
  const [body, setBody] = useState("");
  const [createReply, mutation] = useCreateReplyMutation();

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    try {
      await createReply({ postId, body }).unwrap();
      setBody("");
    } catch {
      // RTK Query exposes the visible error state.
    }
  };

  return (
    <form className="reply-form" onSubmit={submit}>
      <label htmlFor={`reply-${postId}`}>Add a reply</label>
      <div className="reply-form-row">
        <textarea id={`reply-${postId}`} className="form-control" rows={2} value={body} onChange={(event) => setBody(event.target.value)} maxLength={2000} required disabled={mutation.isLoading} />
        <button className="secondary-action" type="submit" disabled={mutation.isLoading}>{mutation.isLoading ? "Sending…" : "Reply"}</button>
      </div>
      {mutation.error ? <p className="form-error" role="alert">{apiErrorMessage(mutation.error)}</p> : null}
    </form>
  );
}

function EmptyBoard() {
  return (
    <div className="empty-board" data-testid="empty-board">
      <span className="empty-mark">□</span>
      <h3>No dispatches yet.</h3>
      <p>Write the first post. It will remain here after the application restarts.</p>
    </div>
  );
}

function SignInRequired() {
  return (
    <div className="centered-notice">
      <p className="eyebrow accent-blue">LOCAL LOOP</p>
      <h1>Application session required.</h1>
      <p>Sign in through tiny-idp to read and write the shared board.</p>
      <a className="primary-action d-inline-block" href="/auth/login?return_to=/">Sign in</a>
    </div>
  );
}

function SessionEnded() {
  return (
    <div className="centered-notice" data-testid="session-ended">
      <p className="eyebrow accent-teal">SESSION ENDED</p>
      <h1>You are logged out of the application.</h1>
      <p>Your tiny-idp identity-provider session may still be active. Starting again will create a new application session.</p>
      <a className="primary-action d-inline-block" href="/auth/login?return_to=/">Sign in again</a>
    </div>
  );
}

function CenteredNotice({ label, tone = "normal" }: { label: string; tone?: "normal" | "error" }) {
  return <div className={`centered-notice ${tone === "error" ? "notice-error" : ""}`} role="status">{label}</div>;
}

function Notice({ label, tone = "normal" }: { label: string; tone?: "normal" | "error" }) {
  return <p className={`notice ${tone === "error" ? "notice-error" : ""}`} role="status">{label}</p>;
}

function userDisplayName(user: { claims: Record<string, unknown> } | undefined, fallback: string): string {
  const value = user?.claims.name || user?.claims.preferredUsername;
  return typeof value === "string" && value.trim() ? value.trim() : fallback;
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}

export default App;
