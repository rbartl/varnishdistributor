package main

import "net/http"
import "log/syslog"
import "io/ioutil"
import "io"
import "os"
import "github.com/pborman/getopt"
import "encoding/json"

var servers []string
var slog *syslog.Writer

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

func vDistribute(w http.ResponseWriter, r *http.Request) {
    client := &http.Client{
    }
	slog.Notice("distributor called " + r.Method + " " + r.Host + " " + r.RequestURI)

    var endpointStatuses []EndpointStatus
    allOK := true

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
            slog.Notice(server + " error returned:" + err.Error())
            status.Error = err.Error()
            status.StatusCode = 0
            status.StatusText = "Connection Error"
            allOK = false
        } else {
            defer resp.Body.Close()
            _, _ = ioutil.ReadAll(resp.Body)
            slog.Notice(server + " returned:" + resp.Status)
            status.StatusCode = resp.StatusCode
            status.StatusText = resp.Status
            
            if resp.StatusCode != 200 {
                allOK = false
            }
        }
        
        endpointStatuses = append(endpointStatuses, status)
    }
    
    // Create response data
    responseData := ResponseData{
        Endpoints: endpointStatuses,
        AllOK:     allOK,
    }
    
    // Convert to JSON
    jsonResponse, err := json.Marshal(responseData)
    if err != nil {
        slog.Notice("JSON marshaling error: " + err.Error())
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


func main() {
    slog, _ = syslog.New(syslog.LOG_INFO, "[vdistribute]")
    defer slog.Close()


    var l = getopt.String('a', ":6083", "listen port (:6083)")

    var opts = getopt.CommandLine

    opts.Parse(os.Args)
    if opts.NArgs() > 0 {
        for _, arg := range opts.Args() {
            slog.Notice("Adding Server:" + arg)
            servers = append(servers, arg)

        }
    } else {
        slog.Notice("Not Enough Servers")
        os.Exit(1)
    }

	http.HandleFunc("/", vDistribute)
	http.ListenAndServe(*l, nil)
}

