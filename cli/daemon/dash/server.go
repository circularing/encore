package dash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"encr.dev/cli/daemon/apps"
	"encr.dev/cli/daemon/dash/ai"
	"encr.dev/cli/daemon/dash/apiproxy"
	"encr.dev/cli/daemon/dash/dashproxy"
	"encr.dev/cli/daemon/engine/trace2"
	"encr.dev/cli/daemon/namespace"
	"encr.dev/cli/daemon/run"
	"encr.dev/cli/internal/jsonrpc2"
	"encr.dev/internal/conf"
	"encr.dev/pkg/fns"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// NewServer starts a new server and returns it.
func NewServer(appsMgr *apps.Manager, runMgr *run.Manager, nsMgr *namespace.Manager, tr trace2.Store, dashPort int) *Server {
	proxy, err := dashproxy.New(conf.DevDashURL)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create dash proxy")
	}

	apiProxy, err := apiproxy.New(conf.APIBaseURL + "/graphql")
	if err != nil {
		log.Fatal().Err(err).Msg("could not create graphql proxy")
	}

	aiMgr := ai.NewAIManager()

	s := &Server{
		proxy:    proxy,
		apiProxy: apiProxy,
		apps:     appsMgr,
		run:      runMgr,
		ns:       nsMgr,
		tr:       tr,
		dashPort: dashPort,
		traceCh:  make(chan trace2.NewSpanEvent, 10),
		clients:  make(map[chan<- *notification]struct{}),
		ai:       aiMgr,
	}

	runMgr.AddListener(s)
	tr.Listen(s.traceCh)
	go s.listenTraces()
	return s
}

// Server is the http.Handler for serving the developer dashboard.
type Server struct {
	proxy    *httputil.ReverseProxy
	apiProxy *httputil.ReverseProxy
	apps     *apps.Manager
	run      *run.Manager
	ns       *namespace.Manager
	tr       trace2.Store
	dashPort int
	traceCh  chan trace2.NewSpanEvent
	ai       *ai.Manager

	mu      sync.Mutex
	clients map[chan<- *notification]struct{}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/__encore/nats-column.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write([]byte(natsColumnPatchJS))
	case "/__encore/status":
		s.StatusJSON(w, req)
	case "/__encore":
		s.WebSocket(w, req)
	case "/__graphql":
		s.apiProxy.ServeHTTP(w, req)
	default:
		s.proxy.ServeHTTP(w, req)
	}
}

