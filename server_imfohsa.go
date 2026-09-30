package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var port = 8815

type Server struct {
	mu               sync.RWMutex
	state            map[string]interface{}
	version          int64
	sessions         map[string]string
	publicURL        string
	root             string
	eventClients     map[chan int64]bool
	captures         map[string]*CaptureSession
	tunnelRunning    bool
	tunnelCmd        *exec.Cmd
	tunnelGeneration uint64
}

type Admin struct {
	Username     string `json:"username"`
	Name         string `json:"name"`
	PasswordHash string `json:"passwordHash"`
	Active       bool   `json:"active"`
	Role         string `json:"role,omitempty"`
	MessengerID  string `json:"messengerId,omitempty"`
}

type CaptureSession struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Kind      string    `json:"kind"`
	Mode      string    `json:"mode"`
	Owner     string    `json:"owner"`
	Min       int       `json:"min"`
	Max       int       `json:"max"`
	URLs      []string  `json:"urls"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"-"`
	Closed    bool      `json:"closed"`
}

func main() {
	if value := os.Getenv("PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 { log.Fatal("PORT inválido") }
		port = parsed
	}
	exe, _ := os.Executable()
	root := filepath.Dir(exe)
	if runtime.GOOS != "windows" {
		if wd, err := os.Getwd(); err == nil {
			root = wd
		}
	}
	s := &Server{sessions: map[string]string{}, root: root, eventClients: map[chan int64]bool{}, captures: map[string]*CaptureSession{}}
	s.publicURL = strings.TrimRight(os.Getenv("RENDER_EXTERNAL_URL"), "/")
	if err := s.loadState(); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/info", s.info)
	mux.HandleFunc("/api/verify-public", s.verifyPublicLink)
	mux.HandleFunc("/api/share/admin", s.shareAdminInfo)
	mux.HandleFunc("/api/share/driver", s.shareDriverInfo)
	mux.HandleFunc("/api/share/panel-request", s.sharePanelRequestInfo)
	mux.HandleFunc("/api/share/panel", s.sharePanelInfo)
	mux.HandleFunc("/api/share/client-upload", s.shareClientUploadInfo)
	mux.HandleFunc("/api/share/seller", s.shareSellerInfo)
	mux.HandleFunc("/compartir-admin", s.shareAdminPage)
	mux.HandleFunc("/compartir-ruta", s.shareRoutePage)
	mux.HandleFunc("/compartir-solicitud-panel", s.sharePanelRequestPage)
	mux.HandleFunc("/captura", s.capturePage)
	mux.HandleFunc("/api/capture/create", s.captureCreate)
	mux.HandleFunc("/api/capture/status", s.captureStatus)
	mux.HandleFunc("/api/capture/upload", s.captureUpload)
	mux.HandleFunc("/api/capture/result", s.captureResult)
	mux.HandleFunc("/api/capture/finish", s.captureFinish)
	mux.HandleFunc("/api/tunnel/restart", s.restartTunnel)
	mux.HandleFunc("/api/login", s.login)
	mux.HandleFunc("/api/logout", s.logout)
	mux.HandleFunc("/api/me", s.me)
	mux.HandleFunc("/api/state", s.stateHandler)
	mux.HandleFunc("/api/events", s.events)
	mux.HandleFunc("/api/admins", s.adminsHandler)
	mux.HandleFunc("/api/route", s.routeProxy)
	mux.HandleFunc("/api/geocode", s.geocodeProxy)
	mux.HandleFunc("/api/credit-status", s.creditStatus)
	mux.HandleFunc("/api/table", s.tableProxy)
	mux.HandleFunc("/api/traffic-route", s.trafficRoute)
	mux.HandleFunc("/api/upload", s.upload)
	mux.HandleFunc("/api/public/driver", s.publicDriver)
	mux.HandleFunc("/api/public/driver-action", s.publicDriverAction)
	mux.HandleFunc("/api/public/panel", s.publicPanel)
	mux.HandleFunc("/api/public/panel-action", s.publicPanelAction)
	mux.HandleFunc("/api/public/fleet-availability", s.fleetAvailability)
	mux.HandleFunc("/api/public/panel-request", s.panelRequest)
	mux.HandleFunc("/api/public/client-upload", s.publicClientUpload)
	mux.HandleFunc("/api/public/seller", s.publicSeller)
	mux.HandleFunc("/uploads/", s.uploadsServe)
	mux.HandleFunc("/", s.static)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo abrir puerto %d: %v", port, err)
	}
	fmt.Println("============================================================")
	fmt.Println(" LOGISTICA IMFOHSA R7.7.6 PILOTO - INTERNET + GOOGLE MAPS + MENSAJEROS")
	fmt.Println("============================================================")
	fmt.Printf("Panel local: http://localhost:%d/index.html\n", port)
	if ip := localIP(); ip != "" {
		fmt.Printf("Red local : http://%s:%d/index.html\n", ip, port)
	}
	if os.Getenv("IMFOHSA_DISABLE_TUNNEL") == "1" {
		fmt.Println("Panel listo. Tunel HTTPS deshabilitado por entorno de prueba.")
	} else {
		fmt.Println("Panel listo. Preparando enlace HTTPS publico en segundo plano...")
		go func() {
			time.Sleep(150 * time.Millisecond)
			s.startTunnel()
		}()
	}
	srv := &http.Server{Handler: withHeaders(recoverHTTP(mux)), ReadHeaderTimeout: 15 * time.Second}
	log.Fatal(srv.Serve(ln))
}

func withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Permite que una pestaña antigua pueda verificar el servidor nuevo durante la migración.
		origin := r.Header.Get("Origin")
		if origin == "http://"+r.Host || origin == "https://"+r.Host {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func recoverHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC %s %s: %v", r.Method, r.URL.Path, rec)
				http.Error(w, "Error interno recuperado; vuelva a intentar", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func (s *Server) statePath() string  { return filepath.Join(s.root, "data", "state.json") }
func (s *Server) adminsPath() string { return filepath.Join(s.root, "data", "admins.json") }
func (s *Server) loadState() error {
	os.MkdirAll(filepath.Join(s.root, "data"), 0755)
	os.MkdirAll(filepath.Join(s.root, "uploads"), 0755)
	b, err := os.ReadFile(s.statePath())
	if err != nil {
		b, err = os.ReadFile(filepath.Join(s.root, "data", "default_state.json"))
		if err != nil {
			return fmt.Errorf("falta data/default_state.json")
		}
		os.WriteFile(s.statePath(), b, 0644)
	}
	var x map[string]interface{}
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	s.state = x
	if v, ok := x["version"].(float64); ok {
		s.version = int64(v)
	} else {
		s.version = 1
	}
	if _, err := os.Stat(s.adminsPath()); err != nil {
		admins := []Admin{
			{Username: "admin1", Name: "Gerencia General", PasswordHash: hash("Imfohsa#2026"), Active: true, Role: "gerente_general"},
			{Username: "admin2", Name: "Jefe de Logística", PasswordHash: hash("Imfohsa#2026"), Active: true, Role: "jefe_logistica"},
			{Username: "admin3", Name: "Asistente de Logística", PasswordHash: hash("Imfohsa#2026"), Active: true, Role: "asistente_logistica"},
			{Username: "admin4", Name: "Ventas", PasswordHash: hash("Imfohsa#2026"), Active: true, Role: "ventas"},
		}
		bb, _ := json.MarshalIndent(admins, "", "  ")
		os.WriteFile(s.adminsPath(), bb, 0600)
	}
	if password := os.Getenv("IMFOHSA_BOOTSTRAP_PASSWORD"); password != "" {
		var admins []Admin
		bb, err := os.ReadFile(s.adminsPath())
		if err != nil { return err }
		if err := json.Unmarshal(bb, &admins); err != nil { return err }
		for i := range admins { admins[i].PasswordHash = hash(password) }
		bb, err = json.MarshalIndent(admins, "", "  ")
		if err != nil { return err }
		if err := os.WriteFile(s.adminsPath(), bb, 0600); err != nil { return err }
	}
	return nil
}
func hash(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func token() string        { b := make([]byte, 24); rand.Read(b); return hex.EncodeToString(b) }
func sval(v interface{}) string {
	if v == nil {
		return ""
	}
	x := strings.TrimSpace(fmt.Sprint(v))
	if x == "<nil>" {
		return ""
	}
	return x
}
func (s *Server) saveLocked() error {
	s.version++
	s.state["version"] = s.version
	b, _ := json.MarshalIndent(s.state, "", "  ")
	if err := os.WriteFile(s.statePath(), b, 0644); err != nil {
		return err
	}
	go s.broadcast()
	return nil
}
func (s *Server) broadcast() {
	s.mu.RLock()
	v := s.version
	chans := make([]chan int64, 0, len(s.eventClients))
	for ch := range s.eventClients {
		chans = append(chans, ch)
	}
	s.mu.RUnlock()
	for _, ch := range chans {
		select {
		case ch <- v:
		default:
		}
	}
}
func jsonOut(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func bodyJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(io.LimitReader(r.Body, 20<<20)).Decode(v)
}
func (s *Server) authUser(r *http.Request) string {
	c, err := r.Cookie("imfohsa_session")
	if err != nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[c.Value]
}
func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case "gerente_operaciones", "jefe_logistica", "asistente_z16", "coordinador_z4", "ventas", "creditos_cobros", "mensajero":
		return r
	case "gerente_general":
		return "gerente_operaciones"
	case "asistente_logistica":
		return "asistente_z16"
	default:
		return "jefe_logistica"
	}
}
func (s *Server) roleForUser(username string) string {
	b, _ := os.ReadFile(s.adminsPath())
	var admins []Admin
	_ = json.Unmarshal(b, &admins)
	for _, a := range admins {
		if strings.EqualFold(a.Username, username) {
			return normalizeRole(a.Role)
		}
	}
	return "jefe_logistica"
}
func (s *Server) messengerAccessForUser(username string) (string, string) {
	b, _ := os.ReadFile(s.adminsPath())
	var admins []Admin
	_ = json.Unmarshal(b, &admins)
	messengerID := ""
	for _, a := range admins {
		if strings.EqualFold(a.Username, username) && normalizeRole(a.Role) == "mensajero" {
			messengerID = strings.TrimSpace(a.MessengerID)
			break
		}
	}
	if messengerID == "" {
		return "", ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	messengers, _ := s.state["messengers"].([]interface{})
	for _, raw := range messengers {
		m, _ := raw.(map[string]interface{})
		if strings.TrimSpace(fmt.Sprint(m["id"])) != messengerID {
			continue
		}
		tok := strings.TrimSpace(fmt.Sprint(m["token"]))
		if tok == "" {
			return messengerID, ""
		}
		_ = tok // el acceso autenticado del mensajero no expone tokens en la URL
		return messengerID, "/index.html?driver=" + url.QueryEscape(messengerID) + "&auth=1"
	}
	return messengerID, ""
}

func managerRole(role string) bool {
	r := normalizeRole(role)
	return r == "gerente_operaciones" || r == "jefe_logistica"
}
func operationalRole(role string) bool {
	r := normalizeRole(role)
	return r == "asistente_z16" || r == "coordinador_z4"
}
func operationalCenter(role string) string {
	switch normalizeRole(role) {
	case "asistente_z16":
		return "cedi16"
	case "coordinador_z4":
		return "lab4"
	}
	return ""
}
func jsonEqual(a, b interface{}) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return bytes.Equal(aa, bb)
}
func mapOf(v interface{}) map[string]interface{} { m, _ := v.(map[string]interface{}); return m }
func orderCenter(o map[string]interface{}) string {
	if o == nil {
		return ""
	}
	c := strings.TrimSpace(fmt.Sprint(o["centerId"]))
	if c == "" {
		return "cedi16"
	}
	return c
}
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	u := s.authUser(r)
	if u == "" {
		http.Error(w, "No autorizado", 401)
		return "", false
	}
	role := s.roleForUser(u)
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		// /api/state tiene validación granular por rol dentro de stateHandler.
		if r.URL.Path == "/api/state" && (managerRole(role) || operationalRole(role) || normalizeRole(role) == "creditos_cobros") {
			return u, true
		}
		if !managerRole(role) {
			http.Error(w, "Permiso insuficiente para modificar este recurso", http.StatusForbidden)
			return "", false
		}
	}
	return u, true
}
func (s *Server) requireManager(w http.ResponseWriter, r *http.Request) (string, bool) {
	u := s.authUser(r)
	if u == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return "", false
	}
	if !managerRole(s.roleForUser(u)) {
		http.Error(w, "Permiso insuficiente", http.StatusForbidden)
		return "", false
	}
	return u, true
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	jsonOut(w, map[string]interface{}{"ok": true, "version": s.version})
}
func (s *Server) verifyPublicLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	pathQ := strings.TrimSpace(r.URL.Query().Get("path"))
	if raw == "" {
		s.mu.RLock()
		pub := strings.TrimSpace(s.publicURL)
		s.mu.RUnlock()
		if pub == "" {
			jsonOut(w, map[string]interface{}{"ok": false, "reason": "no-public-url"})
			return
		}
		if pathQ == "" {
			pathQ = "/api/health"
		}
		raw = strings.TrimRight(pub, "/") + pathQ
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		jsonOut(w, map[string]interface{}{"ok": false, "reason": "invalid-url", "url": raw})
		return
	}
	cli := &http.Client{Timeout: 7 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	req, _ := http.NewRequest(http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", "IMFOHSA-R7.5-Verify")
	resp, err := cli.Do(req)
	if err != nil {
		jsonOut(w, map[string]interface{}{"ok": false, "reason": err.Error(), "url": raw})
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 32768))
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	jsonOut(w, map[string]interface{}{"ok": ok, "status": resp.StatusCode, "url": raw})
}
func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	p := s.publicURL
	s.mu.RUnlock()
	lan := ""
	if ip := localIP(); ip != "" {
		lan = fmt.Sprintf("http://%s:%d", ip, port)
	}
	s.mu.RLock()
	running := s.tunnelRunning
	s.mu.RUnlock()
	jsonOut(w, map[string]interface{}{"publicUrl": p, "lanUrl": lan, "localUrl": fmt.Sprintf("http://localhost:%d", port), "version": s.version, "tunnelRunning": running})
}

