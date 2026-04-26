import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  Activity,
  Cable,
  CircleDot,
  Gauge,
  Globe2,
  KeyRound,
  RefreshCw,
  Server,
  Settings,
} from "lucide-react";
import "./styles.css";

const tabs = [
  { id: "connections", label: "实时连接", icon: Cable },
  { id: "egress", label: "出口检测", icon: Globe2 },
  { id: "latency", label: "延迟诊断", icon: Gauge },
  { id: "vps", label: "VPS 监控", icon: Server },
  { id: "settings", label: "设置", icon: Settings },
];

function App() {
  const [active, setActive] = useState("connections");
  const [token, setToken] = useState(localStorage.getItem("netwatch-token") || "");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [data, setData] = useState({
    connections: [],
    egress: [],
    latency: [],
    vps: [],
    errors: [],
  });

  const authHeaders = useMemo(() => {
    return token ? { Authorization: `Bearer ${token}` } : {};
  }, [token]);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [connections, egress, latency, vps, errors] = await Promise.all([
        api("/api/connections?limit=250", authHeaders),
        api("/api/egress", authHeaders),
        api("/api/latency", authHeaders),
        api("/api/vps/nodes", authHeaders),
        api("/api/errors", authHeaders),
      ]);
      setData({ connections, egress, latency, vps, errors });
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const timer = setInterval(load, 5000);
    return () => clearInterval(timer);
  }, [authHeaders]);

  function saveToken(nextToken) {
    setToken(nextToken);
    localStorage.setItem("netwatch-token", nextToken);
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <div className="brand">
          <CircleDot size={22} />
          <div>
            <strong>vps-netwatch</strong>
            <span>VPS / mihomo / Game</span>
          </div>
        </div>
        <nav>
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                className={active === tab.id ? "active" : ""}
                onClick={() => setActive(tab.id)}
                title={tab.label}
              >
                <Icon size={18} />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </nav>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <h1>{tabs.find((tab) => tab.id === active)?.label}</h1>
            <p>{subtitle(active)}</p>
          </div>
          <button className="iconButton" onClick={load} disabled={loading} title="刷新数据">
            <RefreshCw size={18} className={loading ? "spin" : ""} />
          </button>
        </header>

        {error && <div className="banner">API 错误：{error}</div>}
        {!error && data.errors?.length > 0 && (
          <div className="banner muted">
            最近数据源错误：{data.errors.slice(0, 3).map((item) => `${item.source}: ${item.message}`).join("；")}
          </div>
        )}

        {active === "connections" && <Connections rows={data.connections} />}
        {active === "egress" && <Egress rows={data.egress} />}
        {active === "latency" && <Latency rows={data.latency} />}
        {active === "vps" && <VPS rows={data.vps} />}
        {active === "settings" && <SettingsPanel token={token} onToken={saveToken} />}
      </section>
    </main>
  );
}

function Connections({ rows }) {
  const sorted = [...(rows || [])].sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at));
  return (
    <div className="panel">
      <div className="metricRow">
        <Metric label="当前连接" value={sorted.length} />
        <Metric label="控制器" value={new Set(sorted.map((row) => row.controller)).size} />
        <Metric label="目的 IP" value={new Set(sorted.map((row) => row.dest_ip).filter(Boolean)).size} />
      </div>
      <div className="tableWrap">
        <table>
          <thead>
            <tr>
              <th>目标</th>
              <th>协议</th>
              <th>规则</th>
              <th>代理链路</th>
              <th>进程</th>
              <th>流量</th>
              <th>控制器</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((row) => (
              <tr key={`${row.controller}-${row.id}`}>
                <td>
                  <strong>{row.host || row.dest_ip || "unknown"}</strong>
                  <span>{row.dest_ip}:{row.dest_port}</span>
                </td>
                <td>{row.network || "-"}</td>
                <td>{row.rule || "-"}</td>
                <td>{(row.chains || []).join(" → ") || "-"}</td>
                <td>{row.process || "-"}</td>
                <td>{formatBytes((row.upload || 0) + (row.download || 0))}</td>
                <td>{row.controller}</td>
              </tr>
            ))}
            {sorted.length === 0 && <EmptyRow colSpan={7} text="等待 collector 推送 mihomo 连接数据" />}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Egress({ rows }) {
  const results = Array.isArray(rows) ? rows : rows ? [rows] : [];
  return (
    <div className="grid">
      {results.map((item) => (
        <section className="item" key={item.collector_id || item.ip || item.checked_at}>
          <div className="itemTitle">
            <Globe2 size={18} />
            <strong>{item.collector_id || "unknown"}</strong>
          </div>
          <div className="bigValue small">{item.ip || "未上报"}</div>
          <dl>
            <dt>来源</dt>
            <dd>{item.source || "-"}</dd>
            <dt>检查时间</dt>
            <dd>{formatTime(item.checked_at)}</dd>
            <dt>状态</dt>
            <dd>{item.error ? item.error : "正常"}</dd>
          </dl>
        </section>
      ))}
      {results.length === 0 && <div className="panel">等待 agent 或 collector 上报出口 IP</div>}
      <section className="panel">
        <h2>说明</h2>
        <p className="quiet">
          这里按 agent/collector 分别显示公网出口。端游终端不装 agent 时，若流量经过代理 VM 或网关，对应 collector 的出口结果就是排查代理/VPS 出口的主要依据。
        </p>
      </section>
    </div>
  );
}