const natsColumnPatchJS = `(function () {
  var cachedPayload = null;
  var lastFetchMs = 0;
  var inFlight = null;
  var runTimer = null;

  function getAppID() {
    var m = window.location.pathname.match(/^\/([^/]+)\//);
    return m ? m[1] : "";
  }
  function isNATSRPC(rpc) {
    if (!rpc) return false;
    if ((rpc.proto || "").toUpperCase() === "NATS") return true;
    if (!Array.isArray(rpc.http_methods)) return false;
    return rpc.http_methods.some(function (m) { return (m || "").toUpperCase() === "NATS"; });
  }
  function extractSubject(doc) {
    if (!doc) return "";
    var m = String(doc).match(/subject\s+([a-zA-Z0-9._:-]+)/i);
    return m ? m[1] : "";
  }
  function isServiceCatalogPath() {
    return /\/envs\/[^/]+\/api(?:$|[/?#])/.test(window.location.pathname || "");
  }
  function fetchBrokerStats(appID) {
    if (!appID) return Promise.resolve({ byService: {} });
    var now = Date.now();
    if (cachedPayload && now-lastFetchMs < 5000) return Promise.resolve(cachedPayload);
    if (inFlight) return inFlight;
    inFlight = fetch("/__encore/status?appID=" + encodeURIComponent(appID))
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (st) {
        var out = { byService: {} };
        var services = st && st.apiEncoding && Array.isArray(st.apiEncoding.services) ? st.apiEncoding.services : [];
        services.forEach(function (svc) {
          if (!svc || !svc.name) return;
          var rpcs = Array.isArray(svc.rpcs) ? svc.rpcs : [];
          var natsRpcs = rpcs.filter(isNATSRPC);
          var reqReply = 0;
          var publish = 0;
          var subjects = [];
          natsRpcs.forEach(function (rpc) {
            var mode = rpc && rpc.response_encoding ? "request-reply" : "publish";
            if (mode === "request-reply") reqReply += 1; else publish += 1;
            var subj = extractSubject(rpc && rpc.doc);
            if (subj) subjects.push(subj);
          });
          out.byService[svc.name] = {
            total: natsRpcs.length,
            reqReply: reqReply,
            publish: publish,
            subjects: Array.from(new Set(subjects))
          };
        });
        cachedPayload = out;
        lastFetchMs = Date.now();
        return out;
      })
      .catch(function () { return { byService: {} }; })
      .finally(function () { inFlight = null; });
    return inFlight;
  }
  function descriptionColIndex(table) {
    var headers = table.querySelectorAll("thead tr th");
    for (var i = 0; i < headers.length; i++) {
      var txt = (headers[i].textContent || "").toLowerCase().trim();
      if (txt.indexOf("description") >= 0) return i;
    }
    return -1;
  }
  function natsColIndex(table) {
    var headers = table.querySelectorAll("thead tr th");
    for (var i = 0; i < headers.length; i++) {
      if (headers[i].getAttribute && headers[i].getAttribute("data-encore-nats-col") === "1") return i;
    }
    return -1;
  }
  function ensureHeader(table) {
    var theadRow = table.querySelector("thead tr");
    if (!theadRow) return false;
    var existing = theadRow.querySelector('[data-encore-nats-col="1"]');
    if (existing) {
      existing.style.textAlign = "left";
      existing.style.width = "280px";
      return true;
    }
    var th = document.createElement("th");
    th.textContent = "NATS";
    th.setAttribute("data-encore-nats-col", "1");
    th.style.whiteSpace = "nowrap";
    th.style.textAlign = "left";
    th.style.width = "280px";
    var descIdx = descriptionColIndex(table);
    if (descIdx >= 0 && theadRow.children[descIdx]) theadRow.insertBefore(th, theadRow.children[descIdx]);
    else theadRow.appendChild(th);
    return true;
  }
  function appendMetric(cell, value, label, active) {
    var token = document.createElement("span");
    token.textContent = String(value) + " " + label;
    token.style.display = "inline-block";
    token.style.marginRight = "18px";
    token.style.font = "inherit";
    token.style.color = "inherit";
    token.style.whiteSpace = "nowrap";
    token.style.fontWeight = active ? "600" : "500";
    token.style.opacity = active ? "1" : "0.72";
    cell.appendChild(token);
  }
  function renderNATSCell(cell, stats) {
    var total = stats && stats.total ? stats.total : 0;
    var reqReply = stats && stats.reqReply ? stats.reqReply : 0;
    var publish = stats && stats.publish ? stats.publish : 0;
    cell.style.textAlign = "left";
    cell.style.whiteSpace = "nowrap";
    cell.style.color = "inherit";
    cell.style.font = "inherit";
    cell.textContent = "";
    appendMetric(cell, total, "total", total > 0);
    appendMetric(cell, reqReply, "req/reply", reqReply > 0);
    appendMetric(cell, publish, "publish", publish > 0);

    var parts = ["Total endpoints: " + total, "Request/Reply: " + reqReply, "Publish: " + publish];
    if (stats && Array.isArray(stats.subjects) && stats.subjects.length) {
      parts.push("Subjects: " + stats.subjects.join(", "));
    }
    cell.title = parts.join(" | ");
  }
  function ensureNATSCells(table) {
    var natsIdx = natsColIndex(table);
    var rows = table.querySelectorAll("tbody tr");
    rows.forEach(function (tr) {
      var td = tr.querySelector('td[data-encore-nats-col="1"]');
      if (!td) {
        td = document.createElement("td");
        td.setAttribute("data-encore-nats-col", "1");
        if (natsIdx >= 0 && tr.children[natsIdx]) tr.insertBefore(td, tr.children[natsIdx]);
        else tr.appendChild(td);
      } else if (natsIdx >= 0) {
        var currentIdx = Array.prototype.indexOf.call(tr.children, td);
        if (currentIdx !== natsIdx) {
          if (tr.children[natsIdx]) tr.insertBefore(td, tr.children[natsIdx]);
          else tr.appendChild(td);
        }
      }
      if (!td.hasAttribute("data-encore-nats-sig")) {
        td.style.textAlign = "left";
        td.style.whiteSpace = "nowrap";
        td.textContent = "";
      }
    });
  }
  function patchRows(table, statsByService) {
    var natsIdx = natsColIndex(table);
    var rows = table.querySelectorAll("tbody tr");
    rows.forEach(function (tr) {
      var first = tr.querySelector("td");
      if (!first) return;
      var svc = (first.textContent || "").trim().split(/\s+/)[0];
      if (!svc) return;
      var stats = statsByService[svc] || { total: 0, reqReply: 0, publish: 0, subjects: [] };
      var sig = String(stats.total || 0) + ":" + String(stats.reqReply || 0) + ":" + String(stats.publish || 0) + ":" + (stats.subjects || []).join(",");
      var td = tr.querySelector('td[data-encore-nats-col="1"]');
      if (!td) {
        td = document.createElement("td");
        td.setAttribute("data-encore-nats-col", "1");
        if (natsIdx >= 0 && tr.children[natsIdx]) tr.insertBefore(td, tr.children[natsIdx]);
        else tr.appendChild(td);
      } else if (natsIdx >= 0) {
        var currentIdx = Array.prototype.indexOf.call(tr.children, td);
        if (currentIdx !== natsIdx) {
          if (tr.children[natsIdx]) tr.insertBefore(td, tr.children[natsIdx]);
          else tr.appendChild(td);
        }
      }
      if (td.getAttribute("data-encore-nats-sig") === sig) return;
      td.setAttribute("data-encore-nats-sig", sig);
      renderNATSCell(td, stats);
    });
  }
  function findServicesTable() {
    var tables = document.querySelectorAll("table");
    for (var i = 0; i < tables.length; i++) {
      var t = tables[i];
      var txt = (t.textContent || "").toLowerCase();
      if (txt.indexOf("endpoints") >= 0 && txt.indexOf("description") >= 0 && txt.indexOf("name") >= 0) {
        return t;
      }
    }
    return null;
  }
  function run() {
    if (!isServiceCatalogPath()) return;
    var table = findServicesTable();
    if (table && ensureHeader(table)) {
      ensureNATSCells(table);
      if (cachedPayload && cachedPayload.byService) patchRows(table, cachedPayload.byService);
    }
    fetchBrokerStats(getAppID()).then(function (payload) {
      var table = findServicesTable();
      if (table && ensureHeader(table)) {
        patchRows(table, (payload && payload.byService) || {});
      }
    });
  }
  function scheduleRun() {
    if (runTimer) return;
    runTimer = setTimeout(function () {
      runTimer = null;
      run();
    }, 24);
  }
  var mo = new MutationObserver(function () { scheduleRun(); });
  mo.observe(document.documentElement, { childList: true, subtree: true });
  window.addEventListener("popstate", scheduleRun);
  window.addEventListener("hashchange", scheduleRun);
  window.addEventListener("DOMContentLoaded", run);
  run();
  scheduleRun();
})();`

