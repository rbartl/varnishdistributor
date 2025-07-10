package main

import "net/http"
import "log/syslog"
import "io/ioutil"
import "io"
import "os"
import "github.com/pborman/getopt"
import "encoding/json"
import "log"
import "fmt"
import "time"
import "net"
import "strings"
import "sync"

var servers []string
var serversMutex sync.RWMutex
var originalHostnames []string
var slog *syslog.Writer
var useSyslog bool = false
var logger *log.Logger
var useLookup bool = false

type EndpointStatus struct {
    Server     string `json:"server"`
    StatusCode int    `json:"status_code"`
    StatusText string `json:"status_text"`
    Error      string `json:"error,omitempty"`
}

type ResponseData struct {
    Endpoints []EndpointStatus `json:"endpoints"`
    AllOK     bool            `json:"all_ok"`
}

// safeLog logs a message either to syslog (if available) or stdout
func safeLog(level, message string) {
    if useSyslog && slog != nil {
        switch level {
        case "NOTICE":
            slog.Notice(message)
        case "ERR":
            slog.Err(message)
        case "WARNING":
            slog.Warning(message)
        default:
            slog.Info(message)
        }
    } else {
        logger.Printf("[%s] %s", level, message)
    }
}

func vDistribute(w http.ResponseWriter, r *http.Request) {
    client := &http.Client{
    }
	safeLog("NOTICE", "distributor called " + r.Method + " " + r.Host + " " + r.RequestURI)

    var endpointStatuses []EndpointStatus
    allOK := true

    serversMutex.RLock()
    for _, server := range servers {
        req, _ := http.NewRequest(r.Method, "http://" + server + r.RequestURI, nil)
        req.Header.Add("Host", r.Host)
        req.Host = r.Host
        req.Header = r.Header
        req.URL.Opaque = r.RequestURI
        resp, err := client.Do(req)
        
        status := EndpointStatus{
            Server: server,
        }
        
        if err != nil {
            safeLog("NOTICE", server + " error returned:" + err.Error())
            status.Error = err.Error()
            status.StatusCode = 0
            status.StatusText = "Connection Error"
            allOK = false
        } else {
            defer resp.Body.Close()
            _, _ = ioutil.ReadAll(resp.Body)
            safeLog("NOTICE", server + " returned:" + resp.Status)
            status.StatusCode = resp.StatusCode
            status.StatusText = resp.Status
            
            if resp.StatusCode != 200 {
                allOK = false
            }
        }
        
        endpointStatuses = append(endpointStatuses, status)
    }
    serversMutex.RUnlock()
    
    // Create response data
    responseData := ResponseData{
        Endpoints: endpointStatuses,
        AllOK:     allOK,
    }
    
    // Convert to JSON
    jsonResponse, err := json.Marshal(responseData)
    if err != nil {
        safeLog("NOTICE", "JSON marshaling error: " + err.Error())
        w.WriteHeader(500)
        io.WriteString(w, "Internal Server Error")
        return
    }
    
    // Set response headers
    w.Header().Set("Content-Type", "application/json")
    
    // Set HTTP status based on whether all endpoints are OK
    if allOK {
        w.WriteHeader(200)
    } else {
        w.WriteHeader(207) // Multi-Status
    }
    
    // Write JSON response
    io.WriteString(w, string(jsonResponse))
}

// healthCheck handles the health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
    safeLog("NOTICE", "health check called " + r.Method + " " + r.Host + " " + r.RequestURI)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    
    response := map[string]interface{}{
        "status": "healthy",
        "service": "vdistribute",
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    jsonResponse, err := json.Marshal(response)
    if err != nil {
        safeLog("ERR", "Health check JSON marshaling error: " + err.Error())
        w.WriteHeader(500)
        io.WriteString(w, "Internal Server Error")
        return
    }
    
    io.WriteString(w, string(jsonResponse))
}

// resolveHostname resolves a hostname to all its IP addresses
func resolveHostname(hostname string) ([]string, error) {
    // Extract host and port if present
    host := hostname
    port := ""
    
    if strings.Contains(hostname, ":") {
        parts := strings.SplitN(hostname, ":", 2)
        host = parts[0]
        port = ":" + parts[1]
    }
    
    // Resolve the hostname
    ips, err := net.LookupHost(host)
    if err != nil {
        return nil, err
    }
    
    // Add port back to each IP if it was present
    var result []string
    for _, ip := range ips {
        if port != "" {
            result = append(result, ip+port)
        } else {
            result = append(result, ip)
        }
    }
    
    return result, nil
}


