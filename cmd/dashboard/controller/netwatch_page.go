package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/service/singleton"
)

func netwatchShouldInjectUserIndex(filePath string) bool {
	return filePath == singleton.Conf.UserTemplate+"/index.html"
}

func netwatchServeInjectedUserIndex(c *gin.Context, statusCode int, content []byte) {
	html := string(content)
	if !strings.Contains(html, "vps-netwatch-home-button") {
		if strings.Contains(html, "</body>") {
			html = strings.Replace(html, "</body>", netwatchHomeButtonScript+"</body>", 1)
		} else {
			html += netwatchHomeButtonScript
		}
	}
	c.Data(statusCode, "text/html; charset=utf-8", []byte(html))
}

const netwatchHomeButtonScript = `<script id="vps-netwatch-home-button">
(function () {
  var marker = "data-vps-netwatch-latency";
  var buttonClass = "inset-shadow-2xs inset-shadow-white/20 flex cursor-pointer flex-col items-center gap-0 rounded-[50px] bg-blue-100 p-2.5 text-blue-600 transition-all dark:bg-blue-900 dark:text-blue-100";
  var activeClass = "inset-shadow-black/20 bg-blue-600 text-white ring-2 ring-blue-300/70 dark:bg-blue-600 dark:text-white dark:ring-blue-300/70";
  var colors = ["#5276d8", "#f2c14e", "#73bf69", "#e4576b", "#69bde7", "#8b5cf6", "#14b8a6", "#f97316", "#ec4899", "#64748b"];
  var state = { visible: false, loaded: false, loading: false, data: null, domain: null, view: null, hover: null, date: "", protocol: "ICMP", showExtremes: false, showAverage: false, selectedServerId: "", peerTargetServerId: "", peerSaving: false, refreshTimer: 0, peerTimer: 0, peerWaitUntil: 0 };
  var panel;
  var canvas;
  var ctx;
  var tooltip;
  var lastPlot;

  function injectStyle() {
    if (document.getElementById("vps-netwatch-inline-style")) return;
    var style = document.createElement("style");
    style.id = "vps-netwatch-inline-style";
    style.textContent =
      "#vps-netwatch-latency-panel{margin-top:14px;border:1px solid rgba(148,163,184,.26);border-radius:16px;background:rgba(255,255,255,.86);box-shadow:0 10px 28px rgba(15,23,42,.08);overflow:hidden;color:#111827}" +
      ".dark #vps-netwatch-latency-panel{background:rgba(0,0,0,.48);border-color:rgba(255,255,255,.12);color:#f8fafc}" +
      "#vps-netwatch-latency-panel[hidden]{display:none!important}" +
      ".vpsnw-head{display:grid;grid-template-columns:max-content minmax(0,1fr);align-items:center;gap:8px 10px;padding:12px 16px;border-bottom:1px solid rgba(148,163,184,.22)}" +
      ".vpsnw-title{grid-column:1;grid-row:1;font-weight:800;font-size:14px;white-space:nowrap}" +
      ".vpsnw-server-tabs{grid-column:2;grid-row:1;display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-server-btn{border:1px solid rgba(148,163,184,.35);border-radius:999px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 10px;white-space:nowrap}.vpsnw-server-btn.active{border-color:#2563eb;background:#2563eb;color:#fff}.dark .vpsnw-server-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-server-btn.active{background:#dbeafe;border-color:#dbeafe;color:#1d4ed8}" +
      ".vpsnw-peer-row{grid-column:2;grid-row:2;display:flex;flex-wrap:wrap;align-items:center;justify-content:flex-start;gap:6px;min-width:0;font-size:12px;color:#64748b}.dark .vpsnw-peer-row{color:#94a3b8}.vpsnw-peer-label{grid-column:1;grid-row:2;font-size:12px;white-space:nowrap;color:#64748b}.dark .vpsnw-peer-label{color:#94a3b8}.vpsnw-peer-tabs{display:flex;flex-wrap:wrap;justify-content:flex-start;gap:6px;min-width:0}.vpsnw-peer-note{min-width:48px;color:#2563eb}.dark .vpsnw-peer-note{color:#93c5fd}.vpsnw-peer-btn{border:1px solid rgba(148,163,184,.35);border-radius:7px;background:rgba(248,250,252,.9);color:#334155;cursor:pointer;font-size:12px;line-height:1;padding:5px 9px;white-space:nowrap}.vpsnw-peer-btn.active{border-color:#0f172a;background:#0f172a;color:#fff}.vpsnw-peer-btn:disabled{cursor:not-allowed;opacity:.58}.dark .vpsnw-peer-btn{background:rgba(255,255,255,.08);border-color:rgba(255,255,255,.16);color:#e2e8f0}.dark .vpsnw-peer-btn.active{background:#f8fafc;border-color:#f8fafc;color:#0f172a}" +
      ".vpsnw-tools{display:flex;flex-wrap:wrap;align-items:center;gap:8px;padding:8px 16px;border-bottom:1px solid rgba(148,163,184,.18)}.vpsnw-tool,.vpsnw-date,.vpsnw-protocol{display:inline-flex;align-items:center;gap:7px;min-height:30px;border:1px solid rgba(148,163,184,.32);border-radius:7px;background:rgba(248,250,252,.88);color:#334155;font-size:12px;line-height:1;padding:4px 8px}.dark .vpsnw-tool,.dark .vpsnw-date,.dark .vpsnw-protocol{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#e2e8f0}.vpsnw-date input{border:0;outline:0;background:transparent;color:inherit;font:inherit;font-weight:700;min-width:116px}.vpsnw-icon-btn{display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border:1px solid rgba(148,163,184,.32);border-radius:7px;background:rgba(248,250,252,.88);color:#334155;cursor:pointer}.dark .vpsnw-icon-btn{background:rgba(255,255,255,.07);border-color:rgba(255,255,255,.14);color:#e2e8f0}.vpsnw-icon-btn svg{width:15px;height:15px}.vpsnw-chip{border:1px solid rgba(148,163,184,.32);border-radius:6px;background:transparent;color:inherit;cursor:pointer;font:inherit;line-height:1;padding:5px 8px}.vpsnw-chip.active{border-color:#2563eb;background:#2563eb;color:#fff}.dark .vpsnw-chip.active{border-color:#dbeafe;background:#dbeafe;color:#1d4ed8}.vpsnw-muted{color:#64748b}.dark .vpsnw-muted{color:#94a3b8}" +
      ".vpsnw-legend{display:flex;flex-wrap:wrap;justify-content:center;gap:10px 18px;padding:10px 14px 2px;font-size:13px}.vpsnw-legend span{display:inline-flex;align-items:center;gap:6px}.vpsnw-dot{width:10px;height:10px;border-radius:50%;display:inline-block}" +
      ".vpsnw-chart{position:relative;height:360px;padding:6px 14px 14px}.vpsnw-chart canvas{width:100%;height:100%;display:block}" +
      ".vpsnw-tip{display:none;position:absolute;z-index:20;min-width:160px;max-width:260px;padding:10px 12px;border:1px solid rgba(148,163,184,.35);border-radius:8px;background:rgba(255,255,255,.96);box-shadow:0 12px 28px rgba(15,23,42,.2);font-size:13px;color:#111827;pointer-events:none}.dark .vpsnw-tip{background:rgba(15,15,15,.96);color:#f8fafc}" +
      ".vpsnw-tip-time{color:#475569;margin-bottom:6px}.dark .vpsnw-tip-time{color:#cbd5e1}.vpsnw-tip-row{display:flex;align-items:center;justify-content:space-between;gap:14px;line-height:1.7}.vpsnw-tip-name{display:flex;align-items:center;gap:6px}" +
      ".vpsnw-empty{display:none;padding:22px 16px;color:#94a3b8;text-align:center;font-size:13px}" +
      "@media(max-width:760px){.vpsnw-chart{height:300px;padding-left:8px;padding-right:8px}.vpsnw-head{grid-template-columns:1fr}.vpsnw-title,.vpsnw-server-tabs,.vpsnw-peer-label,.vpsnw-peer-row{grid-column:1;grid-row:auto}.vpsnw-tools{padding:8px}.vpsnw-date{flex:1}.vpsnw-date input{min-width:0;width:100%}}";
    document.head.appendChild(style);
  }

  function getToken() {
    function scan(value, depth) {
      if (!value || depth > 3) return "";
      if (typeof value === "string") {
        var text = value.trim();
        if (/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(text)) return text;
        if ((text.charAt(0) === "{" || text.charAt(0) === "[") && text.length < 20000) {
          try { return scan(JSON.parse(text), depth + 1); } catch (_) { return ""; }
        }
        return "";
      }
      if (Array.isArray(value)) {
        for (var i = 0; i < value.length; i++) { var a = scan(value[i], depth + 1); if (a) return a; }
      } else if (typeof value === "object") {
        for (var key in value) { var b = scan(value[key], depth + 1); if (b) return b; }
      }
      return "";
    }
    var stores = [window.localStorage, window.sessionStorage];
    for (var s = 0; s < stores.length; s++) {
      try {
        for (var i = 0; i < stores[s].length; i++) {
          var token = scan(stores[s].getItem(stores[s].key(i)), 0);
          if (token) return token;
        }
      } catch (_) {}
    }
    return "";
  }

  function createPanel(host) {
    injectStyle();
    if (!panel) {
      panel = document.createElement("section");
      panel.id = "vps-netwatch-latency-panel";
      panel.hidden = true;
      panel.innerHTML =
        '<div class="vpsnw-head"><div class="vpsnw-title">延迟</div><div class="vpsnw-server-tabs" id="vpsnw-server-tabs"></div><span class="vpsnw-peer-label">互 ping 目标</span><div class="vpsnw-peer-row"><div class="vpsnw-peer-tabs" id="vpsnw-peer-tabs"></div><span class="vpsnw-peer-note" id="vpsnw-peer-note"></span></div></div>' +
        '<div class="vpsnw-tools"><button class="vpsnw-icon-btn" id="vpsnw-prev-day" title="上一天"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M12.8 4.2a1 1 0 0 1 0 1.4L8.4 10l4.4 4.4a1 1 0 1 1-1.4 1.4l-5.1-5.1a1 1 0 0 1 0-1.4l5.1-5.1a1 1 0 0 1 1.4 0Z"/></svg></button><label class="vpsnw-date"><span class="vpsnw-muted">日期</span><input id="vpsnw-date" type="date"></label><button class="vpsnw-icon-btn" id="vpsnw-next-day" title="下一天"><svg viewBox="0 0 20 20" fill="currentColor"><path d="M7.2 15.8a1 1 0 0 1 0-1.4l4.4-4.4-4.4-4.4a1 1 0 0 1 1.4-1.4l5.1 5.1a1 1 0 0 1 0 1.4l-5.1 5.1a1 1 0 0 1-1.4 0Z"/></svg></button><div class="vpsnw-protocol"><span class="vpsnw-muted">协议</span><button class="vpsnw-chip" data-vpsnw-protocol="ICMP">ICMP</button><button class="vpsnw-chip" data-vpsnw-protocol="TCP">TCP</button></div><div class="vpsnw-tool"><span class="vpsnw-muted">显示</span><button class="vpsnw-chip" id="vpsnw-extremes">极值</button><button class="vpsnw-chip" id="vpsnw-average">平均线</button></div></div>' +
        '<div class="vpsnw-empty" id="vpsnw-empty">暂无延迟数据</div>' +
        '<div class="vpsnw-legend" id="vpsnw-legend"></div>' +
        '<div class="vpsnw-chart"><canvas id="vpsnw-canvas"></canvas><div class="vpsnw-tip" id="vpsnw-tip"></div></div>';
      canvas = panel.querySelector("#vpsnw-canvas");
      ctx = canvas.getContext("2d");
      tooltip = panel.querySelector("#vpsnw-tip");
      canvas.addEventListener("wheel", onWheel, { passive: false });
      canvas.addEventListener("mousemove", onMove);
      canvas.addEventListener("mouseleave", function () { state.hover = null; tooltip.style.display = "none"; draw(); });
      canvas.addEventListener("dblclick", function () { if (state.domain) { state.view = { start: state.domain.start, end: state.domain.end }; draw(); } });
      initControls();
      window.addEventListener("resize", draw);
    }
    if (!panel.isConnected || panel.previousElementSibling !== host) {
      host.insertAdjacentElement("afterend", panel);
    }
    return panel;
  }

  function pad(n) { return String(n).padStart(2, "0"); }
  function fmtTime(ts) { var d = new Date(ts); return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()) + " " + pad(d.getHours()) + ":" + pad(d.getMinutes()) + ":" + pad(d.getSeconds()); }
  function fmtAxis(ts) { var d = new Date(ts); return pad(d.getHours()) + ":" + pad(d.getMinutes()); }
  function dateInputValue(ts) { var d = new Date(ts); return d.getFullYear() + "-" + pad(d.getMonth()+1) + "-" + pad(d.getDate()); }
  function dayRange(value) {
    var text = value || dateInputValue(Date.now());
    var parts = text.split("-").map(Number);
    var start = new Date(parts[0], parts[1] - 1, parts[2]).getTime();
    return { start: start, end: start + 86400000 };
  }
  function fmtMs(v) { return !isFinite(v) ? "-" : (v < 100 ? v.toFixed(2) : Math.round(v)) + "ms"; }
  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[ch];
    });
  }

  function initControls() {
    state.date = state.date || dateInputValue(Date.now());
    panel.querySelector("#vpsnw-date").value = state.date;
    panel.querySelector("#vpsnw-date").onchange = function () {
      if (!this.value) return;
      state.date = this.value;
      state.view = null;
      load({ resetView: true }).catch(function (err) { setEmptyText(err.message || String(err)); });
    };
    panel.querySelector("#vpsnw-prev-day").onclick = function () { shiftDate(-1); };
    panel.querySelector("#vpsnw-next-day").onclick = function () { shiftDate(1); };
    panel.querySelector("#vpsnw-extremes").onclick = function () { state.showExtremes = !state.showExtremes; draw(); };
    panel.querySelector("#vpsnw-average").onclick = function () { state.showAverage = !state.showAverage; draw(); };
    Array.prototype.forEach.call(panel.querySelectorAll("[data-vpsnw-protocol]"), function (btn) {
      btn.onclick = function () {
        state.protocol = btn.getAttribute("data-vpsnw-protocol") || "ICMP";
        draw();
      };
    });
    syncControls();
  }

  function syncControls() {
    if (!panel) return;
    var dateInput = panel.querySelector("#vpsnw-date");
    if (dateInput) dateInput.value = state.date || dateInputValue(Date.now());
    Array.prototype.forEach.call(panel.querySelectorAll("[data-vpsnw-protocol]"), function (btn) {
      btn.classList.toggle("active", btn.getAttribute("data-vpsnw-protocol") === state.protocol);
    });
    var extremes = panel.querySelector("#vpsnw-extremes");
    var average = panel.querySelector("#vpsnw-average");
    if (extremes) extremes.classList.toggle("active", state.showExtremes);
    if (average) average.classList.toggle("active", state.showAverage);
  }

  function shiftDate(days) {
    var range = dayRange(state.date);
    state.date = dateInputValue(range.start + days * 86400000);
    state.view = null;
    syncControls();
    load({ resetView: true }).catch(function (err) { setEmptyText(err.message || String(err)); });
  }

  function latencyURL() {
    var range = dayRange(state.date);
    var params = new URLSearchParams();
    params.set("period", "1d");
    params.set("start", String(range.start));
    params.set("end", String(range.end));
    return "/api/v1/netwatch/latency?" + params.toString();
  }

  function ensureSelectedServer() {
    var servers = (state.data && state.data.servers) || [];
    var saved = "";
    try { saved = window.localStorage.getItem("vpsnw-selected-server") || ""; } catch (_) {}
    function exists(id) {
      return servers.some(function (server) { return String(server.id) === String(id); });
    }
    if (state.selectedServerId && exists(state.selectedServerId)) return;
    if (saved && exists(saved)) {
      state.selectedServerId = saved;
      return;
    }
    state.selectedServerId = servers.length ? String(servers[0].id) : "";
  }

  function renderServerTabs() {
    var box = panel && panel.querySelector("#vpsnw-server-tabs");
    if (!box || !state.data) return;
    ensureSelectedServer();
    box.innerHTML = "";
    (state.data.servers || []).forEach(function (server) {
      var id = String(server.id);
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "vpsnw-server-btn" + (id === String(state.selectedServerId) ? " active" : "");
      btn.textContent = server.name || ("VPS " + id);
      btn.onclick = function () {
        state.selectedServerId = id;
        state.hover = null;
        try { window.localStorage.setItem("vpsnw-selected-server", id); } catch (_) {}
        if (state.peerTargetServerId && state.peerTargetServerId !== id) startPeerWait();
        draw();
      };
      box.appendChild(btn);
    });
  }

  function syncPeerState() {
    state.peerTargetServerId = state.data && state.data.peer && state.data.peer.enabled ? String(state.data.peer.target_server_id || "") : "";
  }

  function renderPeerTargets() {
    var box = panel && panel.querySelector("#vpsnw-peer-tabs");
    if (!box || !state.data) return;
    var servers = state.data.servers || [];
    box.innerHTML = "";

    function addButton(label, id, active, disabled, title) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "vpsnw-peer-btn" + (active ? " active" : "");
      btn.textContent = label;
      btn.disabled = disabled || state.peerSaving;
      if (title) btn.title = title;
      btn.onclick = function () { setPeerTarget(id); };
      box.appendChild(btn);
    }

    addButton("关闭", "0", !state.peerTargetServerId, false, "默认只看上海电信和上海联通");
    servers.forEach(function (server) {
      var id = String(server.id);
      var label = server.name || ("VPS " + id);
      var sameAsSource = id === String(state.selectedServerId);
      var disabled = !server.ip || sameAsSource;
      var title = sameAsSource ? "源 VPS 和目标 VPS 不能相同" : (disabled ? "这台 VPS 还没有上报公网 IP" : "让当前 VPS ping " + label);
      addButton(label, id, id === String(state.peerTargetServerId), disabled, title);
    });
  }

  async function setPeerTarget(id) {
    if (state.peerSaving) return;
    state.peerSaving = true;
    renderPeerTargets();
    try {
      var headers = { "Content-Type": "application/json" };
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      var res = await fetch("/api/v1/netwatch/peer-target", {
        method: "POST",
        headers: headers,
        credentials: "same-origin",
        body: JSON.stringify({ target_server_id: id === "0" ? 0 : Number(id) })
      });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "设置失败");
      state.peerTargetServerId = body.data && body.data.enabled ? String(body.data.target_server_id || "") : "";
      state.loaded = false;
      if (state.peerTargetServerId) {
        startPeerWait();
      } else {
        stopPeerWait();
      }
      setEmptyText(state.peerTargetServerId ? "采集中" : "暂无延迟数据");
      await load({ resetView: false });
    } catch (err) {
      setEmptyText(err.message || String(err));
      panel.querySelector("#vpsnw-empty").style.display = "block";
    } finally {
      state.peerSaving = false;
      renderPeerTargets();
    }
  }

  function aggregate() {
    if (!state.data) return [];
    ensureSelectedServer();
    var grouped = {};
    (state.data.series || []).forEach(function (raw) {
      if (state.selectedServerId && String(raw.server_id) !== String(state.selectedServerId)) return;
      if (raw.type_name !== state.protocol) return;
      var isPeer = !!raw.is_peer;
      if (isPeer) {
        if (!state.peerTargetServerId || String(raw.peer_server_id) !== String(state.peerTargetServerId)) return;
        if (String(raw.peer_server_id) === String(state.selectedServerId)) return;
      }
      var key = isPeer ? ("peer:" + String(raw.peer_server_id)) : String(raw.service_id);
      if (!grouped[key]) grouped[key] = { name: raw.service_name, target: raw.target || "", type_name: raw.type_name || "", display_index: raw.display_index || 0, peer_server_id: raw.peer_server_id || 0, is_peer: isPeer, points: {}, total: 0, count: 0 };
      (raw.data_points || []).forEach(function (p) {
        if (!p || p.status === 0 || !(p.delay > 0)) return;
        var minute = Math.floor(p.ts / 60000) * 60000;
        if (!grouped[key].points[minute]) grouped[key].points[minute] = [];
        grouped[key].points[minute].push(p.delay);
        grouped[key].total += p.delay;
        grouped[key].count += 1;
      });
    });
    return Object.keys(grouped).map(function (key) {
      var g = grouped[key];
      var points = Object.keys(g.points).map(function (ts) {
        var values = g.points[ts];
        var sum = values.reduce(function (a, b) { return a + b; }, 0);
        return { ts: Number(ts), delay: sum / values.length };
      }).sort(function (a, b) { return a.ts - b.ts; });
      return { name: g.name, target: g.target, type_name: g.type_name, display_index: g.display_index, is_peer: g.is_peer, peer_server_id: g.peer_server_id, avg: g.count ? g.total / g.count : 0, points: points };
    }).filter(function (s) { return s.points.length; }).sort(function (a, b) {
      if (a.display_index !== b.display_index) return b.display_index - a.display_index;
      return a.name.localeCompare(b.name);
    });
  }

  function setEmptyText(text) {
    var empty = panel && panel.querySelector("#vpsnw-empty");
    if (empty) empty.textContent = text || "暂无延迟数据";
  }

  function hasActivePeerSeries(series) {
    if (!state.peerTargetServerId || String(state.peerTargetServerId) === String(state.selectedServerId)) return false;
    return series.some(function (item) { return item.is_peer && String(item.peer_server_id) === String(state.peerTargetServerId) && item.points.length; });
  }

  function updatePeerNote(series) {
    var note = panel && panel.querySelector("#vpsnw-peer-note");
    if (!note) return;
    note.textContent = "";
    if (!state.peerTargetServerId) return;
    if (String(state.peerTargetServerId) === String(state.selectedServerId)) {
      note.textContent = "请选择其它目标";
      stopPeerWait();
      return;
    }
    if (hasActivePeerSeries(series)) {
      stopPeerWait();
      return;
    }
    note.textContent = state.peerWaitUntil && Date.now() <= state.peerWaitUntil ? "采集中" : "暂无数据";
  }

  function updateDomain(resetView) {
    var range = state.data && state.data.start && state.data.end ? { start: state.data.start, end: state.data.end } : dayRange(state.date);
    var start = range.start;
    var end = range.end;
    var oldView = state.view;
    state.domain = { start: start, end: end };
    if (resetView || !oldView) {
      state.view = { start: start, end: end };
    } else {
      clampView(oldView.start, oldView.end);
    }
  }

  function clampView(start, end) {
    var minRange = 5 * 60000;
    var maxRange = state.domain.end - state.domain.start;
    var range = Math.max(minRange, Math.min(maxRange, end - start));
    var center = (start + end) / 2;
    start = center - range / 2;
    end = center + range / 2;
    if (start < state.domain.start) { end += state.domain.start - start; start = state.domain.start; }
    if (end > state.domain.end) { start -= end - state.domain.end; end = state.domain.end; }
    state.view = { start: Math.max(state.domain.start, start), end: Math.min(state.domain.end, end) };
  }

  async function load(options) {
    options = options || {};
    if (state.loading) return;
    state.loading = true;
    var headers = {};
    try {
      var token = getToken();
      if (token) headers.Authorization = "Bearer " + token;
      state.date = state.date || dateInputValue(Date.now());
      var res = await fetch(latencyURL(), { headers: headers, credentials: "same-origin" });
      var body = await res.json();
      if (!body.success) throw new Error(body.error || "请求失败");
      state.data = body.data;
      state.loaded = true;
      syncPeerState();
      ensureSelectedServer();
      updateDomain(options.resetView !== false);
      syncControls();
      draw();
    } finally {
      state.loading = false;
    }
  }

  function visiblePoints(series) {
    return series.points.filter(function (p) { return p.ts >= state.view.start && p.ts <= state.view.end; });
  }

  function fillRoundRect(x, y, w, h, r) {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + w - r, y);
    ctx.quadraticCurveTo(x + w, y, x + w, y + r);
    ctx.lineTo(x + w, y + h - r);
    ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
    ctx.lineTo(x + r, y + h);
    ctx.quadraticCurveTo(x, y + h, x, y + h - r);
    ctx.lineTo(x, y + r);
    ctx.quadraticCurveTo(x, y, x + r, y);
    ctx.closePath();
    ctx.fill();
  }

  function drawValueLabel(text, px, py, color, plot) {
    ctx.save();
    ctx.font = "700 11px -apple-system,BlinkMacSystemFont,Segoe UI,Arial";
    var w = ctx.measureText(text).width + 12;
    var h = 18;
    var x0 = Math.max(plot.left, Math.min(plot.left + plot.width - w, px - w / 2));
    var y0 = Math.max(plot.top, Math.min(plot.top + plot.height - h, py - 22));
    ctx.fillStyle = color;
    fillRoundRect(x0, y0, w, h, 4);
    ctx.fillStyle = "#fff";
    ctx.fillText(text, x0 + 6, y0 + 13);
    ctx.restore();
  }

  function seriesStats(points) {
    if (!points.length) return null;
    var min = points[0];
    var max = points[0];
    var total = 0;
    points.forEach(function (p) {
      if (p.delay < min.delay) min = p;
      if (p.delay > max.delay) max = p;
      total += p.delay;
    });
    return { min: min, max: max, avg: total / points.length };
  }

  function drawStats(series, plot, x, y) {
    if (!state.showAverage && !state.showExtremes) return;
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      var stats = seriesStats(pts);
      if (!stats) return;
      var color = colors[index % colors.length];
      if (state.showAverage) {
        var yy = y(stats.avg);
        ctx.save();
        ctx.strokeStyle = color;
        ctx.globalAlpha = 0.46;
        ctx.setLineDash([6, 4]);
        ctx.beginPath();
        ctx.moveTo(plot.left, yy);
        ctx.lineTo(plot.left + plot.width, yy);
        ctx.stroke();
        ctx.restore();
        drawValueLabel("平均 " + fmtMs(stats.avg), plot.left + plot.width - 42, yy, color, plot);
      }
      if (state.showExtremes) {
        drawValueLabel("↑ " + fmtMs(stats.max.delay), x(stats.max.ts), y(stats.max.delay), "#ef6461", plot);
        if (stats.min.ts !== stats.max.ts || Math.abs(stats.min.delay - stats.max.delay) > 0.01) {
          drawValueLabel("↓ " + fmtMs(stats.min.delay), x(stats.min.ts), y(stats.min.delay) + 28, "#4fa8df", plot);
        }
      }
    });
  }

  function draw() {
    if (!panel || panel.hidden || !canvas || !ctx || !state.data) return;
    syncControls();
    renderServerTabs();
    renderPeerTargets();
    var series = aggregate();
    updatePeerNote(series);
    if (!series.length) setEmptyText(state.peerTargetServerId && state.peerTargetServerId !== state.selectedServerId && state.peerWaitUntil && Date.now() <= state.peerWaitUntil ? "采集中" : "暂无延迟数据");
    panel.querySelector("#vpsnw-empty").style.display = series.length ? "none" : "block";
    panel.querySelector(".vpsnw-chart").style.display = series.length ? "block" : "none";
    renderLegend(series);
    if (!series.length) return;

    var rect = canvas.getBoundingClientRect();
    var dpr = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, rect.width, rect.height);
    var isDark = document.documentElement.classList.contains("dark") || document.body.classList.contains("dark");
    ctx.fillStyle = isDark ? "#090909" : "#ffffff";
    ctx.fillRect(0, 0, rect.width, rect.height);

    var pad = { left: 44, right: 18, top: 18, bottom: 54 };
    var plot = { left: pad.left, top: pad.top, width: rect.width - pad.left - pad.right, height: rect.height - pad.top - pad.bottom };
    lastPlot = plot;
    var visible = [];
    series.forEach(function (s) { visiblePoints(s).forEach(function (p) { visible.push(p); }); });
    var maxY = Math.max.apply(null, (visible.length ? visible : [{ delay: 10 }]).map(function (p) { return p.delay; }));
    maxY = Math.max(10, Math.ceil(maxY * 1.18 / 10) * 10);
    function x(ts) { return plot.left + (ts - state.view.start) / (state.view.end - state.view.start) * plot.width; }
    function y(v) { return plot.top + plot.height - v / maxY * plot.height; }

    ctx.strokeStyle = isDark ? "rgba(255,255,255,.12)" : "#dbe2ee";
    ctx.lineWidth = 1;
    ctx.fillStyle = isDark ? "#cbd5e1" : "#334155";
    ctx.font = "12px -apple-system,BlinkMacSystemFont,Segoe UI,Arial";
    for (var gy = 0; gy <= 4; gy++) {
      var value = maxY * gy / 4;
      var yy = y(value);
      ctx.beginPath(); ctx.moveTo(plot.left, yy); ctx.lineTo(plot.left + plot.width, yy); ctx.stroke();
      ctx.fillText(Math.round(value) + "ms", 0, yy + 4);
    }
    for (var gx = 0; gx <= 6; gx++) {
      var ts = state.view.start + (state.view.end - state.view.start) * gx / 6;
      ctx.fillText(fmtAxis(ts), plot.left + plot.width * gx / 6 - 14, plot.top + plot.height + 22);
    }

    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      if (!pts.length) return;
      ctx.strokeStyle = colors[index % colors.length];
      ctx.lineWidth = 1.8;
      ctx.beginPath();
      pts.forEach(function (p, i) { if (i === 0) ctx.moveTo(x(p.ts), y(p.delay)); else ctx.lineTo(x(p.ts), y(p.delay)); });
      ctx.stroke();
    });
    drawStats(series, plot, x, y);

    if (state.hover) drawHover(series, plot, x, y);
  }

  function renderLegend(series) {
    var legend = panel.querySelector("#vpsnw-legend");
    legend.innerHTML = "";
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      var avg = pts.length ? pts.reduce(function (sum, p) { return sum + p.delay; }, 0) / pts.length : s.avg;
      var item = document.createElement("span");
      item.innerHTML = '<i class="vpsnw-dot" style="background:' + colors[index % colors.length] + '"></i>' + escapeHtml(s.name) + " " + fmtMs(avg);
      legend.appendChild(item);
    });
  }

  function drawHover(series, plot, x, y) {
    if (state.hover.x < plot.left || state.hover.x > plot.left + plot.width || state.hover.y < plot.top || state.hover.y > plot.top + plot.height) {
      tooltip.style.display = "none";
      return;
    }
    var hoverTs = state.view.start + (state.hover.x - plot.left) / plot.width * (state.view.end - state.view.start);
    var items = [];
    series.forEach(function (s, index) {
      var pts = visiblePoints(s);
      if (!pts.length) return;
      var nearest = pts[0];
      for (var i = 1; i < pts.length; i++) if (Math.abs(pts[i].ts - hoverTs) < Math.abs(nearest.ts - hoverTs)) nearest = pts[i];
      items.push({ s: s, p: nearest, color: colors[index % colors.length] });
    });
    if (!items.length) return;
    var lineX = x(items[0].p.ts);
    ctx.strokeStyle = "#9fb1cf";
    ctx.setLineDash([4, 3]);
    ctx.beginPath(); ctx.moveTo(lineX, plot.top); ctx.lineTo(lineX, plot.top + plot.height); ctx.stroke();
    ctx.setLineDash([]);
    items.forEach(function (item) {
      ctx.fillStyle = item.color;
      ctx.beginPath(); ctx.arc(x(item.p.ts), y(item.p.delay), 4, 0, Math.PI * 2); ctx.fill();
      ctx.strokeStyle = "#fff"; ctx.lineWidth = 1.4; ctx.stroke();
    });
    tooltip.innerHTML = '<div class="vpsnw-tip-time">' + fmtTime(items[0].p.ts) + '</div>' + items.map(function (item) {
      return '<div class="vpsnw-tip-row"><span class="vpsnw-tip-name"><i class="vpsnw-dot" style="background:' + item.color + '"></i>' + escapeHtml(item.s.name) + '</span><strong>' + fmtMs(item.p.delay) + '</strong></div>';
    }).join("");
    tooltip.style.display = "block";
    tooltip.style.left = Math.min(panel.clientWidth - tooltip.offsetWidth - 20, Math.max(10, state.hover.x + 18)) + "px";
    tooltip.style.top = Math.min(canvas.parentElement.clientHeight - tooltip.offsetHeight - 12, Math.max(10, state.hover.y - 18)) + "px";
  }

  function onWheel(event) {
    if (!state.domain || !state.view || !lastPlot) return;
    event.preventDefault();
    var rect = canvas.getBoundingClientRect();
    var x = event.clientX - rect.left;
    var center = state.view.start + Math.max(0, Math.min(1, (x - lastPlot.left) / lastPlot.width)) * (state.view.end - state.view.start);
    var range = (state.view.end - state.view.start) * (event.deltaY > 0 ? 1.22 : 0.82);
    clampView(center - range / 2, center + range / 2);
    draw();
  }

  function onMove(event) {
    var rect = canvas.getBoundingClientRect();
    state.hover = { x: event.clientX - rect.left, y: event.clientY - rect.top };
    draw();
  }

  function toggle(btn, host) {
    createPanel(host);
    state.visible = !state.visible;
    panel.hidden = !state.visible;
    btn.className = buttonClass + (state.visible ? " " + activeClass : "");
    if (state.visible) {
      startRefresh();
    } else {
      stopRefresh();
      stopPeerWait();
    }
    if (state.visible && !state.loaded) {
      load({ resetView: true }).catch(function (err) {
        setEmptyText(err.message || String(err));
        panel.querySelector("#vpsnw-empty").style.display = "block";
      });
    } else if (state.visible) {
      draw();
    }
  }

  function startRefresh() {
    if (state.refreshTimer) return;
    state.refreshTimer = window.setInterval(function () {
      if (state.visible) load({ resetView: false }).catch(function () {});
    }, 30000);
  }

  function stopRefresh() {
    if (state.refreshTimer) window.clearInterval(state.refreshTimer);
    state.refreshTimer = 0;
  }

  function startPeerWait() {
    state.peerWaitUntil = Date.now() + 120000;
    if (state.peerTimer) return;
    state.peerTimer = window.setInterval(function () {
      if (!state.visible || !state.peerWaitUntil || Date.now() > state.peerWaitUntil) {
        stopPeerWait();
        draw();
        return;
      }
      load({ resetView: false }).catch(function () {});
    }, 10000);
  }

  function stopPeerWait() {
    if (state.peerTimer) window.clearInterval(state.peerTimer);
    state.peerTimer = 0;
    state.peerWaitUntil = 0;
  }

  function ensureButton() {
    var controlsRow = document.querySelector(".server-overview-controls");
    var controls = document.querySelector(".server-overview-controls section");
    if (!controlsRow || !controls) return false;

    var btn = document.querySelector("[" + marker + "]");
    if (btn && !controls.contains(btn)) btn.remove();
    btn = controls.querySelector("[" + marker + "]");

    if (!btn) {
      btn = document.createElement("button");
      btn.setAttribute(marker, "1");
      btn.type = "button";
      btn.title = "延迟";
      btn.setAttribute("aria-label", "延迟");
      btn.innerHTML = '<svg viewBox="0 0 20 20" width="13" height="13" fill="currentColor" aria-hidden="true"><path d="M3 12.6a1 1 0 0 1 1-1h1.9l2.2-5.7a1 1 0 0 1 1.86 0l2.63 6.84 1.55-3.1a1 1 0 0 1 .9-.55H17a1 1 0 1 1 0 2h-1.34l-2.36 4.72a1 1 0 0 1-1.83-.08L9.03 9.37l-1.5 3.88a1 1 0 0 1-.93.64H4a1 1 0 0 1-1-1.29Z"/></svg>';
    }
    btn.className = buttonClass + (state.visible ? " " + activeClass : "");
    btn.onclick = function () { toggle(btn, controlsRow); };

    var buttons = Array.prototype.filter.call(controls.querySelectorAll(":scope > button"), function (item) {
      return !item.hasAttribute(marker);
    });
    if (buttons.length < 3) return !!btn.isConnected;
    if (btn.parentNode !== controls || btn.previousElementSibling !== buttons[2]) {
      buttons[2].after(btn);
    }
    return true;
  }

  ensureButton();
  window.setInterval(ensureButton, 1000);
  var observer = new MutationObserver(ensureButton);
  observer.observe(document.documentElement, { childList: true, subtree: true });
  document.addEventListener("visibilitychange", ensureButton);
})();
</script>`