func (s *Server) StatusJSON(w http.ResponseWriter, req *http.Request) {
	appID := strings.TrimSpace(req.URL.Query().Get("appID"))
	if appID == "" {
		http.Error(w, "missing appID", http.StatusBadRequest)
		return
	}

	appInst, err := s.apps.FindLatestByPlatformOrLocalID(appID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	st, err := buildAppStatus(appInst, s.run.FindRunByAppID(appID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

// WebSocket serves the jsonrpc2 API over WebSocket.
func (s *Server) WebSocket(w http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Error().Err(err).Msg("dash: could not upgrade websocket")
		return
	}
	defer fns.CloseIgnore(c)
	log.Info().Msg("dash: websocket connection established")

	stream := &wsStream{c: c}
	conn := jsonrpc2.NewConn(stream)
	handler := &handler{rpc: conn, apps: s.apps, run: s.run, ns: s.ns, tr: s.tr, ai: s.ai}
	conn.Go(req.Context(), handler.Handle)

	ch := make(chan *notification, 20)
	s.addClient(ch)
	defer s.removeClient(ch)

	// nosemgrep: tools.semgrep-rules.semgrep-go.http-request-go-context
	go handler.listenNotify(req.Context(), ch)

	<-conn.Done()
	if err := conn.Err(); err != nil {
		if ce, ok := err.(*websocket.CloseError); ok && ce.Code == websocket.CloseNormalClosure {
			log.Info().Msg("dash: websocket closed")
		} else {
			log.Info().Err(err).Msg("dash: websocket closed with error")
		}
	}
}

func (s *Server) addClient(ch chan *notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[ch] = struct{}{}
}

func (s *Server) removeClient(ch chan *notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
}

// hasClients reports whether there are any active clients.
func (s *Server) hasClients() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients) > 0
}

type notification struct {
	Method string
	Params interface{}
}

// notify notifies any active clients.
func (s *Server) notify(n *notification) {
	var clients []chan<- *notification
	s.mu.Lock()
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.Unlock()

	for _, c := range clients {
		select {
		case c <- n:
		default:
		}
	}
}

// wsStream implements jsonrpc2.Stream over a websocket.
type wsStream struct {
	writeMu sync.Mutex
	c       *websocket.Conn
}

func (s *wsStream) Close() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.c.Close()
}

func (s *wsStream) Read(context.Context) (jsonrpc2.Message, int64, error) {
	typ, data, err := s.c.ReadMessage()
	if err != nil {
		return nil, 0, err
	}
	if typ != websocket.TextMessage {
		return nil, 0, fmt.Errorf("webedit.wsStream: got non-text message type %v", typ)
	}
	msg, err := jsonrpc2.DecodeMessage(data)
	if err != nil {
		return nil, 0, err
	}
	return msg, int64(len(data)), nil
}

func (s *wsStream) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	err = s.c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}
