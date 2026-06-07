"use client";

import {
  Activity,
  Building2,
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  LogOut,
  MailPlus,
  Plus,
  RefreshCw,
  Send,
  Trash2,
  Users
} from "lucide-react";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";
import {
  api,
  ApiToken,
  Channel,
  Company,
  CompanyDetail,
  LogEvent,
  User
} from "@/lib/api";

type Tab = "channels" | "logs" | "tokens" | "members";
type AuthMode = "signin" | "signup";

export function DashboardApp() {
  const [user, setUser] = useState<User | null>(null);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [detail, setDetail] = useState<CompanyDetail | null>(null);
  const [selectedCompanyID, setSelectedCompanyID] = useState<string>("");
  const [selectedChannelID, setSelectedChannelID] = useState<string>("");
  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [tab, setTab] = useState<Tab>("channels");
  const [authMode, setAuthMode] = useState<AuthMode>("signin");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [visibleTokenID, setVisibleTokenID] = useState<number | null>(null);

  const selectedChannel = detail?.channels.find((channel) => channel.id === selectedChannelID);

  const loadCompanies = useCallback(async (preferredID?: string) => {
    const list = await api<Company[]>("/api/companies");
    setCompanies(list);
    const nextID = preferredID || selectedCompanyID || list[0]?.id || "";
    setSelectedCompanyID(nextID);
    if (nextID) {
      const company = await api<CompanyDetail>(`/api/companies/${nextID}`);
      setDetail(company);
      setSelectedChannelID((current) => current || company.channels[0]?.id || "");
    } else {
      setDetail(null);
      setSelectedChannelID("");
    }
  }, [selectedCompanyID]);

  const refreshLogs = useCallback(async () => {
    if (!selectedChannelID) {
      setLogs([]);
      return;
    }
    const data = await api<LogEvent[]>(`/api/logs/${selectedChannelID}?limit=200`);
    setLogs(data);
  }, [selectedChannelID]);

  useEffect(() => {
    api<User>("/api/auth/me")
      .then((currentUser) => {
        setUser(currentUser);
        return loadCompanies();
      })
      .catch(() => {
        setUser(null);
      });
  }, [loadCompanies]);

  useEffect(() => {
    if (!selectedChannelID || !user) return;
    const first = window.setTimeout(() => {
      refreshLogs().catch((err: Error) => setError(err.message));
    }, 0);
    const id = window.setInterval(() => {
      refreshLogs().catch(() => {});
    }, 5000);
    return () => {
      window.clearTimeout(first);
      window.clearInterval(id);
    };
  }, [refreshLogs, selectedChannelID, user]);

  const eventChart = useMemo(() => {
    const counts = new Map<string, number>();
    for (const log of logs) {
      counts.set(log.event_name, (counts.get(log.event_name) ?? 0) + 1);
    }
    return [...counts.entries()]
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 8);
  }, [logs]);

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  }

  async function submitAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    await run(async () => {
      const currentUser = await api<{ user: User }>(`/api/auth/${authMode}`, {
        method: "POST",
        body: JSON.stringify({
          name: String(form.get("name") ?? ""),
          email: String(form.get("email") ?? ""),
          password: String(form.get("password") ?? "")
        })
      });
      setUser(currentUser.user);
      await loadCompanies();
    });
  }

  async function signOut() {
    await run(async () => {
      await api("/api/auth/signout", { method: "POST", body: "{}" });
      setUser(null);
      setCompanies([]);
      setDetail(null);
      setLogs([]);
    });
  }

  async function createCompany(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    await run(async () => {
      const company = await api<Company>("/api/companies", {
        method: "POST",
        body: JSON.stringify({ name: form.get("name") })
      });
      event.currentTarget.reset();
      await loadCompanies(company.id);
    });
  }

  async function renameCompany(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail) return;
    const form = new FormData(event.currentTarget);
    await run(async () => {
      await api<Company>(`/api/companies/${detail.company.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name: form.get("name") })
      });
      await loadCompanies(detail.company.id);
    });
  }

  async function createChannel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail) return;
    const form = new FormData(event.currentTarget);
    await run(async () => {
      const channel = await api<Channel>(`/api/companies/${detail.company.id}/channels`, {
        method: "POST",
        body: JSON.stringify({
          name: form.get("name"),
          icon: form.get("icon")
        })
      });
      event.currentTarget.reset();
      setSelectedChannelID(channel.id);
      await loadCompanies(detail.company.id);
    });
  }

  async function createToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail) return;
    const form = new FormData(event.currentTarget);
    await run(async () => {
      await api<ApiToken>(`/api/companies/${detail.company.id}/tokens`, {
        method: "POST",
        body: JSON.stringify({ name: form.get("name") })
      });
      event.currentTarget.reset();
      await loadCompanies(detail.company.id);
    });
  }

  async function deleteToken(token: ApiToken) {
    if (!detail) return;
    await run(async () => {
      await api(`/api/companies/${detail.company.id}/tokens/${token.id}`, {
        method: "DELETE"
      });
      await loadCompanies(detail.company.id);
    });
  }

  async function createInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail) return;
    const form = new FormData(event.currentTarget);
    await run(async () => {
      await api(`/api/companies/${detail.company.id}/invites`, {
        method: "POST",
        body: JSON.stringify({ email: form.get("email") })
      });
      event.currentTarget.reset();
      await loadCompanies(detail.company.id);
    });
  }

  async function deleteInvite(inviteID: string) {
    if (!detail) return;
    await run(async () => {
      await api(`/api/companies/${detail.company.id}/invites/${inviteID}`, {
        method: "DELETE"
      });
      await loadCompanies(detail.company.id);
    });
  }

  async function sendTestEvent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail || !selectedChannelID || detail.tokens.length === 0) return;
    const form = new FormData(event.currentTarget);
    await run(async () => {
      await api("/api/logs", {
        method: "POST",
        headers: { Authorization: `Bearer ${detail.tokens[0].token}` },
        body: JSON.stringify({
          channel_id: selectedChannelID,
          event_name: form.get("event_name"),
          event_payload: form.get("event_payload"),
          metadata: { source: "dashboard" }
        })
      });
      event.currentTarget.reset();
      await refreshLogs();
    });
  }

  if (!user) {
    return (
      <main className="auth-shell">
        <section className="auth-panel">
          <div>
            <p className="eyebrow">Raterlog</p>
            <h1>Realtime product monitoring</h1>
            <p className="muted">
              Sign in to manage companies, channels, team invites, tokens, and live event logs.
            </p>
          </div>
          <div className="segmented" role="tablist" aria-label="Authentication mode">
            <button className={authMode === "signin" ? "active" : ""} onClick={() => setAuthMode("signin")}>
              Sign in
            </button>
            <button className={authMode === "signup" ? "active" : ""} onClick={() => setAuthMode("signup")}>
              Sign up
            </button>
          </div>
          <form className="stack" onSubmit={submitAuth}>
            {authMode === "signup" && (
              <label>
                Name
                <input name="name" minLength={3} required placeholder="Your name" />
              </label>
            )}
            <label>
              Email
              <input name="email" type="email" required placeholder="you@example.com" />
            </label>
            <label>
              Password
              <input name="password" type="password" minLength={6} required placeholder="Min 6 characters" />
            </label>
            {error && <p className="error">{error}</p>}
            <button className="primary" disabled={busy}>
              {authMode === "signin" ? "Sign in" : "Create account"}
            </button>
          </form>
        </section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <Activity size={22} />
          <span>Raterlog</span>
        </div>
        <form className="compact-form" onSubmit={createCompany}>
          <input name="name" placeholder="New company" required />
          <button aria-label="Create company" disabled={busy}>
            <Plus size={16} />
          </button>
        </form>
        <nav className="company-list">
          {companies.map((company) => (
            <button
              key={company.id}
              className={company.id === selectedCompanyID ? "active" : ""}
              onClick={() => run(() => loadCompanies(company.id))}
            >
              <Building2 size={16} />
              <span>{company.name}</span>
            </button>
          ))}
        </nav>
        <div className="account">
          <span>{user.name}</span>
          <button aria-label="Sign out" onClick={signOut}>
            <LogOut size={16} />
          </button>
        </div>
      </aside>

      <section className="workspace">
        {detail ? (
          <>
            <header className="topbar">
              <div>
                <p className="eyebrow">Company</p>
                <h1>{detail.company.name}</h1>
              </div>
              <form className="rename-form" onSubmit={renameCompany}>
                <input name="name" defaultValue={detail.company.name} aria-label="Company name" />
                <button disabled={busy}>Rename</button>
              </form>
            </header>

            {error && <p className="error inline">{error}</p>}

            <div className="tabs">
              <TabButton tab="channels" active={tab} setTab={setTab} icon={<Building2 size={16} />} />
              <TabButton tab="logs" active={tab} setTab={setTab} icon={<Activity size={16} />} />
              <TabButton tab="tokens" active={tab} setTab={setTab} icon={<KeyRound size={16} />} />
              <TabButton tab="members" active={tab} setTab={setTab} icon={<Users size={16} />} />
            </div>

            {tab === "channels" && (
              <section className="grid two">
                <div className="panel">
                  <h2>Channels</h2>
                  <div className="list">
                    {detail.channels.map((channel) => (
                      <button
                        key={channel.id}
                        className={`list-row ${channel.id === selectedChannelID ? "active" : ""}`}
                        onClick={() => {
                          setSelectedChannelID(channel.id);
                          setTab("logs");
                        }}
                      >
                        <span className="icon-chip">{channel.icon}</span>
                        <span>
                          <strong>{channel.name}</strong>
                          <small>{channel.id}</small>
                        </span>
                      </button>
                    ))}
                    {detail.channels.length === 0 && <p className="muted">No channels yet.</p>}
                  </div>
                </div>
                <form className="panel stack" onSubmit={createChannel}>
                  <h2>Create Channel</h2>
                  <label>
                    Name
                    <input name="name" required placeholder="signups" />
                  </label>
                  <label>
                    Icon
                    <input name="icon" required maxLength={4} placeholder="S" />
                  </label>
                  <button className="primary" disabled={busy}>Create channel</button>
                </form>
              </section>
            )}

            {tab === "logs" && (
              <section className="grid logs-layout">
                <div className="panel">
                  <div className="panel-header">
                    <div>
                      <h2>{selectedChannel ? selectedChannel.name : "Logs"}</h2>
                      <p className="muted">{selectedChannelID || "Create a channel to view events."}</p>
                    </div>
                    <button className="icon-button" onClick={() => run(refreshLogs)} aria-label="Refresh logs">
                      <RefreshCw size={16} />
                    </button>
                  </div>
                  <div className="stats">
                    <Metric label="Events" value={logs.length} />
                    <Metric label="Types" value={eventChart.length} />
                  </div>
                  <div className="chart">
                    {eventChart.length > 0 ? (
                      <ResponsiveContainer width="100%" height={220}>
                        <BarChart data={eventChart}>
                          <CartesianGrid strokeDasharray="3 3" stroke="#263245" />
                          <XAxis dataKey="name" stroke="#8793a5" tick={{ fontSize: 12 }} />
                          <YAxis stroke="#8793a5" tick={{ fontSize: 12 }} allowDecimals={false} />
                          <Tooltip contentStyle={{ background: "#111827", border: "1px solid #263245" }} />
                          <Bar dataKey="count" fill="#38bdf8" radius={[4, 4, 0, 0]} />
                        </BarChart>
                      </ResponsiveContainer>
                    ) : (
                      <div className="empty-chart">No events yet</div>
                    )}
                  </div>
                  <div className="event-list">
                    {logs.map((log) => (
                      <article key={`${log.channel_id}-${log.timestamp}`} className="event-row">
                        <div>
                          <strong>{log.event_name}</strong>
                          <span>{log.event_payload}</span>
                          {log.metadata && <code>{log.metadata}</code>}
                        </div>
                        <time>{new Date(log.timestamp).toLocaleString()}</time>
                      </article>
                    ))}
                  </div>
                </div>
                <form className="panel stack" onSubmit={sendTestEvent}>
                  <h2>Send Test Event</h2>
                  <label>
                    Channel
                    <select value={selectedChannelID} onChange={(event) => setSelectedChannelID(event.target.value)}>
                      {detail.channels.map((channel) => (
                        <option key={channel.id} value={channel.id}>{channel.name}</option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Event name
                    <input name="event_name" required placeholder="signup.created" />
                  </label>
                  <label>
                    Payload
                    <input name="event_payload" required placeholder="Created from dashboard" />
                  </label>
                  <button className="primary with-icon" disabled={busy || detail.tokens.length === 0 || !selectedChannelID}>
                    <Send size={16} /> Send event
                  </button>
                  {detail.tokens.length === 0 && <p className="muted">Create an API token before sending events.</p>}
                </form>
              </section>
            )}

            {tab === "tokens" && (
              <section className="grid two">
                <div className="panel">
                  <h2>API Tokens</h2>
                  <div className="list">
                    {detail.tokens.map((token) => (
                      <div key={token.id} className="token-row">
                        <div>
                          <strong>{token.name}</strong>
                          <code>{visibleTokenID === token.id ? token.token : "x".repeat(24)}</code>
                        </div>
                        <button onClick={() => setVisibleTokenID(visibleTokenID === token.id ? null : token.id)} aria-label="Toggle token">
                          {visibleTokenID === token.id ? <EyeOff size={16} /> : <Eye size={16} />}
                        </button>
                        <button onClick={() => navigator.clipboard.writeText(token.token)} aria-label="Copy token">
                          <Copy size={16} />
                        </button>
                        <button onClick={() => deleteToken(token)} aria-label="Delete token">
                          <Trash2 size={16} />
                        </button>
                      </div>
                    ))}
                    {detail.tokens.length === 0 && <p className="muted">No tokens yet.</p>}
                  </div>
                </div>
                <form className="panel stack" onSubmit={createToken}>
                  <h2>Create Token</h2>
                  <label>
                    Name
                    <input name="name" required placeholder="production-server" />
                  </label>
                  <button className="primary" disabled={busy}>Generate token</button>
                </form>
              </section>
            )}

            {tab === "members" && (
              <section className="grid two">
                <div className="panel">
                  <h2>Members</h2>
                  <div className="table">
                    {[...detail.members.map((member) => ({ id: member.user_id, name: member.name, email: member.email, status: member.role })),
                      ...detail.invites.filter((invite) => invite.status === "pending").map((invite) => ({ id: invite.id, name: "-", email: invite.email, status: "pending" }))]
                      .map((row) => (
                        <div className="table-row" key={row.id}>
                          <span>{row.name}</span>
                          <span>{row.email}</span>
                          <span className="pill">{row.status}</span>
                          {row.status === "pending" && (
                            <button onClick={() => deleteInvite(row.id)} aria-label="Revoke invite">
                              <Trash2 size={16} />
                            </button>
                          )}
                        </div>
                      ))}
                  </div>
                </div>
                <form className="panel stack" onSubmit={createInvite}>
                  <h2>Invite Member</h2>
                  <label>
                    Email
                    <input name="email" type="email" required placeholder="teammate@example.com" />
                  </label>
                  <button className="primary with-icon" disabled={busy}>
                    <MailPlus size={16} /> Send invite
                  </button>
                </form>
              </section>
            )}
          </>
        ) : (
          <div className="empty-state">
            <Building2 size={32} />
            <h1>Create a company to start logging events</h1>
          </div>
        )}
      </section>
    </main>
  );
}

function TabButton({ tab, active, setTab, icon }: {
  tab: Tab;
  active: Tab;
  setTab: (tab: Tab) => void;
  icon: React.ReactNode;
}) {
  return (
    <button className={active === tab ? "active" : ""} onClick={() => setTab(tab)}>
      {icon}
      <span>{tab[0].toUpperCase() + tab.slice(1)}</span>
    </button>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
