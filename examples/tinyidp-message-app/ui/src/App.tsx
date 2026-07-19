import { FormEvent, useState } from "react";

import {
  useCreateAccountMutation,
  useCreateMessageMutation,
  useFeedQuery,
  useLocalLogoutMutation,
  useLogoutMutation,
  useRegistrationQuery,
  useSessionQuery,
} from "./api";

export default function App() {
  const session = useSessionQuery();
  const feed = useFeedQuery();
  const [localLogout] = useLocalLogoutMutation();
  const [logout] = useLogoutMutation();
  const signOutOfDesk = async (csrf: string) => {
    await localLogout({ csrf }).unwrap();
  };
  const signOut = async (csrf: string) => {
    const result = await logout({ csrf }).unwrap();
    if (result.endSessionUrl) location.assign(result.endSessionUrl);
  };

  if (session.isLoading) return <Notice text="Checking desk session…" />;
  if (session.error) return <Notice text="The desk is temporarily unavailable." />;

  const me = session.data!;
  return (
    <main className="desk">
      <header className="mast">
        <p className="kicker">TINY-IDP / LOCAL SERVICE</p>
        <div className="mast-row">
          <div>
            <h1>Message Desk</h1>
            <p className="deck">A small, persistent place for notes left in plain sight.</p>
          </div>
          <div className="status">
            <span>{me.authenticated ? "SIGNED IN" : "GUEST MODE"}</span>
            {me.authenticated ? (
              <>
                <b>{me.displayName}</b>
                <a className="quiet" href="/auth/login?return_to=/&switch_account=1">Change account</a>
                <button className="quiet" onClick={() => signOutOfDesk(me.csrfToken!)}>Log out of Message Desk</button>
                <button className="quiet" onClick={() => signOut(me.csrfToken!)}>Log out everywhere</button>
              </>
            ) : <a className="quiet" href="/auth/login?return_to=/">Sign in</a>}
          </div>
        </div>
      </header>
      <div className="layout">
        <aside>
          <Intro />
          {me.authenticated ? <Composer csrf={me.csrfToken!} /> : <Welcome registrationEnabled={me.registrationEnabled === true} providerRegistrationEnabled={me.providerRegistrationEnabled === true} />}
        </aside>
        <section className="feed">
          <div className="feed-heading">
            <div><p className="kicker accent">CURRENT NOTES</p><h2>From the desk</h2></div>
            <span>{feed.data?.messages.length ?? 0} SHOWN</span>
          </div>
          {feed.isLoading ? <Notice text="Loading notes…" /> : null}
          {feed.error ? <Notice text="Notes could not be loaded." /> : null}
          {feed.data?.messages.length === 0 ? <div className="empty">□<h3>No notes yet.</h3><p>The first message will stay here after a restart.</p></div> : null}
          {feed.data?.messages.map((message) => (
            <article className="note" key={message.id}>
              <div className="note-meta"><b>{message.authorName}</b><time>{new Date(message.createdAt).toLocaleString()}</time></div>
              <p>{message.body}</p>
            </article>
          ))}
        </section>
      </div>
      <footer>MESSAGE DESK <span>IDENTITY BY TINY-IDP</span></footer>
    </main>
  );
}

function Intro() {
  return <section className="intro"><p className="kicker accent">01 / ABOUT</p><h2>Leave a clear record.</h2><p>Messages are stored locally. Sign in to add one; everyone may read the desk.</p></section>;
}

function Welcome({ registrationEnabled, providerRegistrationEnabled }: { registrationEnabled: boolean; providerRegistrationEnabled: boolean }) {
	if (providerRegistrationEnabled) {
		return <section className="composer">
			<p className="kicker">02 / JOIN</p><h2>Open a desk account</h2>
			<p className="form-help">Tiny-IDP creates and protects your account. Message Desk receives only the signed-in identity after the secure authorization flow completes.</p>
			<a className="quiet" href="/auth/register?return_to=/">Create an account with Tiny-IDP</a>
			<a className="quiet" href="/auth/login?return_to=/">I already have an account</a>
		</section>;
	}
	if (!registrationEnabled) {
    return <section className="composer">
      <p className="kicker">02 / SIGN IN</p><h2>Use a desk account</h2>
      <p className="form-help">This standalone demo keeps accounts in tiny-idp. Use one of the operator-seeded identities to continue.</p>
      <a className="quiet" href="/auth/login?return_to=/">Choose an account or sign in</a>
    </section>;
  }
  const registration = useRegistrationQuery();
  const [create, created] = useCreateAccountMutation();
  const [login, setLogin] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!registration.data) return;
    setError(null);
    try {
      await create({ csrf: registration.data.csrfToken, body: { login, displayName: name, password, passwordConfirmation: confirmation } }).unwrap();
      location.assign("/auth/login?return_to=/");
    } catch {
      // The registration pre-session is one-time. Refresh it after failure so
      // a correct retry cannot be mistaken for a stale CSRF submission.
      await registration.refetch();
      setError("Account creation was not accepted. Passwords need at least 15 characters and must not match your login. The login may already be in use. Choose a different combination and try again; this form has been refreshed for a new attempt.");
    }
  };

  return (
    <section className="composer">
      <p className="kicker">02 / JOIN</p><h2>Open an account</h2>
      <form onSubmit={submit}>
        <label>NAME<input value={name} onChange={(event) => setName(event.target.value)} required /></label>
        <label>LOGIN<input value={login} onChange={(event) => setLogin(event.target.value)} required /></label>
        <label>PASSWORD<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} minLength={15} aria-describedby="password-guidance" required /></label>
        <p className="form-help" id="password-guidance">Use 15 or more characters. Do not use your login as the password.</p>
        <label>CONFIRM PASSWORD<input type="password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} minLength={15} required /></label>
        <button disabled={!registration.data || registration.isFetching || created.isLoading}>{created.isLoading ? "Creating…" : registration.isFetching ? "Refreshing form…" : "Create account"}</button>
        {error ? <p className="error" role="alert">{error}</p> : null}
      </form>
      <a className="quiet" href="/auth/login?return_to=/">I already have an account</a>
    </section>
  );
}

function Composer({ csrf }: { csrf: string }) {
  const [body, setBody] = useState("");
  const [create, state] = useCreateMessageMutation();
  const submit = async (event: FormEvent) => { event.preventDefault(); await create({ body, csrf }).unwrap(); setBody(""); };
  return <section className="composer"><p className="kicker">02 / WRITE</p><h2>Leave a note</h2><form onSubmit={submit}><label>MESSAGE<textarea rows={6} value={body} onChange={(event) => setBody(event.target.value)} maxLength={1000} required /></label><div className="form-foot"><span>{body.length} / 1000</span><button disabled={state.isLoading}>{state.isLoading ? "Placing…" : "Place note"}</button></div>{state.error ? <p className="error">Message could not be placed.</p> : null}</form></section>;
}

function Notice({ text }: { text: string }) { return <div className="notice">{text}</div>; }
