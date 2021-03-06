package core

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"

	"github.com/goburrow/melon/health"
)

const (
	pingPath        = "/ping"
	runtimePath     = "/runtime"
	healthCheckPath = "/healthcheck"
	tasksPath       = "/tasks"

	adminHTML = `<!DOCTYPE html>
<html>
<head>
	<title>Operational Menu</title>
</head>
<body>
	<h1>Operational Menu</h1>
	<ul>%[1]s</ul>
</body>
</html>
`
	noHealthChecksWarning = `
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
!    THIS APPLICATION HAS NO HEALTHCHECKS.    !
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
`

	gcTaskName = "gc"
)

// AdminHandler is an item listed in the admin homepage.
type AdminHandler interface {
	Path() string
	Name() string
	http.Handler
}

// AdminEnvironment is an environment context for administrating the application.
type AdminEnvironment struct {
	Router       Router
	HealthChecks health.Registry

	handlers []AdminHandler
	tasks    []Task
}

// NewAdminEnvironment allocates and returns a new AdminEnvironment.
func NewAdminEnvironment() *AdminEnvironment {
	env := &AdminEnvironment{
		HealthChecks: health.NewRegistry(),
	}
	// Default handlers
	env.AddHandler(&pingHandler{}, &runtimeHandler{}, &healthCheckHandler{env.HealthChecks})
	// Default tasks
	env.AddTask(&gcTask{})
	return env
}

// AddTask adds a new task to admin environment. AddTask is not concurrent-safe.
func (env *AdminEnvironment) AddTask(task ...Task) {
	env.tasks = append(env.tasks, task...)
}

// AddHandler registers a handler entry for admin page.
func (env *AdminEnvironment) AddHandler(handler ...AdminHandler) {
	env.handlers = append(env.handlers, handler...)
}

// start registers all required HTTP handlers
func (env *AdminEnvironment) start() {
	env.Router.Handle("GET", "/", &adminIndex{
		handlers:    env.handlers,
		contextPath: env.Router.PathPrefix(),
	})
	// Registered handlers
	for _, h := range env.handlers {
		env.Router.Handle("*", h.Path(), h)
	}
	// Registered tasks
	for _, task := range env.tasks {
		path := tasksPath + "/" + task.Name()
		env.Router.Handle("POST", path, task)
	}
	env.logTasks()
	env.logHealthChecks()
}

// logTasks prints all registered tasks to the log
func (env *AdminEnvironment) logTasks() {
	var buf bytes.Buffer
	for _, task := range env.tasks {
		fmt.Fprintf(&buf, "    %-7s %s%s/%s (%T)\n", "POST",
			env.Router.PathPrefix(), tasksPath, task.Name(), task)
	}
	GetLogger("melon").Infof("tasks =\n\n%s", buf.String())
}

// logTasks prints all registered tasks to the log
func (env *AdminEnvironment) logHealthChecks() {
	names := env.HealthChecks.Names()
	logger := GetLogger("melon")
	logger.Debugf("health checks = %v", names)
	if len(names) <= 0 {
		logger.Warnf(noHealthChecksWarning)
	}
}

// Task is simply a HTTP Handler.
type Task interface {
	Name() string
	http.Handler
}

// adminIndex is the home page of admin.
type adminIndex struct {
	handlers    []AdminHandler
	contextPath string
}

// ServeHTTP handles request to the root of Admin page
func (handler *adminIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer

	for _, h := range handler.handlers {
		fmt.Fprintf(&buf, "<li><a href=\"%[1]s%[2]s\">%[3]s</a></li>",
			handler.contextPath, h.Path(), h.Name())
	}

	w.Header().Set("Cache-Control", "must-revalidate,no-cache,no-store")
	w.Header().Set("Content-Type", "text/html")

	fmt.Fprintf(w, adminHTML, buf.String())
}

// healthCheckHandler is the http handler for /healthcheck page
type healthCheckHandler struct {
	registry health.Registry
}

func (handler *healthCheckHandler) Name() string {
	return "Healthcheck"
}