function Latency({ rows }) {
  const sorted = [...(rows || [])].sort((a, b) => `${a.collector_id}-${a.name}`.localeCompare(`${b.collector_id}-${b.name}`));
  return (
    <div className="grid">
      {sorted.map((row) => (
        <section className="item" key={`${row.collector_id}-${row.name}-${row.host}-${row.port}`}>
          <div className="itemTitle">
            <Activity size={18} />
            <strong>{row.name}</strong>
          </div>
          <div className={row.ok ? "status ok" : "status bad"}>{row.ok ? `${row.rtt_ms.toFixed(1)} ms` : "失败"}</div>
          <p>{row.collector_id || "unknown"}</p>
          <p>{row.host}:{row.port} / {row.protocol}</p>
          {row.error && <p className="errorText">{row.error}</p>}
        </section>
      ))}
      {sorted.length === 0 && <div className="panel">等待延迟探测结果</div>}
    </div>
  );
}

function VPS({ rows }) {
  return (
    <div className="grid">
      {(rows || []).map((node) => (
        <section className="item" key={node.id}>
          <div className="itemTitle">
            <Server size={18} />
            <strong>{node.name || node.id}</strong>
          </div>
          <p>{node.public_ip || "no public ip"}</p>
          <div className="metricLine">
            <span>CPU</span>
            <strong>{percent(node.cpu)}</strong>
          </div>
          <div className="metricLine">
            <span>内存</span>
            <strong>{percent(node.memory?.ratio)}</strong>
          </div>
          <div className="metricLine">
            <span>磁盘</span>
            <strong>{percent(node.disk?.ratio)}</strong>
          </div>
          <div className="metricLine">
            <span>流量</span>
            <strong>↓ {formatBytes(node.net?.rx_bytes || 0)} / ↑ {formatBytes(node.net?.tx_bytes || 0)}</strong>
          </div>
          <div className="metricLine">
            <span>在线</span>
            <strong>{formatDuration(node.uptime_sec || 0)}</strong>
          </div>
          <div className="metricLine">
            <span>sing-box</span>
            <strong>{serviceText(node.services, "sing-box")}</strong>
          </div>
          <small>{formatTime(node.updated_at)}</small>
        </section>
      ))}
      {(!rows || rows.length === 0) && <div className="panel">等待 VPS 节点上报</div>}
    </div>
  );
}

function SettingsPanel({ token, onToken }) {
  return (
    <div className="panel narrow">
      <label className="field">
        <span><KeyRound size={16} /> API Token</span>
        <input
          value={token}
          onChange={(event) => onToken(event.target.value)}
          placeholder="config.yaml 里的 auth.token"
          type="password"
        />
      </label>
      <p className="quiet">
        Token 只保存在浏览器 localStorage。mihomo 由内网 collector 只读读取，不需要暴露给浏览器。
      </p>
    </div>
  );
}

function Metric({ label, value }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function EmptyRow({ colSpan, text }) {
  return (
    <tr>
      <td colSpan={colSpan} className="empty">{text}</td>
    </tr>
  );
}

async function api(path, headers) {
  const response = await fetch(path, { headers });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`${response.status} ${text}`);
  }
  return response.json();
}

function subtitle(tab) {
  const map = {
    connections: "从 mihomo external-controller 只读读取连接，不碰端游终端。",
    egress: "确认代理 VM、网关或 VPS 当前公网出口。",
    latency: "对游戏 IP、VPS 或关键服务做 TCP 延迟采样。",
    vps: "汇总多台 VPS agent 与代理节点状态。",
    settings: "配置浏览器访问 dashboard API 的 token。",
  };
  return map[tab] || "";
}

function formatBytes(value) {
  if (!Number.isFinite(value)) return "-";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let next = value;
  let index = 0;
  while (next >= 1024 && index < units.length - 1) {
    next /= 1024;
    index += 1;
  }
  return `${next.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatTime(value) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function formatDuration(seconds) {
  if (!Number.isFinite(seconds) || seconds <= 0) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分钟`;
  return `${minutes}分钟`;
}

function serviceText(services, name) {
  const service = (services || []).find((item) => item.name === name);
  if (!service) return "-";
  return service.active ? "运行中" : "未运行";
}

function percent(value) {
  if (!Number.isFinite(value)) return "-";
  return `${(value * 100).toFixed(1)}%`;
}

createRoot(document.getElementById("root")).render(<App />);