func (s *Server) bestBase() (string, string) {
	s.mu.RLock()
	pub := s.publicURL
	s.mu.RUnlock()
	if pub != "" {
		return strings.TrimRight(pub, "/"), "public"
	}
	if ip := localIP(); ip != "" {
		return fmt.Sprintf("http://%s:%d", ip, port), "lan"
	}
	return fmt.Sprintf("http://localhost:%d", port), "local"
}

// ensurePublicBase nunca bloquea una solicitud normal. Los endpoints de
// compartir crean token/ruta de inmediato y el HTTPS se prepara aparte.
func (s *Server) ensurePublicBase(maxWait time.Duration) (string, string) {
	_ = maxWait
	s.mu.RLock()
	pub := strings.TrimSpace(s.publicURL)
	running := s.tunnelRunning
	s.mu.RUnlock()
	if pub != "" {
		return strings.TrimRight(pub, "/"), "public"
	}
	if !running {
		go s.startTunnel()
	}
	return s.bestBase()
}

func (s *Server) shareAdminInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	base, mode := s.ensurePublicBase(40 * time.Second)
	path := "/index.html"
	jsonOut(w, map[string]interface{}{
		"ok": true, "mode": mode, "path": path,
		"url":         strings.TrimRight(base, "/") + path,
		"publicReady": mode == "public", "serverPort": port, "generatedAt": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) sharePanelRequestInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.mu.Lock()
	st, _ := s.state["settings"].(map[string]interface{})
	if st == nil {
		st = map[string]interface{}{}
		s.state["settings"] = st
	}
	pt := sval(st["panelRequestPublicToken"])
	if pt == "" || r.URL.Query().Get("reset") == "1" {
		pt = token()
		st["panelRequestPublicToken"] = pt
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	base, mode := s.ensurePublicBase(40 * time.Second)
	path := "/index.html?request=panel&token=" + url.QueryEscape(pt)
	jsonOut(w, map[string]interface{}{
		"ok": true, "mode": mode, "path": path, "token": pt,
		"url": strings.TrimRight(base, "/") + path, "publicReady": mode == "public",
	})
}

func (s *Server) shareClientUploadInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.mu.Lock()
	st, _ := s.state["settings"].(map[string]interface{})
	if st == nil {
		st = map[string]interface{}{}
		s.state["settings"] = st
	}
	t := sval(st["clientUploadPublicToken"])
	if t == "" || r.URL.Query().Get("reset") == "1" {
		t = token()
		st["clientUploadPublicToken"] = t
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	base, mode := s.ensurePublicBase(40 * time.Second)
	path := "/index.html?clientUpload=1&token=" + url.QueryEscape(t)
	jsonOut(w, map[string]interface{}{
		"ok": true, "mode": mode, "path": path, "token": t,
		"url": strings.TrimRight(base, "/") + path, "publicReady": mode == "public",
	})
}

func (s *Server) shareSellerInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.mu.Lock()
	st, _ := s.state["settings"].(map[string]interface{})
	if st == nil {
		st = map[string]interface{}{}
		s.state["settings"] = st
	}
	t := strings.TrimSpace(sval(st["sellerPublicToken"]))
	if t == "" || r.URL.Query().Get("reset") == "1" {
		t = token()
		st["sellerPublicToken"] = t
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	base, mode := s.ensurePublicBase(40 * time.Second)
	path := "/index.html?seller=1&token=" + url.QueryEscape(t)
	jsonOut(w, map[string]interface{}{"ok": true, "mode": mode, "path": path, "token": t, "url": strings.TrimRight(base, "/") + path, "publicReady": mode == "public"})
}

func (s *Server) validSellerToken(t string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, _ := s.state["settings"].(map[string]interface{})
	return st != nil && strings.TrimSpace(t) != "" && t == sval(st["sellerPublicToken"])
}

func (s *Server) publicSeller(w http.ResponseWriter, r *http.Request) {
	if !s.validSellerToken(r.URL.Query().Get("token")) {
		http.Error(w, "Enlace de consulta inválido", http.StatusForbidden)
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(q) < 2 {
		jsonOut(w, map[string]interface{}{"ok": true, "orders": []interface{}{}})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []interface{}{}
	for _, raw := range s.state["orders"].([]interface{}) {
		o, _ := raw.(map[string]interface{})
		hay := strings.ToLower(sval(o["orderNo"]) + " " + sval(o["vendorName"]) + " " + sval(o["clientName"]))
		if !strings.Contains(hay, q) {
			continue
		}
		row := map[string]interface{}{
			"orderNo": o["orderNo"], "clientName": o["clientName"], "address": o["address"], "status": o["status"],
			"date": o["date"], "cut": o["cut"], "promise": o["promise"], "operationType": o["operationType"], "vendorName": o["vendorName"],
			"photos": o["photos"], "returnPhotos": o["returnPhotos"], "deliveredAt": o["deliveredAt"], "receivedBy": o["receivedBy"], "phone": o["phone"],
		}
		if mid := sval(o["messengerId"]); mid != "" {
			for _, mr := range s.state["messengers"].([]interface{}) {
				m, _ := mr.(map[string]interface{})
				if sval(m["id"]) == mid {
					row["messenger"] = m["name"]
					break
				}
			}
		}
		out = append(out, row)
		if len(out) >= 50 {
			break
		}
	}
	jsonOut(w, map[string]interface{}{"ok": true, "orders": out})
}

func (s *Server) sharePanelInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	base, mode := s.ensurePublicBase(40 * time.Second)
	s.mu.RLock()
	var found map[string]interface{}
	if arr, ok := s.state["panelAssignments"].([]interface{}); ok {
		for _, v := range arr {
			a, _ := v.(map[string]interface{})
			if fmt.Sprint(a["id"]) == id {
				found = a
				break
			}
		}
	}
	s.mu.RUnlock()
	if found == nil {
		http.Error(w, "Asignación no encontrada", 404)
		return
	}
	t := sval(found["token"])
	if t == "" {
		http.Error(w, "Asignación sin token", 409)
		return
	}
	path := "/index.html?panel=" + url.QueryEscape(id) + "&token=" + url.QueryEscape(t)
	jsonOut(w, map[string]interface{}{"ok": true, "mode": mode, "path": path, "token": t, "url": strings.TrimRight(base, "/") + path, "publicReady": mode == "public"})
}

func (s *Server) shareDriverInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	mid := strings.TrimSpace(r.URL.Query().Get("mid"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	reset := r.URL.Query().Get("reset") == "1"
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	s.mu.Lock()
	var name, phone, accessToken string
	found := false
	if arr, ok := s.state["messengers"].([]interface{}); ok {
		for _, v := range arr {
			m, _ := v.(map[string]interface{})
			if fmt.Sprint(m["id"]) != mid {
				continue
			}
			found = true
			name = fmt.Sprint(m["name"])
			phone = fmt.Sprint(m["phone"])
			if reset || sval(m["routeAccessDate"]) != date || sval(m["routeAccessToken"]) == "" {
				m["routeAccessDate"] = date
				m["routeAccessToken"] = token()
				m["routeAccessDevice"] = ""
				m["routeAccessIssuedAt"] = time.Now().Format(time.RFC3339)
			}
			accessToken = sval(m["routeAccessToken"])
			if fmt.Sprint(m["routeAssignedDate"]) != date {
				m["routeAssignedDate"] = date
				m["routeAssignedAt"] = time.Now().Format(time.RFC3339)
				m["routeStatus"] = "Pendiente"
			}
			break
		}
	}
	if found {
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	if !found {
		http.Error(w, "Mensajero no encontrado", 404)
		return
	}

	var base, mode string
	if r.URL.Query().Get("fast") == "1" {
		s.mu.RLock()
		pub := strings.TrimSpace(s.publicURL)
		s.mu.RUnlock()
		if pub != "" {
			base, mode = strings.TrimRight(pub, "/"), "public"
		} else if ip := localIP(); ip != "" {
			base, mode = fmt.Sprintf("http://%s:%d", ip, port), "lan"
		} else {
			base, mode = fmt.Sprintf("http://localhost:%d", port), "local"
		}
	} else {
		base, mode = s.ensurePublicBase(40 * time.Second)
	}
	path := fmt.Sprintf("/index.html?driver=%s&token=%s&date=%s",
		url.QueryEscape(mid), url.QueryEscape(accessToken), url.QueryEscape(date))
	previewPath := fmt.Sprintf("/index.html?driver=%s&date=%s&preview=1",
		url.QueryEscape(mid), url.QueryEscape(date))
	jsonOut(w, map[string]interface{}{
		"ok": true, "mode": mode, "publicReady": mode == "public", "name": name, "phone": phone,
		"url": strings.TrimRight(base, "/") + path, "path": path,
		"previewUrl": previewPath, "deviceLocked": true,
	})
}

func (s *Server) ensureDeviceCookie(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie("imfohsa_route_device"); err == nil && strings.TrimSpace(c.Value) != "" {
		return c.Value
	}
	d := token()
	http.SetCookie(w, &http.Cookie{
		Name: "imfohsa_route_device", Value: d, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 365 * 24 * 3600, Secure: r.TLS != nil || os.Getenv("RENDER") == "true",
	})
	return d
}

func (s *Server) validateDriverAccess(w http.ResponseWriter, r *http.Request, driver, t, date string, allowPreview bool) bool {
	if user := s.authUser(r); user != "" {
		role := s.roleForUser(user)
		if allowPreview && r.URL.Query().Get("preview") == "1" {
			return true
		}
		if role == "mensajero" {
			mid, _ := s.messengerAccessForUser(user)
			if mid != "" && mid == driver {
				return true
			}
		}
	}
	if strings.TrimSpace(driver) == "" || strings.TrimSpace(date) == "" {
		return false
	}

	device := ""
	if c, err := r.Cookie("imfohsa_route_device"); err == nil {
		device = c.Value
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	arr, _ := s.state["messengers"].([]interface{})
	for _, v := range arr {
		m, _ := v.(map[string]interface{})
		if fmt.Sprint(m["id"]) != driver {
			continue
		}
		if sval(m["routeAccessDate"]) != date {
			return false
		}

		expected := sval(m["routeAccessToken"])
		bound := fmt.Sprint(m["routeAccessDevice"])

		// Once claimed, the same browser/device can continue without the token in the URL.
		if bound != "" {
			return device != "" && hash(device) == bound
		}
		if t == "" || expected == "" || t != expected {
			return false
		}

		if device == "" {
			device = s.ensureDeviceCookie(w, r)
		}
		m["routeAccessDevice"] = hash(device)
		m["routeAccessClaimedAt"] = time.Now().Format(time.RFC3339)
		_ = s.saveLocked()
		return true
	}
	return false
}

func sharePageHTML(title, status, link, wa, email, extra string, autoRefresh bool) string {
	refresh := ""
	if autoRefresh {
		refresh = `<meta http-equiv="refresh" content="4">`
	}
	mailBtn := ""
	if email != "" {
		mailBtn = `<a class="secondary" href="` + html.EscapeString(email) + `">✉ Enviar por correo</a>`
	}
	return fmt.Sprintf(`<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">%s<title>%s</title><style>
	body{font-family:Segoe UI,Arial,sans-serif;background:#071526;color:#eef6ff;margin:0;padding:24px}.box{max-width:860px;margin:28px auto;background:#0d2036;border:1px solid #2b4f78;border-radius:18px;padding:28px;box-shadow:0 18px 50px #0006}h1{margin:0 0 8px}.muted{color:#a9c0da;line-height:1.5}.status{background:#102b48;border:1px solid #315e8e;padding:14px;border-radius:12px;margin:18px 0}.link{width:100%%;box-sizing:border-box;background:#06111f;color:#fff;border:1px solid #4178b7;border-radius:10px;padding:14px;font-size:16px}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:16px}a,button{border:0;cursor:pointer;display:inline-block;text-decoration:none;color:#fff;background:#1e62d0;border-radius:10px;padding:12px 16px;font-weight:700;font-size:14px}.secondary{background:#1d334d}.danger{background:#a62b36}.note{font-size:13px;color:#9fb6d0;margin-top:18px}.ok{color:#75e3a4}.warn{color:#ffd166}</style></head><body><div class="box"><h1>%s</h1><div class="status">%s</div><p class="muted">Enlace:</p><input id="shareLink" class="link" value="%s" readonly onclick="this.select()"><div class="actions"><button onclick="navigator.clipboard?navigator.clipboard.writeText(document.getElementById('shareLink').value):document.getElementById('shareLink').select()">📋 Copiar enlace</button><a href="%s" target="_blank">Abrir</a><a class="secondary" href="%s" target="_blank">💬 WhatsApp</a>%s</div>%s<p class="note">La computadora que ejecuta IMFOHSA debe permanecer encendida y con la ventana del servidor abierta. Para acceso desde otra red se requiere que el enlace comience con https://...trycloudflare.com.</p></div></body></html>`, refresh, html.EscapeString(title), html.EscapeString(title), status, html.EscapeString(link), html.EscapeString(link), html.EscapeString(wa), mailBtn, extra)
}

func (s *Server) shareAdminPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	base, mode := s.bestBase()
	link := strings.TrimRight(base, "/") + "/index.html"
	status := `<span class="ok"><b>✅ Enlace público HTTPS listo.</b></span> Puede utilizarse desde computadoras en diferentes redes.`
	auto := false
	if mode == "lan" {
		status = `<span class="warn"><b>⏳ Enlace público preparándose.</b></span> Mientras tanto este enlace funciona para equipos conectados a la misma red de la computadora principal.`
		auto = true
	}
	if mode == "local" {
		status = `<span class="warn"><b>⚠ Aún no hay enlace compartible.</b></span> El servidor no detectó una IP de red. Espere el túnel HTTPS o revise la conexión de la computadora principal.`
		auto = true
	}
	wa := "https://wa.me/?text=" + url.QueryEscape("LOGÍSTICA IMFOHSA - Acceso administrativo\n"+link)
	extra := `<div class="actions"><button class="secondary" onclick="fetch('/api/tunnel/restart',{method:'POST'}).then(()=>setTimeout(()=>location.reload(),2500))">↻ Reintentar enlace público</button></div>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, sharePageHTML("🔗 Compartir panel administrativo", status, link, wa, "", extra, auto))
}

func (s *Server) shareRoutePage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	mid := strings.TrimSpace(r.URL.Query().Get("mid"))
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	s.mu.Lock()
	var name, tok, phone string
	if arr, ok := s.state["messengers"].([]interface{}); ok {
		for _, v := range arr {
			m, _ := v.(map[string]interface{})
			if fmt.Sprint(m["id"]) != mid {
				continue
			}
			name = fmt.Sprint(m["name"])
			phone = fmt.Sprint(m["phone"])
			if sval(m["routeAccessDate"]) != date || sval(m["routeAccessToken"]) == "" {
				m["routeAccessDate"] = date
				m["routeAccessToken"] = token()
				m["routeAccessDevice"] = ""
				m["routeAccessIssuedAt"] = time.Now().Format(time.RFC3339)
			}
			tok = sval(m["routeAccessToken"])
			if fmt.Sprint(m["routeAssignedDate"]) != date {
				m["routeAssignedAt"] = time.Now().Format(time.RFC3339)
				m["routeAssignedDate"] = date
				m["routeStatus"] = "Pendiente"
			}
			_ = s.saveLocked()
			break
		}
	}
	s.mu.Unlock()
	if tok == "" {
		http.Error(w, "Mensajero no encontrado", 404)
		return
	}
	base, mode := s.ensurePublicBase(40 * time.Second)
	link := fmt.Sprintf("%s/index.html?driver=%s&token=%s&date=%s", strings.TrimRight(base, "/"), url.QueryEscape(mid), url.QueryEscape(tok), url.QueryEscape(date))
	status := `<span class="ok"><b>✅ Ruta HTTPS verificada y lista para compartir.</b></span>`
	if mode != "public" {
		status = `<span class="warn"><b>⚠ No se pudo preparar HTTPS.</b></span> No comparta este enlace por WhatsApp fuera de la red local; pulse reintentar cuando haya Internet.`
	}
	msg := fmt.Sprintf("Ruta IMFOHSA – %s – %s\n%s", name, date, link)
	waBase := "https://wa.me/?text=" + url.QueryEscape(msg)
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	if len(digits) == 8 {
		waBase = "https://wa.me/502" + digits + "?text=" + url.QueryEscape(msg)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, sharePageHTML("📍 Compartir ruta · "+name, status, link, waBase, "", "", mode != "public"))
}

func (s *Server) sharePanelRequestPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.mu.Lock()
	st, _ := s.state["settings"].(map[string]interface{})
	if st == nil {
		st = map[string]interface{}{}
		s.state["settings"] = st
	}
	pt := sval(st["panelRequestPublicToken"])
	if pt == "" {
		pt = token()
		st["panelRequestPublicToken"] = pt
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	base, mode := s.ensurePublicBase(40 * time.Second)
	link := strings.TrimRight(base, "/") + "/index.html?request=panel&token=" + url.QueryEscape(pt)
	status := `<span class="ok"><b>✅ Formulario HTTPS verificado y listo.</b></span>`
	if mode != "public" {
		status = `<span class="warn"><b>⚠ HTTPS no disponible.</b></span> El enlace LAN solo funciona dentro de la misma red.`
	}
	msg := "Solicitud de panel IMFOHSA – completar formulario con mínimo 7 días de anticipación:\n" + link
	wa := "https://wa.me/?text=" + url.QueryEscape(msg)
	mail := "mailto:?subject=" + url.QueryEscape("Solicitud de panel IMFOHSA") + "&body=" + url.QueryEscape(msg)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, sharePageHTML("🚐 Compartir formulario de solicitud de panel", status, link, wa, mail, "", mode != "public"))
}

func (s *Server) restartTunnel(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	force := r.URL.Query().Get("force") == "1"
	if force {
		s.stopTunnel()
	}
	s.mu.RLock()
	running := s.tunnelRunning
	pub := strings.TrimSpace(s.publicURL)
	s.mu.RUnlock()
	if !running && (pub == "" || force) {
		go s.startTunnel()
	}
	jsonOut(w, map[string]interface{}{"ok": true, "started": !running || force, "publicUrl": pub})
}

func (s *Server) captureCreate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q struct {
		Label, Kind, Mode, Owner string
		Min, Max                 int
	}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	if q.Max <= 0 {
		q.Max = 5
	}
	if q.Min < 0 {
		q.Min = 0
	}
	id := token() + token()[:8]
	if q.Mode == "" {
		q.Mode = "photo"
	}
	cs := &CaptureSession{ID: id, Label: q.Label, Kind: q.Kind, Mode: q.Mode, Owner: q.Owner, Min: q.Min, Max: q.Max, CreatedAt: time.Now(), URLs: []string{}}
	s.mu.Lock()
	s.captures[id] = cs
	s.mu.Unlock()
	base, mode := s.ensurePublicBase(40 * time.Second)
	path := "/captura?session=" + url.QueryEscape(id)
	jsonOut(w, map[string]interface{}{"ok": true, "id": id, "token": id, "path": path, "url": strings.TrimRight(base, "/") + path, "mode": mode, "publicReady": mode == "public", "expiresMinutes": 120})
}
func (s *Server) captureStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	s.mu.RLock()
	cs := s.captures[id]
	s.mu.RUnlock()
	if cs == nil {
		http.Error(w, "sesion", 404)
		return
	}
	jsonOut(w, cs)
}
func (s *Server) captureResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q struct {
		ID     string `json:"id"`
		Result string `json:"result"`
	}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	s.mu.Lock()
	cs := s.captures[q.ID]
	if cs == nil || cs.Closed {
		s.mu.Unlock()
		http.Error(w, "sesion invalida", 403)
		return
	}
	cs.Result = strings.TrimSpace(q.Result)
	s.mu.Unlock()
	jsonOut(w, map[string]interface{}{"ok": true, "result": q.Result})
}
func (s *Server) captureFinish(w http.ResponseWriter, r *http.Request) {
	var q struct {
		ID string `json:"id"`
	}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	s.mu.Lock()
	if cs := s.captures[q.ID]; cs != nil {
		cs.Closed = true
	}
	s.mu.Unlock()
	jsonOut(w, map[string]bool{"ok": true})
}
func (s *Server) captureUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		http.Error(w, "archivo", 400)
		return
	}
	id := r.FormValue("session")
	s.mu.RLock()
	cs := s.captures[id]
	s.mu.RUnlock()
	if cs == nil || cs.Closed {
		http.Error(w, "sesion invalida", 403)
		return
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file", 400)
		return
	}
	defer f.Close()
	if h.Size > 12<<20 {
		http.Error(w, "max 12MB", 400)
		return
	}
	ext := strings.ToLower(filepath.Ext(h.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		ext = ".jpg"
	}
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), token()[:8], ext)
	out, err := os.Create(filepath.Join(s.root, "uploads", name))
	if err != nil {
		http.Error(w, "save", 500)
		return
	}
	io.Copy(out, io.LimitReader(f, 12<<20))
	out.Close()
	u := "/uploads/" + name
	s.mu.Lock()
	if c := s.captures[id]; c != nil && len(c.URLs) < c.Max {
		c.URLs = append(c.URLs, u)
	}
	urls := append([]string{}, s.captures[id].URLs...)
	s.mu.Unlock()
	jsonOut(w, map[string]interface{}{"ok": true, "url": u, "urls": urls})
}
func (s *Server) capturePage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("session")
	s.mu.RLock()
	cs := s.captures[id]
	s.mu.RUnlock()
	if cs == nil {
		http.Error(w, "Sesión vencida o inválida", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if cs.Mode == "scan" {
		fmt.Fprintf(w, `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Escáner IMFOHSA</title>
<style>body{font-family:Segoe UI,Arial;background:#071526;color:#fff;margin:0;padding:18px}.box{max-width:620px;margin:auto}.card{background:#0d2036;border:1px solid #274c75;border-radius:16px;padding:18px}video{width:100%%;border-radius:12px;background:#000;max-height:60vh}button,input{width:100%%;box-sizing:border-box;padding:14px;border-radius:10px;font-size:16px;margin-top:10px}button{border:0;background:#2468d8;color:#fff;font-weight:800}.sec{background:#263c56}.muted{color:#afc4db;line-height:1.5}.ok{color:#65e6a5;font-weight:800}</style></head><body><div class="box"><div class="card"><h2>▥ %s</h2><p class="muted">Use la cámara del teléfono para leer QR o código de barras. Si el navegador no permite lectura automática, escriba el código manualmente.</p><video id="v" autoplay playsinline></video><button id="start">📷 Activar cámara</button><input id="manual" placeholder="Código detectado / ingreso manual"><button id="send">Enviar código a la computadora</button><div id="msg" class="muted"></div></div></div>
<script>
const sid=%q,v=document.getElementById('v'),msg=document.getElementById('msg'),manual=document.getElementById('manual');let stream=null,det=null,busy=false;
async function submit(val){val=(val||'').trim();if(!val)return;let r=await fetch('/api/capture/result',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:sid,result:val})});if(r.ok){msg.innerHTML='<span class="ok">✓ Código enviado: '+val+'</span>';manual.value=val;if(stream)stream.getTracks().forEach(t=>t.stop())}else msg.textContent=await r.text()}
async function scanLoop(){if(!det||!stream||busy)return;busy=true;try{let a=await det.detect(v);if(a&&a[0]?.rawValue){await submit(a[0].rawValue);return}}catch(e){}busy=false;setTimeout(scanLoop,450)}
async function start(){try{stream=await navigator.mediaDevices.getUserMedia({video:{facingMode:{ideal:'environment'}},audio:false});v.srcObject=stream;if('BarcodeDetector' in window){det=new BarcodeDetector({formats:['qr_code','code_128','ean_13','ean_8','code_39','upc_a','upc_e']});scanLoop()}else msg.textContent='Lectura automática no disponible. Puede ingresar el código manualmente.'}catch(e){v.style.display='none';msg.textContent='No fue posible abrir la cámara. Ingrese el código manualmente.'}}
document.getElementById('start').onclick=start;document.getElementById('send').onclick=()=>submit(manual.value);start();
</script></body></html>`, html.EscapeString(cs.Label), id)
		return
	}
	fmt.Fprintf(w, `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Cámara IMFOHSA</title><style>body{font-family:Segoe UI,Arial;background:#071526;color:#fff;margin:0;padding:18px}.box{max-width:600px;margin:auto}.card{background:#0d2036;border:1px solid #274c75;border-radius:16px;padding:18px;margin-bottom:12px}video{width:100%%;border-radius:12px;background:#000;max-height:65vh}button{width:100%%;padding:14px;border:0;border-radius:10px;background:#2468d8;color:#fff;font-weight:800;font-size:16px;margin-top:10px}.sec{background:#263c56}.thumbs{display:flex;gap:8px;flex-wrap:wrap;margin-top:10px}.thumbs img{width:82px;height:82px;object-fit:cover;border-radius:8px;border:2px solid #355c88}.muted{color:#afc4db;line-height:1.5}</style></head><body><div class="box"><div class="card"><h2>📷 %s</h2><p class="muted">Tome la fotografía directamente con la cámara. Buena iluminación, enfoque claro y elemento completo dentro del encuadre. Mínimo %d · máximo %d.</p><video id="v" autoplay playsinline></video><canvas id="c" style="display:none"></canvas><button id="shot">📷 Capturar foto</button><button id="fallback" class="sec">Abrir cámara del teléfono</button><button id="sendPhotos" class="sec">📤 Enviar fotos</button><input id="fi" type="file" accept="image/*" capture="environment" style="display:none"><div id="msg" class="muted"></div><div id="thumbs" class="thumbs"></div></div></div><script>
const sid=%q,max=%d,min=%d;let stream=null,urls=[];const v=document.getElementById('v'),c=document.getElementById('c'),fi=document.getElementById('fi'),msg=document.getElementById('msg'),thumbs=document.getElementById('thumbs');
function draw(){thumbs.innerHTML=urls.map(x=>'<img src="'+x+'">').join('');msg.textContent=urls.length+' de '+max+' fotos cargadas en tiempo real.';if(urls.length>=max)document.getElementById('shot').disabled=true}
async function upload(blob){let fd=new FormData();fd.append('session',sid);fd.append('file',blob,'camara.jpg');let r=await fetch('/api/capture/upload',{method:'POST',body:fd});if(!r.ok){msg.textContent=await r.text();return}let j=await r.json();urls=j.urls||[];draw()}
async function start(){try{stream=await navigator.mediaDevices.getUserMedia({video:{facingMode:{ideal:'environment'}},audio:false});v.srcObject=stream}catch(e){v.style.display='none';msg.textContent='Use el botón “Abrir cámara del teléfono”.'}}
document.getElementById('shot').onclick=async()=>{if(!stream)return fi.click();c.width=v.videoWidth;c.height=v.videoHeight;c.getContext('2d').drawImage(v,0,0);c.toBlob(upload,'image/jpeg',.85)};document.getElementById('fallback').onclick=()=>fi.click();document.getElementById('sendPhotos').onclick=async()=>{if(urls.length<min){msg.textContent='Debe cargar al menos '+min+' foto(s) antes de enviar.';return}let r=await fetch('/api/capture/finish',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({id:sid})});if(r.ok){if(stream)stream.getTracks().forEach(t=>t.stop());document.getElementById('shot').disabled=true;document.getElementById('fallback').disabled=true;document.getElementById('sendPhotos').disabled=true;msg.innerHTML='<b style="color:#65e6a5">✓ Fotos enviadas correctamente a Logística.</b>'}else msg.textContent=await r.text()};fi.onchange=()=>{if(fi.files[0])upload(fi.files[0])};start();
</script></body></html>`, html.EscapeString(cs.Label), cs.Min, cs.Max, id, cs.Max, cs.Min)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	q.Username = strings.TrimSpace(q.Username)
	q.Password = strings.TrimSpace(q.Password)
	b, _ := os.ReadFile(s.adminsPath())
	var admins []Admin
	json.Unmarshal(b, &admins)
	ok := false
	name := ""
	role := "jefe_logistica"
	messengerID := ""
	defaultHash := hash("Imfohsa#2026")
	for _, a := range admins {
		passwordOK := a.PasswordHash == hash(q.Password)
		// En cuentas iniciales solamente, aceptar una clave simple de respaldo para evitar errores de teclado con #.
		if a.PasswordHash == defaultHash && q.Password == "IMFOHSA2026" {
			passwordOK = true
		}
		if a.Active && strings.EqualFold(a.Username, q.Username) && passwordOK {
			ok = true
			name = a.Name
			role = normalizeRole(a.Role)
			messengerID = strings.TrimSpace(a.MessengerID)
			break
		}
	}
	if !ok {
		http.Error(w, "Credenciales incorrectas", 401)
		return
	}
	t := token()
	s.mu.Lock()
	s.sessions[t] = q.Username
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "imfohsa_session", Value: t, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600, Secure: r.TLS != nil || os.Getenv("RENDER") == "true"})
	resp := map[string]interface{}{"ok": true, "username": q.Username, "name": name, "role": role}
	if role == "mensajero" {
		if messengerID == "" {
			messengerID, _ = s.messengerAccessForUser(q.Username)
		}
		_, driverURL := s.messengerAccessForUser(q.Username)
		resp["messengerId"] = messengerID
		resp["driverUrl"] = driverURL
	}
	jsonOut(w, resp)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("imfohsa_session"); e == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "imfohsa_session", Value: "", Path: "/", MaxAge: -1})
	jsonOut(w, map[string]bool{"ok": true})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := s.authUser(r)
	if u == "" {
		http.Error(w, "No autorizado", 401)
		return
	}
	b, _ := os.ReadFile(s.adminsPath())
	var admins []Admin
	json.Unmarshal(b, &admins)
	name := u
	role := "jefe_logistica"
	messengerID := ""
	for _, a := range admins {
		if strings.EqualFold(a.Username, u) {
			name = a.Name
			role = normalizeRole(a.Role)
			messengerID = strings.TrimSpace(a.MessengerID)
		}
	}
	resp := map[string]interface{}{"username": u, "name": name, "role": role}
	if role == "mensajero" {
		_, driverURL := s.messengerAccessForUser(u)
		resp["messengerId"] = messengerID
		resp["driverUrl"] = driverURL
	}
	jsonOut(w, resp)
}
func (s *Server) stateHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	_ = user
	if r.Method == "GET" {
		s.mu.RLock()
		defer s.mu.RUnlock()
		role := s.roleForUser(user)
		if managerRole(role) {
			jsonOut(w, s.state)
			return
		}
		// Principio de mínimo acceso: los perfiles de consulta reciben solo
		// los datos necesarios para sus módulos autorizados.
		allowed := map[string]interface{}{}
		allowed["version"] = s.state["version"]
		if role == "asistente_z16" || role == "coordinador_z4" {
			// NLIMA y GOLIVA pueden visualizar el Dashboard completo y ambos centros.
			// Estos datos se entregan para consulta; la validación de escritura por centro
			// permanece en el PUT de /api/state.
			for _, k := range []string{"settings", "orders", "messengers", "clients", "centers", "dailySnapshots", "dailyAvailability", "routeLearning", "systemInsights", "fuel", "vehicles", "panelRequests", "manualCosts", "panelTrips", "dispatchDocuments", "inventoryLedger", "warehouseMovements", "improvementReports"} {
				if v, ok := s.state[k]; ok {
					allowed[k] = v
				}
			}
		} else if role == "creditos_cobros" {
			for _, k := range []string{"settings", "orders", "messengers", "clients", "centers"} {
				if v, ok := s.state[k]; ok {
					allowed[k] = v
				}
			}
		} else if role == "ventas" {
			// Ventas solo recibe los campos necesarios para Consulta vendedores.
			ordersOut := []interface{}{}
			for _, raw := range func() []interface{} { a, _ := s.state["orders"].([]interface{}); return a }() {
				o, _ := raw.(map[string]interface{})
				if o == nil {
					continue
				}
				clean := map[string]interface{}{}
				for _, k := range []string{"id", "orderNo", "date", "clientName", "address", "vendorName", "operationType", "cut", "status", "messengerId", "receivedBy", "photos", "initialEstimatedAt", "latestEstimatedAt"} {
					if v, ok := o[k]; ok {
						clean[k] = v
					}
				}
				ordersOut = append(ordersOut, clean)
			}
			messengersOut := []interface{}{}
			for _, raw := range func() []interface{} { a, _ := s.state["messengers"].([]interface{}); return a }() {
				m, _ := raw.(map[string]interface{})
				if m == nil {
					continue
				}
				messengersOut = append(messengersOut, map[string]interface{}{"id": m["id"], "name": m["name"]})
			}
			allowed["orders"] = ordersOut
			allowed["messengers"] = messengersOut
		}
		jsonOut(w, allowed)
		return
	}
	if r.Method == "PUT" {
		var q map[string]interface{}
		if bodyJSON(r, &q) != nil {
			http.Error(w, "JSON invalido", 400)
			return
		}
		incoming := int64(0)
		if v, ok := q["version"].(float64); ok {
			incoming = int64(v)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if incoming != s.version {
			w.WriteHeader(409)
			jsonOut(w, map[string]interface{}{"error": "version_conflict", "version": s.version})
			return
		}
		role := s.roleForUser(user)
		if !managerRole(role) {
			oldOrders, _ := s.state["orders"].([]interface{})
			newOrders, ok := q["orders"].([]interface{})
			if !ok {
				http.Error(w, "Pedidos requeridos", 400)
				return
			}
			if operationalRole(role) {
				target := operationalCenter(role)
				oldByID := map[string]map[string]interface{}{}
				oldOrderSeq := []string{}
				for _, raw := range oldOrders {
					o := mapOf(raw)
					if o != nil {
						id := fmt.Sprint(o["id"])
						oldByID[id] = o
						oldOrderSeq = append(oldOrderSeq, id)
					}
				}
				incomingByID := map[string]map[string]interface{}{}
				incomingSeq := []string{}
				for _, raw := range newOrders {
					n := mapOf(raw)
					if n == nil {
						continue
					}
					id := fmt.Sprint(n["id"])
					incomingByID[id] = n
					incomingSeq = append(incomingSeq, id)
				}
				merged := []interface{}{}
				seen := map[string]bool{}
				for _, id := range incomingSeq {
					n := incomingByID[id]
					o := oldByID[id]
					if o == nil {
						if orderCenter(n) == target {
							merged = append(merged, n)
							seen[id] = true
						}
						continue
					}
					if orderCenter(o) == target {
						n["centerId"] = orderCenter(o)
						merged = append(merged, n)
					} else {
						merged = append(merged, o)
					}
					seen[id] = true
				}
				for _, id := range oldOrderSeq {
					if seen[id] {
						continue
					}
					if o := oldByID[id]; o != nil {
						merged = append(merged, o)
					}
				}
				s.state["orders"] = merged
				if c, ok := q["clients"]; ok {
					s.state["clients"] = c
				}
				if rl, ok := q["routeLearning"]; ok {
					s.state["routeLearning"] = rl
				}
				// La disponibilidad diaria sí es una acción operativa del centro.
				// Se combina por mensajero para impedir que Zona 16 modifique Zona 4 y viceversa.
				if incomingAV, ok := q["dailyAvailability"].(map[string]interface{}); ok {
					allowedMids := map[string]bool{}
					if ms, ok := s.state["messengers"].([]interface{}); ok {
						for _, rawM := range ms {
							m := mapOf(rawM)
							if m != nil && strings.TrimSpace(fmt.Sprint(m["centerId"])) == target {
								allowedMids[fmt.Sprint(m["id"])] = true
							}
						}
					}
					currentAV, _ := s.state["dailyAvailability"].(map[string]interface{})
					if currentAV == nil {
						currentAV = map[string]interface{}{}
					}
					for date, rawDay := range incomingAV {
						inDay, _ := rawDay.(map[string]interface{})
						if inDay == nil {
							continue
						}
						outDay, _ := currentAV[date].(map[string]interface{})
						if outDay == nil {
							outDay = map[string]interface{}{}
						}
						for mid, rawStatus := range inDay {
							if !allowedMids[mid] {
								continue
							}
							status := strings.TrimSpace(fmt.Sprint(rawStatus))
							if status != "Disponible" && status != "Ausente" && status != "Panel" {
								continue
							}
							outDay[mid] = status
						}
						currentAV[date] = outDay
					}
					s.state["dailyAvailability"] = currentAV
				}
			} else if normalizeRole(role) == "creditos_cobros" {
				ids := map[string]bool{}
				for _, raw := range oldOrders {
					o := mapOf(raw)
					if o != nil {
						ids[fmt.Sprint(o["id"])] = true
					}
				}
				merged := append([]interface{}{}, oldOrders...)
				for _, raw := range newOrders {
					n := mapOf(raw)
					if n == nil {
						continue
					}
					id := fmt.Sprint(n["id"])
					if ids[id] {
						continue
					}
					if strings.ToLower(strings.TrimSpace(fmt.Sprint(n["operationType"]))) != "cobro" {
						http.Error(w, "Este perfil solo puede crear nuevos cobros", 403)
						return
					}
					if strings.TrimSpace(fmt.Sprint(n["messengerId"])) != "" {
						http.Error(w, "El cobro nuevo debe iniciar sin asignar", 403)
						return
					}
					n["status"] = "Pendiente"
					merged = append(merged, n)
				}
				s.state["orders"] = merged
				if c, ok := q["clients"]; ok {
					s.state["clients"] = c
				}
			} else {
				http.Error(w, "Solo lectura para este usuario", 403)
				return
			}
			s.state["lastModifiedBy"] = user
			s.state["lastModifiedAt"] = time.Now().Format(time.RFC3339)
			if err := s.saveLocked(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			jsonOut(w, map[string]interface{}{"ok": true, "version": s.version})
			return
		}
		s.state = q
		s.state["lastModifiedBy"] = user
		s.state["lastModifiedAt"] = time.Now().Format(time.RFC3339)
		if err := s.saveLocked(); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonOut(w, map[string]interface{}{"ok": true, "version": s.version})
		return
	}
	http.Error(w, "method", 405)
}

func (s *Server) creditStatus(w http.ResponseWriter, r *http.Request) {
	user := s.authUser(r)
	if user == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}
	role := s.roleForUser(user)
	if !managerRole(role) && normalizeRole(role) != "creditos_cobros" {
		http.Error(w, "Permiso insuficiente", 403)
		return
	}
	if r.Method != "PUT" {
		http.Error(w, "method", 405)
		return
	}
	var q map[string]interface{}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "JSON invalido", 400)
		return
	}
	id := strings.TrimSpace(fmt.Sprint(q["orderId"]))
	if id == "" {
		http.Error(w, "orderId requerido", 400)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arr, _ := s.state["orders"].([]interface{})
	var target map[string]interface{}
	for _, raw := range arr {
		o := mapOf(raw)
		if o != nil && fmt.Sprint(o["id"]) == id {
			target = o
			break
		}
	}
	if target == nil {
		http.Error(w, "Pedido no encontrado", 404)
		return
	}
	old := map[string]interface{}{"paymentTerm": target["paymentTerm"], "paymentStatus": target["paymentStatus"], "paymentMethod": target["paymentMethod"], "paymentReference": target["paymentReference"], "paymentAction": target["paymentAction"]}
	for _, k := range []string{"paymentTerm", "paymentStatus", "paymentMethod", "paymentReference", "paymentAction"} {
		if v, exists := q[k]; exists {
			target[k] = v
		}
	}
	now := time.Now().Format(time.RFC3339)
	target["paymentValidatedAt"] = now
	target["paymentValidatedBy"] = user
	event := map[string]interface{}{"at": now, "user": user, "old": old, "paymentTerm": target["paymentTerm"], "paymentStatus": target["paymentStatus"], "paymentMethod": target["paymentMethod"], "paymentReference": target["paymentReference"], "paymentAction": target["paymentAction"]}
	if hist, ok := target["paymentHistory"].([]interface{}); ok {
		target["paymentHistory"] = append(hist, event)
	} else {
		target["paymentHistory"] = []interface{}{event}
	}
	s.state["lastModifiedBy"] = user
	s.state["lastModifiedAt"] = now
	if err := s.saveLocked(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	jsonOut(w, map[string]interface{}{"ok": true, "version": s.version})
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan int64, 4)
	s.mu.Lock()
	s.eventClients[ch] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.eventClients, ch); s.mu.Unlock() }()
	fmt.Fprintf(w, "data: %d\n\n", s.version)
	fl.Flush()
	for {
		select {
		case v := <-ch:
			fmt.Fprintf(w, "data: %d\n\n", v)
			fl.Flush()
		case <-r.Context().Done():
			return
		case <-time.After(25 * time.Second):
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		}
	}
}
func (s *Server) adminsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireManager(w, r); !ok {
		return
	}
	if r.Method == "GET" {
		b, _ := os.ReadFile(s.adminsPath())
		var admins []Admin
		json.Unmarshal(b, &admins)
		out := []map[string]interface{}{}
		for _, a := range admins {
			out = append(out, map[string]interface{}{"username": a.Username, "name": a.Name, "active": a.Active, "role": normalizeRole(a.Role), "messengerId": a.MessengerID})
		}
		jsonOut(w, out)
		return
	}
	if r.Method == "PUT" {
		var q []struct {
			Username, Name, Password, Role, MessengerID string
			Active                                      bool
		}
		if bodyJSON(r, &q) != nil || len(q) > 20 {
			http.Error(w, "Maximo 20 usuarios", 400)
			return
		}
		admins := []Admin{}
		for _, a := range q {
			if strings.TrimSpace(a.Username) == "" {
				continue
			}
			ph := ""
			oldRole := ""
			oldMessengerID := ""
			b, _ := os.ReadFile(s.adminsPath())
			var old []Admin
			json.Unmarshal(b, &old)
			for _, o := range old {
				if o.Username == a.Username {
					ph = o.PasswordHash
					oldRole = o.Role
					oldMessengerID = o.MessengerID
				}
			}
			if a.Password != "" {
				ph = hash(a.Password)
			}
			if ph == "" {
				ph = hash("Imfohsa#2026")
			}
			role := strings.TrimSpace(a.Role)
			if role == "" {
				role = oldRole
			}
			mid := strings.TrimSpace(a.MessengerID)
			if mid == "" {
				mid = oldMessengerID
			}
			admins = append(admins, Admin{Username: a.Username, Name: a.Name, PasswordHash: ph, Active: a.Active, Role: normalizeRole(role), MessengerID: mid})
		}
		managers := 0
		for _, a := range admins {
			if managerRole(a.Role) && a.Active {
				managers++
			}
		}
		if managers > 4 {
			http.Error(w, "Maximo 4 usuarios con permisos administrativos", 400)
			return
		}
		bb, _ := json.MarshalIndent(admins, "", "  ")
		os.WriteFile(s.adminsPath(), bb, 0600)
		jsonOut(w, map[string]bool{"ok": true})
		return
	}
	http.Error(w, "method", 405)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" || p == "index.html" {
		b, err := os.ReadFile(filepath.Join(s.root, "index.html"))
		if err != nil {
			http.Error(w, "index", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Write(b)
		return
	}
	http.NotFound(w, r)
}
func (s *Server) uploadsServe(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Join(s.root, "uploads")))).ServeHTTP(w, r)
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		http.Error(w, "archivo", 400)
		return
	}
	allowed := s.authUser(r) != ""
	if !allowed {
		tokenQ := r.FormValue("token")
		id := r.FormValue("driver")
		panel := r.FormValue("panel")
		date := r.FormValue("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		if id != "" {
			allowed = s.validateDriverAccess(w, r, id, tokenQ, date, false)
		} else {
			allowed = s.validatePublicToken("", panel, tokenQ)
		}
	}
	if !allowed {
		http.Error(w, "No autorizado", 401)
		return
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file", 400)
		return
	}
	defer f.Close()
	if h.Size > 12<<20 {
		http.Error(w, "max 12MB", 400)
		return
	}
	ext := strings.ToLower(filepath.Ext(h.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		ext = ".jpg"
	}
	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), token()[:8], ext)
	out, err := os.Create(filepath.Join(s.root, "uploads", name))
	if err != nil {
		http.Error(w, "save", 500)
		return
	}
	defer out.Close()
	io.Copy(out, io.LimitReader(f, 12<<20))
	jsonOut(w, map[string]interface{}{"ok": true, "url": "/uploads/" + name})
}

func (s *Server) geocodeProxy(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "Dirección requerida", 400)
		return
	}
	if !strings.Contains(strings.ToLower(q), "guatemala") {
		q += ", Guatemala"
	}

	// R7.7 PILOTO: Google primero cuando hay clave configurada. Si la Demo Key
	// no habilita Geocoding o Google falla, la operación continúa con Nominatim.
	s.mu.RLock()
	settings, _ := s.state["settings"].(map[string]interface{})
	key := strings.TrimSpace(fmt.Sprint(settings["googleMapsApiKey"]))
	s.mu.RUnlock()
	if key != "" {
		u := "https://maps.googleapis.com/maps/api/geocode/json?region=gt&language=es&address=" + url.QueryEscape(q) + "&key=" + url.QueryEscape(key)
		cli := &http.Client{Timeout: 12 * time.Second}
		resp, err := cli.Get(u)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var gr struct {
					Status  string `json:"status"`
					Results []struct {
						FormattedAddress string `json:"formatted_address"`
						Geometry         struct {
							Location struct {
								Lat float64 `json:"lat"`
								Lng float64 `json:"lng"`
							} `json:"location"`
							LocationType string `json:"location_type"`
						} `json:"geometry"`
					} `json:"results"`
				}
				if json.NewDecoder(resp.Body).Decode(&gr) == nil && gr.Status == "OK" && len(gr.Results) > 0 {
					x := gr.Results[0]
					jsonOut(w, map[string]interface{}{"ok": true, "found": true, "lat": x.Geometry.Location.Lat, "lng": x.Geometry.Location.Lng, "displayName": x.FormattedAddress, "source": "google", "locationType": x.Geometry.LocationType})
					return
				}
			}
		}
	}

	u := "https://nominatim.openstreetmap.org/search?format=json&limit=1&countrycodes=gt&q=" + url.QueryEscape(q)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "IMFOHSA-Logistica/7.7-piloto (geocoding administrativo)")
	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		http.Error(w, "No se pudo geocodificar la dirección", 502)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "Servicio de geocodificación no disponible", 502)
		return
	}
	var rows []map[string]interface{}
	if json.NewDecoder(resp.Body).Decode(&rows) != nil || len(rows) == 0 {
		jsonOut(w, map[string]interface{}{"ok": false, "found": false, "source": "nominatim"})
		return
	}
	lat, _ := strconv.ParseFloat(fmt.Sprint(rows[0]["lat"]), 64)
	lng, _ := strconv.ParseFloat(fmt.Sprint(rows[0]["lon"]), 64)
	jsonOut(w, map[string]interface{}{"ok": true, "found": true, "lat": lat, "lng": lng, "displayName": rows[0]["display_name"], "source": "nominatim-fallback"})
}

func (s *Server) routeProxy(w http.ResponseWriter, r *http.Request) {
	coords := r.URL.Query().Get("coords")
	if !regexp.MustCompile(`^-?\d+(\.\d+)?,-?\d+(\.\d+)?(;(-?\d+(\.\d+)?,-?\d+(\.\d+)?))*$`).MatchString(coords) {
		http.Error(w, "coords", 400)
		return
	}
	url := "https://router.project-osrm.org/route/v1/driving/" + coords + "?overview=full&geometries=geojson&steps=false"
	proxyJSON(w, url, nil)
}
func (s *Server) tableProxy(w http.ResponseWriter, r *http.Request) {
	coords := r.URL.Query().Get("coords")
	if !regexp.MustCompile(`^-?\d+(\.\d+)?,-?\d+(\.\d+)?(;(-?\d+(\.\d+)?,-?\d+(\.\d+)?))*$`).MatchString(coords) {
		http.Error(w, "coords", 400)
		return
	}
	// R8: allow block matrices so 100+ orders can be evaluated by road distance
	// before assigning them to a messenger. Sources/destinations are OSRM indexes.
	idxRe := regexp.MustCompile(`^\d+(;\d+)*$`)
	values := url.Values{}
	values.Set("annotations", "distance,duration")
	if src := strings.TrimSpace(r.URL.Query().Get("sources")); src != "" {
		if !idxRe.MatchString(src) {
			http.Error(w, "sources", 400)
			return
		}
		values.Set("sources", src)
	}
	if dst := strings.TrimSpace(r.URL.Query().Get("destinations")); dst != "" {
		if !idxRe.MatchString(dst) {
			http.Error(w, "destinations", 400)
			return
		}
		values.Set("destinations", dst)
	}
	u := "https://router.project-osrm.org/table/v1/driving/" + coords + "?" + values.Encode()
	proxyJSON(w, u, nil)
}

func decodeGooglePolyline(encoded string) [][]float64 {
	coords := [][]float64{}
	lat, lng, index := 0, 0, 0
	for index < len(encoded) {
		result, shift := 0, 0
		for {
			if index >= len(encoded) {
				return coords
			}
			b := int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlat := result >> 1
		if result&1 != 0 {
			dlat = ^dlat
		}
		lat += dlat
		result, shift = 0, 0
		for {
			if index >= len(encoded) {
				return coords
			}
			b := int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlng := result >> 1
		if result&1 != 0 {
			dlng = ^dlng
		}
		lng += dlng
		// GeoJSON uses [longitude, latitude].
		coords = append(coords, []float64{float64(lng) / 1e5, float64(lat) / 1e5})
	}
	return coords
}

func durationSeconds(v string) float64 {
	v = strings.TrimSpace(strings.TrimSuffix(v, "s"))
	n, _ := strconv.ParseFloat(v, 64)
	return n
}

// R8 fallback: if Google Routes is missing, misconfigured or temporarily fails,
// the planner still returns a road route using OSRM and clearly labels it as an estimate.
func writeOSRMTrafficFallback(w http.ResponseWriter, points [][]float64, factor float64) bool {
	coords := []string{}
	for _, p := range points {
		coords = append(coords, fmt.Sprintf("%f,%f", p[1], p[0]))
	}
	u := "https://router.project-osrm.org/route/v1/driving/" + strings.Join(coords, ";") + "?overview=full&geometries=geojson&steps=false"
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Get(u)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	var x map[string]interface{}
	if json.NewDecoder(resp.Body).Decode(&x) != nil {
		return false
	}
	if rs, ok := x["routes"].([]interface{}); ok && len(rs) > 0 {
		if rr, ok := rs[0].(map[string]interface{}); ok {
			if d, ok := rr["duration"].(float64); ok {
				rr["durationTraffic"] = d * factor
			}
			if m, ok := rr["distance"].(float64); ok {
				rr["distanceMeters"] = m
			}
			rr["trafficSource"] = "estimated-factor"
			rr["optimizedIntermediateWaypointIndex"] = []int{}
		}
	}
	jsonOut(w, x)
	return true
}

func (s *Server) trafficRoute(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Points            [][]float64 `json:"points"`
		Optimize          bool        `json:"optimize"`
		DepartureTime     string      `json:"departureTime"`
		RoutingPreference string      `json:"routingPreference"`
	}
	if bodyJSON(r, &q) != nil || len(q.Points) < 2 {
		http.Error(w, "points", 400)
		return
	}
	for _, p := range q.Points {
		if len(p) < 2 || p[0] < -90 || p[0] > 90 || p[1] < -180 || p[1] > 180 {
			http.Error(w, "invalid point", 400)
			return
		}
	}

	s.mu.RLock()
	settings, _ := s.state["settings"].(map[string]interface{})
	key, _ := settings["googleMapsApiKey"].(string)
	factor := 1.25
	if v, ok := settings["trafficFactor"].(float64); ok && v > 0 {
		factor = v
	}
	s.mu.RUnlock()

	// Without a Google key the application remains functional using OSRM plus
	// the configured traffic factor. The source is explicitly marked as estimated.
	if strings.TrimSpace(key) == "" {
		if writeOSRMTrafficFallback(w, q.Points, factor) {
			return
		}
		http.Error(w, "OSRM routing unavailable", 502)
		return
	}

	origin := q.Points[0]
	dest := q.Points[len(q.Points)-1]
	inter := []map[string]interface{}{}
	for _, p := range q.Points[1 : len(q.Points)-1] {
		inter = append(inter, map[string]interface{}{"location": map[string]interface{}{"latLng": map[string]float64{"latitude": p[0], "longitude": p[1]}}})
	}
	pref := strings.TrimSpace(q.RoutingPreference)
	if pref != "TRAFFIC_AWARE" && pref != "TRAFFIC_AWARE_OPTIMAL" {
		pref = "TRAFFIC_AWARE"
	}
	// Google waypoint optimization cannot be combined with TRAFFIC_AWARE_OPTIMAL.
	if q.Optimize && pref == "TRAFFIC_AWARE_OPTIMAL" {
		pref = "TRAFFIC_AWARE"
	}
	depart := time.Now().Add(2 * time.Minute).UTC()
	if q.DepartureTime != "" {
		if t, err := time.Parse(time.RFC3339, q.DepartureTime); err == nil && t.After(time.Now()) {
			depart = t.UTC()
		}
	}
	body := map[string]interface{}{
		"origin":            map[string]interface{}{"location": map[string]interface{}{"latLng": map[string]float64{"latitude": origin[0], "longitude": origin[1]}}},
		"destination":       map[string]interface{}{"location": map[string]interface{}{"latLng": map[string]float64{"latitude": dest[0], "longitude": dest[1]}}},
		"intermediates":     inter,
		"travelMode":        "DRIVE",
		"routingPreference": pref,
		"departureTime":     depart.Format(time.RFC3339),
	}
	if q.Optimize && len(inter) > 1 {
		body["optimizeWaypointOrder"] = true
	}
	bb, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://routes.googleapis.com/directions/v2:computeRoutes", strings.NewReader(string(bb)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", strings.TrimSpace(key))
	req.Header.Set("X-Goog-FieldMask", "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline,routes.optimizedIntermediateWaypointIndex")
	cli := &http.Client{Timeout: 30 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		if writeOSRMTrafficFallback(w, q.Points, factor) {
			return
		}
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	var gr struct {
		Routes []struct {
			Duration       string  `json:"duration"`
			DistanceMeters float64 `json:"distanceMeters"`
			Polyline       struct {
				EncodedPolyline string `json:"encodedPolyline"`
			} `json:"polyline"`
			Optimized []int `json:"optimizedIntermediateWaypointIndex"`
		} `json:"routes"`
		Error interface{} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		if writeOSRMTrafficFallback(w, q.Points, factor) {
			return
		}
		http.Error(w, "invalid Google Routes response", 502)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || len(gr.Routes) == 0 {
		// Invalid quota/key/network must not freeze logistics. Fall back to OSRM.
		if writeOSRMTrafficFallback(w, q.Points, factor) {
			return
		}
		msg := fmt.Sprintf("Google Routes error %d", resp.StatusCode)
		if gr.Error != nil {
			msg += ": " + fmt.Sprint(gr.Error)
		}
		http.Error(w, msg, 502)
		return
	}
	rr := gr.Routes[0]
	sec := durationSeconds(rr.Duration)
	geometry := map[string]interface{}{"type": "LineString", "coordinates": decodeGooglePolyline(rr.Polyline.EncodedPolyline)}
	jsonOut(w, map[string]interface{}{
		"code": "Ok",
		"routes": []map[string]interface{}{{
			"distance":                           rr.DistanceMeters,
			"distanceMeters":                     rr.DistanceMeters,
			"duration":                           sec,
			"durationTraffic":                    sec,
			"geometry":                           geometry,
			"optimizedIntermediateWaypointIndex": rr.Optimized,
			"trafficSource":                      "google-live",
			"routingPreference":                  pref,
		}},
	})
}

func proxyJSON(w http.ResponseWriter, url string, headers map[string]string) {
	cli := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := cli.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) validatePublicToken(driver, panel, t string) bool {
	if t == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if driver != "" {
		if arr, ok := s.state["messengers"].([]interface{}); ok {
			for _, v := range arr {
				m, _ := v.(map[string]interface{})
				if fmt.Sprint(m["id"]) == driver && fmt.Sprint(m["token"]) == t {
					return true
				}
			}
		}
	}
	if panel != "" {
		if arr, ok := s.state["panelAssignments"].([]interface{}); ok {
			for _, v := range arr {
				m, _ := v.(map[string]interface{})
				if fmt.Sprint(m["id"]) == panel && fmt.Sprint(m["token"]) == t {
					return true
				}
			}
		}
	}
	return false
}
func (s *Server) publicDriver(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	t := r.URL.Query().Get("token")
	routeDate := r.URL.Query().Get("date")
	if routeDate == "" {
		routeDate = time.Now().Format("2006-01-02")
	}
	if !s.validateDriverAccess(w, r, id, t, routeDate, true) {
		http.Error(w, "Enlace invalido, vencido o asignado a otro dispositivo", 403)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var messenger map[string]interface{}
	for _, v := range s.state["messengers"].([]interface{}) {
		m, _ := v.(map[string]interface{})
		if fmt.Sprint(m["id"]) == id {
			messenger = m
		}
	}
	orders := []interface{}{}
	published := messenger != nil && sval(messenger["routePublishedDate"]) == routeDate
	preview := r.URL.Query().Get("preview") == "1"
	if published || preview {
		for _, v := range s.state["orders"].([]interface{}) {
			o, _ := v.(map[string]interface{})
			if fmt.Sprint(o["messengerId"]) == id && fmt.Sprint(o["date"]) == routeDate {
				orders = append(orders, o)
			}
		}
	}
	jsonOut(w, map[string]interface{}{"messenger": messenger, "orders": orders, "routeDate": routeDate, "settings": s.state["settings"], "centers": s.state["centers"], "version": s.version})
}
func (s *Server) publicDriverAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q map[string]interface{}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	id := fmt.Sprint(q["driver"])
	t := fmt.Sprint(q["token"])
	action := fmt.Sprint(q["action"])
	orderId := fmt.Sprint(q["orderId"])
	routeDate := fmt.Sprint(q["date"])
	if routeDate == "" {
		routeDate = time.Now().Format("2006-01-02")
	}
	if !s.validateDriverAccess(w, r, id, t, routeDate, false) {
		http.Error(w, "Enlace invalido, vencido o asignado a otro dispositivo", 403)
		return
	}
	if routeDate != time.Now().Format("2006-01-02") {
		http.Error(w, "La ruta no corresponde al dia actual", 409)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// R7.7.4: la asignación es borrador hasta que Logística entrega/publica la ruta.
	published := false
	if marr, ok := s.state["messengers"].([]interface{}); ok {
		for _, mv := range marr {
			mm, _ := mv.(map[string]interface{})
			if fmt.Sprint(mm["id"]) == id && sval(mm["routePublishedDate"]) == routeDate {
				published = true
				break
			}
		}
	}
	if !published {
		http.Error(w, "La ruta aun no ha sido enviada oficialmente por Logistica", 409)
		return
	}
	arr, _ := s.state["orders"].([]interface{})

	// Las gestiones se deben cerrar en orden. Esto evita saltarse una parada y
	// después olvidar cerrar la gestión anterior. Los estados finalizados o
	// anulados no forman parte de la secuencia activa.
	var target map[string]interface{}
	var firstOpen map[string]interface{}
	firstOpenPos := math.MaxFloat64
	for _, v := range arr {
		o, _ := v.(map[string]interface{})
		if fmt.Sprint(o["messengerId"]) != id || fmt.Sprint(o["date"]) != routeDate {
			continue
		}
		if fmt.Sprint(o["id"]) == orderId {
			target = o
		}
		status := strings.TrimSpace(fmt.Sprint(o["status"]))
		if status != "Entregado" && status != "No entregado" && status != "Anulado" {
			pos := numberValue(o["routePosition"])
			if pos <= 0 {
				pos = 999999
			}
			if firstOpen == nil || pos < firstOpenPos {
				firstOpen = o
				firstOpenPos = pos
			}
		}
	}

	if action == "arrived" || action == "startService" || action == "delivered" || action == "failed" {
		if target == nil {
			http.Error(w, "Gestión no encontrada en la ruta activa", 404)
			return
		}
		if firstOpen != nil && fmt.Sprint(firstOpen["id"]) != orderId {
			http.Error(w, "Debe cerrar la gestión anterior antes de continuar con la siguiente", 409)
			return
		}
		status := strings.TrimSpace(fmt.Sprint(target["status"]))
		if status == "Entregado" || status == "No entregado" || status == "Anulado" {
			http.Error(w, "Esta gestión ya fue cerrada y está bloqueada", 409)
			return
		}
		switch action {
		case "arrived":
			if status == "En cliente" {
				http.Error(w, "La llegada ya fue registrada; el temporizador está corriendo", 409)
				return
			}
			now := time.Now().Format(time.RFC3339)
			target["arrivedAt"] = now
			target["status"] = "En cliente"
			target["lockedSequence"] = true
			visit := map[string]interface{}{"at": now, "messengerId": id, "cut": target["cut"], "address": target["address"], "result": "En cliente"}
			if old, ok := target["visitHistory"].([]interface{}); ok {
				target["visitHistory"] = append(old, visit)
			} else {
				target["visitHistory"] = []interface{}{visit}
			}
		case "startService":
			if status != "En cliente" {
				http.Error(w, "Primero debe marcar Llegué antes de iniciar atención", 409)
				return
			}
			if strings.TrimSpace(fmt.Sprint(target["serviceStartedAt"])) == "" {
				target["serviceStartedAt"] = time.Now().Format(time.RFC3339)
			}
			target["serviceStatus"] = "Atendiendo"
		case "delivered":
			if status != "En cliente" {
				http.Error(w, "Primero debe marcar Llegué antes de cerrar la entrega", 409)
				return
			}
			failedAttempts := 0
			if old, ok := target["attemptHistory"].([]interface{}); ok {
				failedAttempts = len(old)
			}
			target["deliveryAttemptNo"] = failedAttempts + 1
			target["visitCount"] = failedAttempts + 1
			target["status"] = "Entregado"
			target["resultStatus"] = "Entregado"
			target["deliveredAt"] = time.Now().Format(time.RFC3339)
			target["closedAt"] = time.Now().Format(time.RFC3339)
			target["locked"] = true
			if vh, ok := target["visitHistory"].([]interface{}); ok && len(vh) > 0 {
				if last, ok := vh[len(vh)-1].(map[string]interface{}); ok {
					last["result"] = "Entregado"
					last["closedAt"] = target["closedAt"]
				}
			}
			for k, v := range q {
				if k != "token" {
					target[k] = v
				}
			}
		case "failed":
			if status != "En cliente" {
				http.Error(w, "Primero debe marcar Llegué antes de registrar No entregado", 409)
				return
			}
			now := time.Now().Format(time.RFC3339)
			reason := strings.TrimSpace(fmt.Sprint(q["reason"]))
			notes := strings.TrimSpace(fmt.Sprint(q["notes"]))
			resolution := strings.ToLower(strings.TrimSpace(fmt.Sprint(q["resolution"])))
			responsible := strings.TrimSpace(fmt.Sprint(q["responsible"]))
			causeType := strings.TrimSpace(fmt.Sprint(q["causeType"]))
			if responsible == "" {
				responsible = "Cliente"
			}
			if causeType == "" {
				causeType = "Externo"
			}
			target["failedAt"] = now
			target["lastAttemptStatus"] = "No entregado"
			target["resultStatus"] = "No entregado"
			target["failureReason"] = reason
			target["failureNotes"] = notes
			target["failureResponsible"] = responsible
			target["failureCauseType"] = causeType
			target["lastMessengerId"] = id
			target["lastArrivedAt"] = target["arrivedAt"]
			target["closedAt"] = now
			target["locked"] = true
			if vh, ok := target["visitHistory"].([]interface{}); ok && len(vh) > 0 {
				if last, ok := vh[len(vh)-1].(map[string]interface{}); ok {
					last["result"] = "No entregado"
					last["closedAt"] = now
					last["reason"] = reason
					last["responsible"] = responsible
					last["causeType"] = causeType
				}
			}
			attemptNo := 1
			attempt := map[string]interface{}{"at": now, "messengerId": id, "result": "No entregado", "reason": reason, "notes": notes, "responsible": responsible, "causeType": causeType}
			if old, ok := target["attemptHistory"].([]interface{}); ok {
				attemptNo = len(old) + 1
				attempt["attemptNo"] = attemptNo
				target["attemptHistory"] = append(old, attempt)
			} else {
				attempt["attemptNo"] = attemptNo
				target["attemptHistory"] = []interface{}{attempt}
			}
			target["attemptCount"] = attemptNo
			target["visitCount"] = attemptNo
			delete(target, "arrivedAt")
			delete(target, "serviceStartedAt")
			cancel := resolution == "anular" || strings.Contains(strings.ToLower(notes), "anular") || attemptNo >= 3
			needsCorrection := resolution == "corregir" || resolution == "pendiente_correccion"
			if cancel {
				target["status"] = "Anulado"
				if attemptNo >= 3 {
					target["resolution"] = "Anulado por 3 intentos no efectivos"
				} else {
					target["resolution"] = "Anulado"
				}
				target["plannerExcluded"] = true
				target["needsReassignment"] = false
				target["messengerId"] = ""
			} else if needsCorrection {
				target["status"] = "Pendiente corrección"
				target["resolution"] = "Pendiente de corrección"
				target["plannerExcluded"] = true
				target["needsReassignment"] = false
				target["messengerId"] = ""
			} else {
				target["status"] = "Pendiente"
				target["resolution"] = fmt.Sprintf("Reintento %d", attemptNo+1)
				target["plannerExcluded"] = false
				target["needsReassignment"] = true
				target["messengerId"] = ""
			}
		}
	}

	// R7.5.16: después de cerrar cada gestión, conservar plan vs ejecución y recalcular ETA de las restantes.
	if action == "delivered" || action == "failed" {
		now := time.Now()
		mins := 0.0
		lastLat, lastLng := numberValue(target["lat"]), numberValue(target["lng"])
		for _, rv := range arr {
			ro, _ := rv.(map[string]interface{})
			if ro == nil || fmt.Sprint(ro["messengerId"]) != id || fmt.Sprint(ro["date"]) != routeDate {
				continue
			}
			st := strings.TrimSpace(fmt.Sprint(ro["status"]))
			if st == "Entregado" || st == "No entregado" || st == "Anulado" {
				continue
			}
			lat, lng := numberValue(ro["lat"]), numberValue(ro["lng"])
			if lastLat != 0 && lastLng != 0 && lat != 0 && lng != 0 {
				mins += geoKm(lastLat, lastLng, lat, lng) / 25.0 * 60.0
			}
			mins += 15
			eta := now.Add(time.Duration(mins * float64(time.Minute))).Format("15:04")
			ro["latestEstimatedAt"] = eta
			ro["lastRecalculatedAt"] = now.Format(time.RFC3339)
			lastLat, lastLng = lat, lng
		}
		if hist, ok := s.state["routeLearning"].([]interface{}); ok {
			s.state["routeLearning"] = append(hist, map[string]interface{}{"type": "recalculo", "date": routeDate, "messengerId": id, "orderId": orderId, "at": now.Format(time.RFC3339)})
		} else {
			s.state["routeLearning"] = []interface{}{map[string]interface{}{"type": "recalculo", "date": routeDate, "messengerId": id, "orderId": orderId, "at": now.Format(time.RFC3339)}}
		}
	}

	if action == "startRoute" || action == "returnCedi" || action == "closeRoute" {
		for _, v := range s.state["messengers"].([]interface{}) {
			m, _ := v.(map[string]interface{})
			if fmt.Sprint(m["id"]) == id {
				if action == "startRoute" {
					if q["photo"] == nil || strings.TrimSpace(fmt.Sprint(q["photo"])) == "" || numberValue(q["odometer"]) <= 0 {
						http.Error(w, "Foto y lectura inicial del odometro son obligatorias", 400)
						return
					}
					now := time.Now()
					m["routeStartedAt"] = now.Format(time.RFC3339)
					m["routeStatus"] = "En ruta"
					m["odometerStart"] = q["odometer"]
					m["odometerStartPhoto"] = q["photo"]
					if v := strings.TrimSpace(fmt.Sprint(q["delayReason"])); v != "" {
						m["routeStartDelayReason"] = v
					}
					if v := strings.TrimSpace(fmt.Sprint(q["delayResponsible"])); v != "" {
						m["routeStartDelayResponsible"] = v
					}
					if v := strings.TrimSpace(fmt.Sprint(q["daypart"])); v != "" {
						m["activeDaypart"] = v
					}
					grace := 10.0
					if st, ok := s.state["settings"].(map[string]interface{}); ok {
						if g, ok := st["routeDispatchGraceMinutes"].(float64); ok {
							grace = g
						}
					}
					if at := fmt.Sprint(m["routeAssignedAt"]); at != "" {
						if t0, err := time.Parse(time.RFC3339, at); err == nil {
							delay := now.Sub(t0.Add(time.Duration(grace) * time.Minute)).Minutes()
							if delay < 0 {
								delay = 0
							}
							m["routeStartDelayMinutes"] = math.Round(delay)
						}
					}
				}
				if action == "returnCedi" {
					m["routeStatus"] = "Retorno CEDI"
					m["returnCediAt"] = time.Now().Format(time.RFC3339)
				}
				if action == "closeRoute" {
					// Cierre por AM/PM cuando el cliente móvil informa daypart; evita que una ruta PM ya asignada bloquee el cierre AM.
					daypart := strings.ToUpper(strings.TrimSpace(fmt.Sprint(q["daypart"])))
					for _, ov := range arr {
						o, _ := ov.(map[string]interface{})
						if fmt.Sprint(o["messengerId"]) != id || fmt.Sprint(o["date"]) != routeDate {
							continue
						}
						if daypart != "" {
							cut := strings.TrimSpace(fmt.Sprint(o["cut"]))
							op := "PM"
							if cut == "08:00" || cut == "09:30" {
								op = "AM"
							}
							if op != daypart {
								continue
							}
						}
						st := strings.TrimSpace(fmt.Sprint(o["status"]))
						if st != "Entregado" && st != "No entregado" && st != "Anulado" {
							http.Error(w, "No puede cerrar la ruta: todavía hay gestiones pendientes", 409)
							return
						}
					}
					if q["photo"] == nil || strings.TrimSpace(fmt.Sprint(q["photo"])) == "" || numberValue(q["odometer"]) <= 0 {
						http.Error(w, "Foto y lectura final del odometro son obligatorias", 400)
						return
					}
					m["routeStatus"] = "Finalizada"
					m["routeEndedAt"] = time.Now().Format(time.RFC3339)
					m["odometerEnd"] = q["odometer"]
					m["odometerEndPhoto"] = q["photo"]
					startKm := numberValue(m["odometerStart"])
					endKm := numberValue(q["odometer"])
					if endKm >= startKm && startKm > 0 {
						m["routeKmReal"] = endKm - startKm
					}
					daypart = strings.ToUpper(strings.TrimSpace(fmt.Sprint(q["daypart"])))
					event := map[string]interface{}{"type": "r770_route_session", "date": routeDate, "messengerId": id, "daypart": daypart, "routeAssignedAt": m["routeAssignedAt"], "odometerStart": startKm, "odometerEnd": endKm, "odometerStartPhoto": m["odometerStartPhoto"], "odometerEndPhoto": m["odometerEndPhoto"], "kmReal": math.Max(0, endKm-startKm), "startedAt": m["routeStartedAt"], "endedAt": m["routeEndedAt"], "delayMinutes": m["routeStartDelayMinutes"], "delayReason": m["routeStartDelayReason"], "delayResponsible": m["routeStartDelayResponsible"], "at": time.Now().Format(time.RFC3339)}
					if hist, ok := s.state["routeLearning"].([]interface{}); ok {
						s.state["routeLearning"] = append(hist, event)
					} else {
						s.state["routeLearning"] = []interface{}{event}
					}
					delete(m, "activeDaypart")
				}
			}
		}
	}
	s.saveLocked()
	jsonOut(w, map[string]interface{}{"ok": true, "version": s.version})
}

func geoKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	toRad := func(v float64) float64 { return v * math.Pi / 180 }
	dlat, dlon := toRad(lat2-lat1), toRad(lon2-lon1)
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * R * math.Asin(math.Sqrt(a))
}

func numberValue(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f
	}
}

func (s *Server) publicPanel(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	t := r.URL.Query().Get("token")
	if !s.validatePublicToken("", id, t) {
		http.Error(w, "Enlace invalido", 403)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.state["panelAssignments"].([]interface{}) {
		m, _ := v.(map[string]interface{})
		if fmt.Sprint(m["id"]) == id {
			jsonOut(w, map[string]interface{}{"assignment": m, "settings": s.state["settings"], "vehicles": s.state["vehicles"]})
			return
		}
	}
	http.Error(w, "no encontrado", 404)
}
func (s *Server) publicPanelAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q map[string]interface{}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	id := fmt.Sprint(q["panel"])
	t := fmt.Sprint(q["token"])
	if !s.validatePublicToken("", id, t) {
		http.Error(w, "Enlace invalido", 403)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.state["panelAssignments"].([]interface{}) {
		m, _ := v.(map[string]interface{})
		if fmt.Sprint(m["id"]) == id {
			for k, v := range q {
				if k != "token" {
					m[k] = v
				}
			}
			m["lastUpdate"] = time.Now().Format(time.RFC3339)
			if _, ok := q["endPhotos"]; ok {
				m["damageAnalysis"] = s.compareVehiclePhotos(m["startPhotos"], m["endPhotos"])
			}
			vehicleID := fmt.Sprint(m["vehicleId"])
			status := fmt.Sprint(q["status"])
			if arr, ok := s.state["vehicles"].([]interface{}); ok {
				for _, vv := range arr {
					veh, _ := vv.(map[string]interface{})
					if fmt.Sprint(veh["id"]) != vehicleID {
						continue
					}
					if status == "En uso" {
						veh["status"] = "Asignado"
						veh["assignedTo"] = m["messengerId"]
					}
					if status == "Finalizada" {
						veh["status"] = "Disponible"
						veh["assignedTo"] = ""
						if odo, ok := q["odometerEnd"].(float64); ok && odo > 0 {
							veh["mileage"] = odo
						}
					}
				}
			}
		}
	}
	s.saveLocked()
	jsonOut(w, map[string]bool{"ok": true})
}
func (s *Server) compareVehiclePhotos(startV, endV interface{}) map[string]interface{} {
	toStrings := func(v interface{}) []string {
		out := []string{}
		if a, ok := v.([]interface{}); ok {
			for _, x := range a {
				out = append(out, fmt.Sprint(x))
			}
		}
		return out
	}
	a, b := toStrings(startV), toStrings(endV)
	notes := []string{}
	flag := false
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 1; i < n; i++ {
		r1, g1, b1, ok1 := s.avgImage(a[i])
		r2, g2, b2, ok2 := s.avgImage(b[i])
		if !ok1 || !ok2 {
			continue
		}
		diff := math.Sqrt((r1-r2)*(r1-r2) + (g1-g2)*(g1-g2) + (b1-b2)*(b1-b2))
		if diff > 38 {
			flag = true
			notes = append(notes, fmt.Sprintf("Vista %d presenta una diferencia visual relevante (%.0f). Revisar manualmente.", i, diff))
		}
	}
	if len(a) != len(b) {
		flag = true
		notes = append(notes, "La cantidad de fotos iniciales y finales no coincide.")
	}
	if len(notes) == 0 {
		notes = append(notes, "No se detectaron diferencias globales relevantes. Confirmar visualmente antes de cerrar cualquier incidencia.")
	}
	return map[string]interface{}{"possibleChange": flag, "notes": notes, "analyzedAt": time.Now().Format(time.RFC3339)}
}
func (s *Server) avgImage(u string) (float64, float64, float64, bool) {
	name := filepath.Base(strings.TrimPrefix(u, "/uploads/"))
	f, err := os.Open(filepath.Join(s.root, "uploads", name))
	if err != nil {
		return 0, 0, 0, false
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return 0, 0, 0, false
	}
	b := img.Bounds()
	sx := int(math.Max(1, float64(b.Dx()/40)))
	sy := int(math.Max(1, float64(b.Dy()/40)))
	var rr, gg, bb, c float64
	for y := b.Min.Y; y < b.Max.Y; y += sy {
		for x := b.Min.X; x < b.Max.X; x += sx {
			r, g, bl, _ := img.At(x, y).RGBA()
			rr += float64(r >> 8)
			gg += float64(g >> 8)
			bb += float64(bl >> 8)
			c++
		}
	}
	if c == 0 {
		return 0, 0, 0, false
	}
	return rr / c, gg / c, bb / c, true
}

func (s *Server) validPanelRequestToken(t string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, _ := s.state["settings"].(map[string]interface{})
	return st != nil && strings.TrimSpace(t) != "" && t == sval(st["panelRequestPublicToken"])
}

func (s *Server) fleetAvailability(w http.ResponseWriter, r *http.Request) {
	if !s.validPanelRequestToken(r.URL.Query().Get("token")) {
		http.Error(w, "Enlace de solicitud inválido", 403)
		return
	}
	date := r.URL.Query().Get("date")
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	items := []interface{}{}
	for _, v := range s.state["vehicles"].([]interface{}) {
		m, _ := v.(map[string]interface{})
		typ := strings.ToLower(fmt.Sprint(m["type"]))
		if !strings.Contains(typ, "panel") {
			continue
		}
		state := strings.ToLower(fmt.Sprint(m["status"]))
		if state == "mantenimiento" || state == "fuera de servicio" {
			continue
		}
		busy := false
		for _, a := range s.state["panelAssignments"].([]interface{}) {
			aa, _ := a.(map[string]interface{})
			if fmt.Sprint(aa["vehicleId"]) == fmt.Sprint(m["id"]) && fmt.Sprint(aa["date"]) == date && overlap(start, end, fmt.Sprint(aa["start"]), fmt.Sprint(aa["end"])) {
				busy = true
			}
		}
		if !busy {
			count++
			items = append(items, m)
		}
	}
	jsonOut(w, map[string]interface{}{"available": count, "vehicles": items})
}
func overlap(a, b, c, d string) bool {
	if a == "" || b == "" || c == "" || d == "" {
		return true
	}
	return a < d && c < b
}
func (s *Server) panelRequest(w http.ResponseWriter, r *http.Request) {
	if !s.validPanelRequestToken(r.URL.Query().Get("token")) {
		http.Error(w, "Enlace de solicitud inválido", 403)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method", 405)
		return
	}
	var q map[string]interface{}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	date := fmt.Sprint(q["date"])
	d, err := time.ParseInLocation("2006-01-02", date, time.Local)
	advance := 7
	s.mu.RLock()
	if st, ok := s.state["settings"].(map[string]interface{}); ok {
		if v, ok := st["panelRequestAdvanceDays"].(float64); ok && v >= 0 {
			advance = int(v)
		}
	}
	s.mu.RUnlock()
	now := time.Now()
	minDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, advance)
	if err != nil || d.Before(minDate) {
		http.Error(w, fmt.Sprintf("Debe solicitarse con al menos %d dias de anticipacion", advance), 400)
		return
	}
	q["id"] = "REQ-" + strconv.FormatInt(time.Now().Unix(), 10)
	q["status"] = "Solicitada"
	q["createdAt"] = time.Now().Format(time.RFC3339)
	s.mu.Lock()
	arr, _ := s.state["panelRequests"].([]interface{})
	s.state["panelRequests"] = append(arr, q)
	s.saveLocked()
	s.mu.Unlock()
	jsonOut(w, map[string]interface{}{"ok": true, "id": q["id"]})
}

func (s *Server) validClientUploadToken(t string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, _ := s.state["settings"].(map[string]interface{})
	return st != nil && strings.TrimSpace(t) != "" && t == sval(st["clientUploadPublicToken"])
}

func (s *Server) publicClientUpload(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("token")
	if !s.validClientUploadToken(t) {
		http.Error(w, "Enlace de carga inválido", 403)
		return
	}
	if r.Method == http.MethodGet {
		jsonOut(w, map[string]interface{}{"ok": true})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	var q map[string]interface{}
	if bodyJSON(r, &q) != nil {
		http.Error(w, "json", 400)
		return
	}
	name := strings.TrimSpace(sval(q["name"]))
	address := strings.TrimSpace(sval(q["address"]))
	if name == "" || address == "" {
		http.Error(w, "Nombre y dirección son obligatorios", 400)
		return
	}
	odoo := strings.TrimSpace(sval(q["odoo"]))
	phone := strings.TrimSpace(sval(q["phone"]))

	s.mu.Lock()
	defer s.mu.Unlock()
	arr, _ := s.state["clients"].([]interface{})
	updated := false
	var id string
	for _, v := range arr {
		c, _ := v.(map[string]interface{})
		if c == nil {
			continue
		}
		match := (odoo != "" && strings.EqualFold(strings.TrimSpace(sval(c["odoo"])), odoo)) || (phone != "" && strings.TrimSpace(sval(c["phone"])) == phone)
		if !match {
			continue
		}
		c["name"] = name
		c["address"] = address
		if odoo != "" {
			c["odoo"] = odoo
		}
		if phone != "" {
			c["phone"] = phone
		}
		if q["lat"] != nil {
			c["lat"] = q["lat"]
		}
		if q["lng"] != nil {
			c["lng"] = q["lng"]
		}
		c["updatedAt"] = time.Now().Format(time.RFC3339)
		c["source"] = "public-link"
		id = sval(c["id"])
		updated = true
		break
	}
	if !updated {
		id = "c" + strconv.FormatInt(time.Now().UnixNano(), 10)
		c := map[string]interface{}{
			"id": id, "odoo": odoo, "name": name, "phone": phone, "address": address,
			"lat": q["lat"], "lng": q["lng"], "photos": []interface{}{},
			"createdAt": time.Now().Format(time.RFC3339), "source": "public-link",
		}
		s.state["clients"] = append(arr, c)
	}
	if err := s.saveLocked(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	jsonOut(w, map[string]interface{}{"ok": true, "id": id, "updated": updated, "version": s.version})
}

func (s *Server) stopTunnel() {
	s.mu.Lock()
	cmd := s.tunnelCmd
	s.tunnelGeneration++
	s.publicURL = ""
	s.tunnelRunning = false
	s.tunnelCmd = nil
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (s *Server) startTunnel() {
	s.mu.Lock()
	if s.tunnelRunning {
		s.mu.Unlock()
		return
	}
	s.tunnelRunning = true
	s.publicURL = ""
	s.tunnelGeneration++
	gen := s.tunnelGeneration
	s.mu.Unlock()

	finish := func() {
		s.mu.Lock()
		if s.tunnelGeneration == gen {
			s.tunnelRunning = false
			s.tunnelCmd = nil
			s.publicURL = ""
		}
		s.mu.Unlock()
	}
	defer finish()

	cf := filepath.Join(s.root, "cloudflared.exe")
	if _, err := os.Stat(cf); err != nil {
		if found, lookErr := exec.LookPath("cloudflared.exe"); lookErr == nil {
			cf = found
		} else {
			fmt.Println("cloudflared.exe no incluido. Descargando release oficial Cloudflare 2026.8.2...")
			const cfURL = "https://github.com/cloudflare/cloudflared/releases/download/2026.8.2/cloudflared-windows-amd64.exe"
			const cfSHA256 = "c29eee2b121f5436a642eed69fd9767da7e7b8c510fa50aaa130337f931357b5"
			if err := download(cfURL, cf); err != nil {
				fmt.Println("No se pudo descargar cloudflared oficial:", err)
				return
			}
			if err := verifySHA256(cf, cfSHA256); err != nil {
				_ = os.Remove(cf)
				fmt.Println("La descarga de cloudflared no superó la verificación SHA256:", err)
				return
			}
			fmt.Println("cloudflared 2026.8.2 descargado y SHA256 verificado.")
		}
	}

	cfHome := filepath.Join(s.root, ".cloudflared-imfohsa")
	_ = os.MkdirAll(cfHome, 0700)
	cmd := exec.Command(cf, "tunnel", "--no-autoupdate", "--protocol", "http2", "--url", fmt.Sprintf("http://localhost:%d", port))
	cmd.Env = append(os.Environ(), "HOME="+cfHome, "USERPROFILE="+cfHome)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		fmt.Println("Cloudflare no pudo iniciar:", err)
		if filepath.Clean(cf) == filepath.Clean(filepath.Join(s.root, "cloudflared.exe")) {
			_ = os.Remove(cf)
		}
		return
	}
	s.mu.Lock()
	if s.tunnelGeneration != gen {
		s.mu.Unlock()
		_ = cmd.Process.Kill()
		return
	}
	s.tunnelCmd = cmd
	s.mu.Unlock()

	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	foundURL := make(chan string, 1)
	scan := func(rd io.Reader) {
		sc := bufio.NewScanner(rd)
		for sc.Scan() {
			line := sc.Text()
			fmt.Println("[cloudflared]", line)
			if u := re.FindString(line); u != "" {
				select {
				case foundURL <- u:
				default:
				}
			}
		}
	}
	go scan(stderr)
	go scan(stdout)
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	select {
	case u := <-foundURL:
		// La URL de Quick Tunnel es solo una candidata hasta que responda realmente.
		// No se publica a los usuarios antes de comprobar que Cloudflare conserva
		// una conexión saludable; esto evita entregar enlaces que terminan en 1033.
		fmt.Println("URL HTTPS candidata detectada. Validando conexión HTTP/2:", u)
		if !waitPublicReachable(u, 45*time.Second) {
			fmt.Println("ERROR TUNEL: la URL candidata no quedó saludable. No se compartirá. Revise las líneas [cloudflared] anteriores.")
			_ = cmd.Process.Kill()
			<-waitCh
			return
		}
		s.mu.Lock()
		if s.tunnelGeneration == gen {
			s.publicURL = u
		}
		s.mu.Unlock()
		lanLine := "No detectada"
		if ip := localIP(); ip != "" {
			lanLine = fmt.Sprintf("http://%s:%d", ip, port)
		}
		accessText := "LOGÍSTICA IMFOHSA R7.7.6 PILOTO\r\n\r\n" +
			"RED LOCAL - MISMA WIFI:\r\n" + lanLine + "/index.html?v=776\r\n" +
			"(Funciona únicamente dentro de la misma red local.)\r\n\r\n" +
			"HTTPS PUBLICO - CUALQUIER RED:\r\n" + u + "/index.html?v=776\r\n" +
			"(Enlace verificado antes de ser publicado.)\r\n"
		_ = os.WriteFile(filepath.Join(s.root, "ENLACES_DE_ACCESO.txt"), []byte(accessText), 0644)
		fmt.Println("ENLACE PUBLICO LISTO Y VERIFICADO:", u)
		<-waitCh
		fmt.Println("El enlace HTTPS terminó. La próxima acción Compartir generará uno nuevo.")
		return
	case err := <-waitCh:
		fmt.Println("El túnel HTTPS terminó antes de publicar una URL:", err)
		return
	case <-time.After(75 * time.Second):
		fmt.Println("No se obtuvo enlace HTTPS en 75 segundos. El acceso LAN continúa disponible.")
		_ = cmd.Process.Kill()
		<-waitCh
		return
	}
}
func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, strings.TrimSpace(expected)) {
		return fmt.Errorf("SHA256 inesperado: %s", got)
	}
	return nil
}

func download(u, path string) error {
	cli := &http.Client{Timeout: 3 * time.Minute}
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 IMFOHSA")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	tmp := path + ".download"
	_ = os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < 5<<20 {
		_ = os.Remove(tmp)
		return fmt.Errorf("descarga cloudflared incompleta: %d bytes", n)
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func waitPublicReachable(base string, maxWait time.Duration) bool {
	// Solo para pruebas automatizadas internas; no se define en el iniciador de Windows.
	if os.Getenv("IMFOHSA_SKIP_PUBLIC_VERIFY") == "1" {
		return true
	}
	client := &http.Client{Timeout: 4 * time.Second}
	deadline := time.Now().Add(maxWait)
	url := strings.TrimRight(base, "/") + "/api/health?publiccheck=" + strconv.FormatInt(time.Now().UnixNano(), 10)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "IMFOHSA-R7.5-Link-Check")
		resp, err := client.Do(req)
		if err == nil {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return true
			}
		}
		time.Sleep(700 * time.Millisecond)
	}
	return false
}

func localIP() string {
	if c, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer c.Close()
		if a, ok := c.LocalAddr().(*net.UDPAddr); ok && a.IP.To4() != nil {
			return a.IP.String()
		}
	}
	addrs, _ := net.InterfaceAddrs()
	var fallback string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip := ipnet.IP.To4(); ip != nil && !strings.HasPrefix(ip.String(), "169.254.") {
				v := ip.String()
				if strings.HasPrefix(v, "192.168.") || strings.HasPrefix(v, "10.") || strings.HasPrefix(v, "172.") {
					return v
				}
				if fallback == "" {
					fallback = v
				}
			}
		}
	}
	return fallback
}
func openBrowser(url string) {
	if runtime.GOOS == "windows" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

var _ multipart.File