func main() {
    // Add panic recovery for the entire main function
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintf(os.Stderr, "Fatal error: %v\n", r)
            os.Exit(1)
        }
    }()
    
    // Initialize stdout logger as fallback
    logger = log.New(os.Stdout, "", log.LstdFlags)

    var l = getopt.String('a', ":6083", "listen port (:6083)")
    var syslogFlag = getopt.Bool('s', "use syslog for logging (default: stdout)")
    var lookupFlag = getopt.Bool('l', "perform DNS lookup for hostnames (default: false)")
    var helpFlag = getopt.BoolLong("help", 'h', "show this help message")
    var versionFlag = getopt.BoolLong("version", 'v', "show version information")

    var opts = getopt.CommandLine

    opts.Parse(os.Args)
    
    // Show version if requested
    if *versionFlag {
        fmt.Fprintf(os.Stderr, "vdistribute version 1.0.0\n")
        fmt.Fprintf(os.Stderr, "HTTP request distributor for multiple backend servers\n")
        os.Exit(0)
    }
    
    // Show help if requested
    if *helpFlag {
        fmt.Fprintf(os.Stderr, "vdistribute - HTTP request distributor\n\n")
        fmt.Fprintf(os.Stderr, "DESCRIPTION:\n")
        fmt.Fprintf(os.Stderr, "  vdistribute is a simple HTTP request distributor that forwards requests\n")
        fmt.Fprintf(os.Stderr, "  to multiple backend servers and returns a consolidated JSON response\n")
        fmt.Fprintf(os.Stderr, "  with the status of each endpoint.\n\n")
        fmt.Fprintf(os.Stderr, "USAGE:\n")
        fmt.Fprintf(os.Stderr, "  vdistribute [OPTIONS] server1 server2 [server3...]\n\n")
        fmt.Fprintf(os.Stderr, "OPTIONS:\n")
        opts.PrintUsage(os.Stderr)
        fmt.Fprintf(os.Stderr, "\nARGUMENTS:\n")
        fmt.Fprintf(os.Stderr, "  server1, server2, ...  Backend servers to distribute requests to\n")
        fmt.Fprintf(os.Stderr, "                         (format: host:port or just host for port 80)\n")
        fmt.Fprintf(os.Stderr, "                         With -l flag: hostnames are resolved to all IP addresses\n\n")
        fmt.Fprintf(os.Stderr, "EXAMPLES:\n")
        fmt.Fprintf(os.Stderr, "  vdistribute -a :8080 server1:8080 server2:8080\n")
        fmt.Fprintf(os.Stderr, "  vdistribute -s backend1 backend2 backend3\n")
        fmt.Fprintf(os.Stderr, "  vdistribute -l example.com:8080 api.example.com\n")
        fmt.Fprintf(os.Stderr, "  vdistribute --help\n\n")
        fmt.Fprintf(os.Stderr, "RESPONSE FORMAT:\n")
        fmt.Fprintf(os.Stderr, "  The service returns JSON with the following structure:\n")
        fmt.Fprintf(os.Stderr, "  {\n")
        fmt.Fprintf(os.Stderr, "    \"endpoints\": [\n")
        fmt.Fprintf(os.Stderr, "      {\n")
        fmt.Fprintf(os.Stderr, "        \"server\": \"server:port\",\n")
        fmt.Fprintf(os.Stderr, "        \"status_code\": 200,\n")
        fmt.Fprintf(os.Stderr, "        \"status_text\": \"200 OK\",\n")
        fmt.Fprintf(os.Stderr, "        \"error\": \"error message\" (if connection failed)\n")
        fmt.Fprintf(os.Stderr, "      }\n")
        fmt.Fprintf(os.Stderr, "    ],\n")
        fmt.Fprintf(os.Stderr, "    \"all_ok\": true/false\n")
        fmt.Fprintf(os.Stderr, "  }\n\n")
        fmt.Fprintf(os.Stderr, "HTTP STATUS CODES:\n")
        fmt.Fprintf(os.Stderr, "  200 - All endpoints responded successfully\n")
        fmt.Fprintf(os.Stderr, "  207 - Multi-Status (some endpoints failed)\n")
        fmt.Fprintf(os.Stderr, "  500 - Internal server error\n\n")
        fmt.Fprintf(os.Stderr, "FEATURES:\n")
        fmt.Fprintf(os.Stderr, "  - Forwards all HTTP methods (GET, POST, PUT, DELETE, etc.)\n")
        fmt.Fprintf(os.Stderr, "  - Preserves original request headers and host information\n")
        fmt.Fprintf(os.Stderr, "  - Provides detailed status for each backend server\n")
        fmt.Fprintf(os.Stderr, "  - Supports syslog logging for production environments\n")
        fmt.Fprintf(os.Stderr, "  - Returns appropriate HTTP status codes based on backend responses\n")
        fmt.Fprintf(os.Stderr, "  - DNS lookup support for hostname resolution to all IP addresses\n\n")
        fmt.Fprintf(os.Stderr, "LOGGING:\n")
        fmt.Fprintf(os.Stderr, "  - Default: Logs to stdout with timestamps\n")
        fmt.Fprintf(os.Stderr, "  - With -s flag: Logs to syslog (falls back to stdout if unavailable)\n")
        fmt.Fprintf(os.Stderr, "  - Logs include request details and backend server responses\n\n")
        os.Exit(0)
    }
    
    useSyslog = *syslogFlag
    useLookup = *lookupFlag
    
    if useSyslog {
        // Try to initialize syslog, but don't fail if it's not available
        var err error
        safeLog("NOTICE", "trying to start syslog logging")
        
        // Add panic recovery for syslog initialization
        func() {
            defer func() {
                if r := recover(); r != nil {
                    fmt.Fprintf(os.Stderr, "Panic during syslog initialization: %v\n", r)
                    useSyslog = false
                    slog = nil
                }
            }()
            
            slog, err = syslog.New(syslog.LOG_INFO, "[vdistribute]")
            if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: Could not initialize syslog: %v. Falling back to stdout logging.\n", err)
                useSyslog = false
                slog = nil
            } else {
                defer slog.Close()
                safeLog("NOTICE", "syslog logging initialized successfully")
            }
        }()
    }
    
    if opts.NArgs() > 0 {
        for _, arg := range opts.Args() {
            if useLookup {
                originalHostnames = append(originalHostnames, arg)
            } else {
                safeLog("NOTICE", "Adding Server:" + arg)
                servers = append(servers, arg)
            }
        }
        if useLookup {
            // Initial resolve
            var resolved []string
            for _, hostname := range originalHostnames {
                safeLog("NOTICE", "Resolving server address: " + hostname)
                addrs, err := resolveHostname(hostname)
                if err != nil {
                    safeLog("ERR", "DNS lookup failed for " + hostname + ": " + err.Error())
                    fmt.Fprintf(os.Stderr, "Error: DNS lookup failed for %s: %v\n", hostname, err)
                    os.Exit(1)
                }
                for _, addr := range addrs {
                    safeLog("NOTICE", "Resolved " + hostname + " to " + addr)
                    resolved = append(resolved, addr)
                }
            }
            serversMutex.Lock()
            servers = resolved
            serversMutex.Unlock()
        }
    } else {
        fmt.Fprintf(os.Stderr, "Error: No backend servers specified\n\n")
        fmt.Fprintf(os.Stderr, "USAGE: vdistribute [OPTIONS] server1 server2 [server3...]\n")
        fmt.Fprintf(os.Stderr, "Try 'vdistribute --help' for more information\n")
        os.Exit(1)
    }

    if useLookup {
        go func() {
            ticker := time.NewTicker(60 * time.Second)
            defer ticker.Stop()
            for {
                <-ticker.C
                var resolved []string
                for _, hostname := range originalHostnames {
                    safeLog("NOTICE", "Refreshing DNS for: " + hostname)
                    addrs, err := resolveHostname(hostname)
                    if err != nil {
                        safeLog("ERR", "DNS refresh failed for " + hostname + ": " + err.Error())
                        continue
                    }
                    for _, addr := range addrs {
                        safeLog("NOTICE", "Refreshed " + hostname + " to " + addr)
                        resolved = append(resolved, addr)
                    }
                }
                serversMutex.Lock()
                servers = resolved
                serversMutex.Unlock()
            }
        }()
    }

	http.HandleFunc("/", vDistribute)
    http.HandleFunc("/vdistribute-health", healthCheck)
    safeLog("NOTICE", "Listening on " + *l)
	err := http.ListenAndServe(*l, nil)
	if err != nil {
		safeLog("ERR", "Failed to start server: " + err.Error())
		fmt.Fprintf(os.Stderr, "Error: Failed to start server: %v\n", err)
		os.Exit(1)
	}
    safeLog("NOTICE", "Stopping")
}

