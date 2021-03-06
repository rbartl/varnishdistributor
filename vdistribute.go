package main

import "net/http"
import "log/syslog"
import "io/ioutil"
import "io"
import "os"
import "github.com/pborman/getopt"


var servers []string
var slog *syslog.Writer

func vDistribute(w http.ResponseWriter, r *http.Request) {
    client := &http.Client{
    }
	slog.Notice("distributor called " + r.Method + " " + r.Host + " " + r.RequestURI)

    var body []byte
    var status int
    var statusText string
    for _, server := range servers {
        req, _ := http.NewRequest(r.Method, "http://" + server + r.RequestURI, nil)
        req.Header.Add("Host", r.Host)
        req.Host = r.Host
        req.Header = r.Header
        req.URL.Opaque = r.RequestURI
        resp, err := client.Do(req)
        if err != nil {
            slog.Notice (server + " error returned:" + err.Error())
            continue
        }
        defer resp.Body.Close()
        body, _ = ioutil.ReadAll(resp.Body)
        slog.Notice (server + " returned:" + resp.Status)
        status = resp.StatusCode
        statusText = resp.Status
    }
    // return last status and body
    w.Header().Add("Status", statusText);
    w.WriteHeader(status)
    io.WriteString(w, string(body))
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