func (handler *healthCheckHandler) Path() string {
	return healthCheckPath
}

func (handler *healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "must-revalidate,no-cache,no-store")

	results := handler.registry.RunCheckers()
	if len(results) == 0 {
		http.Error(w, "No health checks registered.", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !isAllHealthy(results) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	first := true
	w.Write([]byte("{"))
	for name, result := range results {
		if first {
			first = false
		} else {
			w.Write([]byte(","))
		}
		fmt.Fprintf(w, "\n%q: {\"Healthy\": %t", name, result.Healthy())
		if result.Message() != "" {
			fmt.Fprintf(w, ", \"Message\": %q", result.Message())
		}
		if result.Cause() != nil {
			fmt.Fprintf(w, ", \"Cause\": %q", result.Cause())
		}
		w.Write([]byte("}"))
	}
	w.Write([]byte("\n}\n"))
}

// isAllHealthy checks if all are healthy
func isAllHealthy(results map[string]health.Result) bool {
	for _, result := range results {
		if !result.Healthy() {
			return false
		}
	}
	return true
}

// pingHandler handles ping request to admin /ping
type pingHandler struct {
}

func (handler *pingHandler) Name() string {
	return "Ping"
}

func (handler *pingHandler) Path() string {
	return pingPath
}

func (handler *pingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "must-revalidate,no-cache,no-store")
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("pong\n"))
}

// runtimeHandler displays runtime statistics.
type runtimeHandler struct {
}

func (handler *runtimeHandler) Name() string {
	return "Runtime"
}

func (handler *runtimeHandler) Path() string {
	return runtimePath
}

func (handler *runtimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "must-revalidate,no-cache,no-store")
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "GOARCH: %s\nGOOS: %s\nVersion: %s\nNumCPU: %d\nNumCgoCall: %d\nNumGoroutine: %d\n",
		runtime.GOARCH, runtime.GOOS, runtime.Version(),
		runtime.NumCPU(), runtime.NumCgoCall(), runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// General statistics
	fmt.Fprintf(w, "MemStats:\n\tAlloc: %d\n\tTotalAlloc: %d\n\tSys: %d\n\tLookups: %d\n\tMallocs: %d\n\tFrees: %d\n",
		m.Alloc, m.TotalAlloc, m.Sys, m.Lookups, m.Mallocs, m.Frees)
	// Main allocation heap statistics
	fmt.Fprintf(w, "\tHeapAlloc: %d\n\tHeapSys: %d\n\tHeapIdle: %d\n\tHeapInuse: %d\n\tHeapReleased: %d\n\tHeapObjects: %d\n",
		m.HeapAlloc, m.HeapSys, m.HeapIdle, m.HeapInuse, m.HeapReleased, m.HeapObjects)
	// Low-level fixed-size structure allocator statistics
	fmt.Fprintf(w, "\tStackInuse: %d\n\tStackSys: %d\n\tMSpanInuse: %d\n\tMSpanSys: %d\n\tMCacheInuse: %d\n\tMCacheSys: %d\n\tBuckHashSys: %d\n\tGCSys: %d\n\tOtherSys: %d\n",
		m.StackInuse, m.StackSys, m.MSpanInuse, m.MSpanSys, m.MCacheInuse, m.MCacheSys, m.BuckHashSys, m.GCSys, m.OtherSys)
	// Garbage collector statistics
	fmt.Fprintf(w, "\tNextGC: %d\n\tLastGC: %d\n\tPauseTotalNs: %d\n\tNumGC: %d\n\tEnableGC: %t\n\tDebugGC: %t\n",
		m.NextGC, m.LastGC, m.PauseTotalNs, m.NumGC, m.EnableGC, m.DebugGC)
}

// gcTask performs a garbage collection
type gcTask struct {
}

func (*gcTask) Name() string {
	return gcTaskName
}

func (*gcTask) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Running GC...\n"))
	runtime.GC()
	w.Write([]byte("Done!\n"))
}
