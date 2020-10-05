package main

// Build for raspberry zero: env GOOS=linux GOARCH=arm GOARM=6 go build
// Build for raspberry 3: env GOOS=linux GOARCH=arm GOARM=7 go build

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/urfave/cli"

	"github.com/coreos/go-systemd/daemon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// seconds before next loop is started
	minSleepSeconds = 30
)

var (
	lg = kitlog.NewLogfmtLogger(os.Stdout)

	credentials Credentials

	httpClient *http.Client

	// Nuki Bridge delivers one CompanySet per given ID
	metricsResponse []NukiDevice

	bridgeHost string
	token      string

	// exposed holds the various metrics that are collected
	exposed = map[string]*prometheus.GaugeVec{}
	// show last update time to see if system is working correctly
	lastUpdate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nuki_lastUpdate",
		Help: "Last update timestamp in epoch seconds",
	},
		[]string{"scope"},
	)
)

func init() {
	// add the lastUpdate metrics to prometheus
	prometheus.MustRegister(lastUpdate)
}

// Credentials consists of token only
type Credentials struct {
	Token string `json:"token"`
}

// NukiDevice contains filtered data per Device
// NukiDevice Prefix field with Label if value is to be used as label
// NukiDevice Prefix field with Ignore if value is neither a metric nor a label but you want to handle it programmatically
// NukiDevice Fields without Prefix will be used as metrics value
type NukiDevice struct {
	LabelDeviceType      int
	LabelNukiID          int
	LabelName            string
	LabelFirmwareVersion string
	Mode                 int
	State                int
	DoorsensorState      int
	BatteryChargeState   int
	NumBatteryCharging   int
	NumBatteryCritical   int
}

// NukiJSON contains full JSON response
type NukiJSON struct {
	DeviceType      int    `json:"deviceType"`
	NukiID          int    `json:"nukiId"`
	Name            string `json:"name"`
	FirmwareVersion string `json:"firmwareVersion"`
	LastKnownState  struct {
		Mode               int       `json:"mode"`
		State              int       `json:"state"`
		DoorsensorState    int       `json:"doorsensorState"`
		BatteryChargeState int       `json:"batteryChargeState"`
		StateName          string    `json:"stateName"`
		BatteryCritical    bool      `json:"batteryCritical"`
		BatteryCharging    bool      `json:"batteryCharging"`
		Timestamp          time.Time `json:"timestamp"`
	} `json:"lastKnownState"`
}

func getCredentials(credentialsFile string) {

	// given token takes precedence over credentialsFile
	if token != "" {
		credentials.Token = token
	} else {
		osFile, err := os.Open(credentialsFile)
		if err != nil {
			// return if we weren't successful - we have tokenGraceSeconds to retry
			level.Error(lg).Log("msg", fmt.Sprintf("Couldn't read credentials file: %s", err.Error()))
			os.Exit(2)
		}
		//fmt.Println(dat)
		err = yaml.NewDecoder(osFile).Decode(&credentials)
		if err != nil || credentials.Token == "" {
			errorText := "Token not set"
			if err != nil {
				//errorText = err.Error()
			}
			level.Error(lg).Log("msg", fmt.Sprintf("Couldn't parse credentials file: %s", errorText))
			level.Error(lg).Log("msg", fmt.Sprintf("YAML file needs to contain token field"))
			os.Exit(2)
		}
	}
}

func main() {

	// Destination variables of command line parser
	var listenAddress string
	var credentialsFile string
	var metricsPath string
	var logLevel string

	// Override template
	cli.AppHelpTemplate = `
	NAME:
	   {{.Name}} - {{.Usage}}

	USAGE:
	   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
	   {{if len .Authors}}
	AUTHOR:
	   {{range .Authors}}{{ . }}{{end}}
	   {{end}}{{if .Commands}}
	COMMANDS:
	{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
	GLOBAL OPTIONS:
	   {{range .VisibleFlags}}{{.}}
	   {{end}}{{end}}{{if .Copyright }}
	COPYRIGHT:
	   {{.Copyright}}
	   {{end}}{{if .Version}}
	VERSION:
	   {{.Version}}
	   {{end}}
  `

	// TODO: - value checking
	// app is a command line parser
	app := &cli.App{
		Authors: []*cli.Author{
			{
				Name:  "Torben Frey",
				Email: "torben@torben.dev",
			},
		},
		Commands:  nil,
		HideHelp:  true,
		ArgsUsage: " ",
		Name:      "nuki-exporter",
		Usage:     "report metrics of nuki api",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "credentials_file",
				Aliases:     []string{"c"},
				Usage:       "file containing credentials for nuki api. Credentials file is in YAML format and contains token field. Alternatively give token directly, it wins over credentials file.",
				Destination: &credentialsFile,
				EnvVars:     []string{"CREDENTIALS_FILE"},
			},
			&cli.StringFlag{
				Name:        "bridge_host",
				Value:       "",
				Aliases:     []string{"b"},
				Usage:       "fqdn or ip address of bridge",
				Destination: &bridgeHost,
				EnvVars:     []string{"BRIDGE"},
			},
			&cli.StringFlag{
				Name:        "token",
				Value:       "",
				Aliases:     []string{"t"},
				Usage:       "token, wins over credentials file",
				Destination: &token,
				EnvVars:     []string{"TOKEN"},
			},
			&cli.StringFlag{
				Name:        "listen_address",
				Value:       ":9314",
				Aliases:     []string{"l"},
				Usage:       "[optional] address to listen on, either :port or address:port",
				Destination: &listenAddress,
				EnvVars:     []string{"LISTEN_ADDRESS"},
			},
			&cli.StringFlag{
				Name:        "metrics_path",
				Value:       "/metrics",
				Aliases:     []string{"m"},
				Usage:       "[optional] URL path where metrics are exposed",
				Destination: &metricsPath,
				EnvVars:     []string{"METRICS_PATH"},
			},
			&cli.StringFlag{
				Name:        "log_level",
				Value:       "ERROR",
				Aliases:     []string{"v"},
				Usage:       "[optional] log level, choose from DEBUG, INFO, WARN, ERROR",
				Destination: &logLevel,
				EnvVars:     []string{"LOG_LEVEL"},
			},
		},
		Action: func(c *cli.Context) error {

			if credentialsFile == "" {
				if token == "" {
					level.Error(lg).Log("msg", "Either credentials_file or token need to be set!")
					os.Exit(2)
				}
			}

			// Debugging output
			lg = kitlog.NewLogfmtLogger(os.Stdout)
			lg = kitlog.With(lg, "ts", log.DefaultTimestamp, "caller", kitlog.DefaultCaller)
			switch logLevel {
			case "DEBUG":
				lg = level.NewFilter(lg, level.AllowDebug())
			case "INFO":
				lg = level.NewFilter(lg, level.AllowInfo())
			case "WARN":
				lg = level.NewFilter(lg, level.AllowWarn())
			default:
				lg = level.NewFilter(lg, level.AllowError())
			}

			level.Debug(lg).Log("msg", fmt.Sprintf("listenAddress: %s", listenAddress))
			level.Debug(lg).Log("msg", fmt.Sprintf("credentialsFile: %s", credentialsFile))
			level.Debug(lg).Log("msg", fmt.Sprintf("metricsPath: %s", metricsPath))

			// install promhttp handler for metricsPath (/metrics)
			http.Handle(metricsPath, promhttp.Handler())

			// show nice web page if called without metricsPath
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`<html>
					<head><title>Nuki Exporter</title></head>
					<body>
					<h1>Nuki Exporter</h1>
					<p><a href='` + metricsPath + `'>Metrics</a></p>
					</body>
					</html>`))
			})

			// Start the http server in background, but catch error
			go func() {
				err := http.ListenAndServe(listenAddress, nil)
				level.Error(lg).Log("msg", err.Error())
				os.Exit(2)
			}()

			// wait for initialization of http server before looping so the systemd alive check doesn't fail
			time.Sleep(time.Second * 3)

			// notify systemd that we're ready
			daemon.SdNotify(false, daemon.SdNotifyReady)

			// read in credentials from Yaml file or username/password variables
			getCredentials(credentialsFile)

			// TODO: Proxy URL instead of ""
			httpClient = getHTTPClient("")

			// Working loop
			for {

				// does the individual work, so the rest of the code can be used for other exporters
				workHorse()

				// send aliveness to systemd
				systemAlive(listenAddress, metricsPath)

				// sleep minSleepSeconds seconds before starting next loop
				time.Sleep(time.Second * minSleepSeconds)

			}
		},
	}

	// Start the app
	err := app.Run(os.Args)
	level.Error(lg).Log("msg", err.Error())
}

func workHorse() {

	getMetrics()
}

func getHTTPClient(proxyURLStr string) *http.Client {

	var transport *http.Transport
	transport = http.DefaultTransport.(*http.Transport).Clone()
	// tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if proxyURLStr != "" {
		proxyURL, err := url.Parse(proxyURLStr)
		if err != nil {
			level.Error(lg).Log("msg", fmt.Sprintf("Couldn't parse proxy url: %s", err.Error()))
			os.Exit(2)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	//adding the Transport object to the http Client
	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
	return client
}

func getMetrics() {

	url := "http://" + bridgeHost + ":8080/list?token=" + credentials.Token

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		level.Warn(lg).Log("msg", fmt.Sprintf("Couldn't create request: %s", err.Error()))
		return
	}
	// send the metrics request
	res, err := httpClient.Do(req)
	if res.StatusCode != 200 {
		// return if we weren't successful
		body, _ := ioutil.ReadAll(res.Body)
		level.Warn(lg).Log("msg", fmt.Sprintf("Could not get metrics: %d - %s", res.StatusCode, body))
		return
	}
	defer res.Body.Close()

	// read body from response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		// return if we weren't successful - we have tokenGraceSeconds to retry
		level.Warn(lg).Log("msg", fmt.Sprintf("Couldn't read in body: %s", err.Error()))
		return
	}

	fmt.Println(string(body))
	var nukiJSON []NukiJSON
	// unmarshal body content into metricResponse struct
	err = json.Unmarshal(body, &nukiJSON)
	if err != nil {
		level.Warn(lg).Log("msg", fmt.Sprintf("Couldn't unmarshal json: %s", err.Error()))
		return
	}

	var nukiDevice NukiDevice

	// Loop through all devices in response
	for _, nukiJSONDevice := range nukiJSON {

		// convert bool value to int metrics
		switch nukiJSONDevice.LastKnownState.BatteryCritical {
		case false:
			nukiDevice.NumBatteryCritical = 0
		case true:
			nukiDevice.NumBatteryCritical = 1
		default:
			nukiDevice.NumBatteryCritical = 1
		}

		switch nukiJSONDevice.LastKnownState.BatteryCharging {
		case false:
			nukiDevice.NumBatteryCharging = 0
		case true:
			nukiDevice.NumBatteryCharging = 1
		default:
			nukiDevice.NumBatteryCharging = 0
		}

		nukiDevice.LabelDeviceType = nukiJSONDevice.DeviceType
		nukiDevice.LabelFirmwareVersion = nukiJSONDevice.FirmwareVersion
		nukiDevice.LabelName = nukiJSONDevice.Name
		nukiDevice.LabelNukiID = nukiJSONDevice.NukiID
		nukiDevice.BatteryChargeState = nukiJSONDevice.LastKnownState.BatteryChargeState
		nukiDevice.DoorsensorState = nukiJSONDevice.LastKnownState.DoorsensorState
		nukiDevice.Mode = nukiJSONDevice.LastKnownState.Mode
		nukiDevice.State = nukiJSONDevice.LastKnownState.State

		// // Debugging output
		// level.Debug(lg).Log("msg", fmt.Sprintf(""))
		// level.Debug(lg).Log("msg", fmt.Sprintf("===== Labels ====="))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Name:             %s", companySet.LabelName))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Slug:             %s", companySet.LabelSlug))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Country:          %s", companySet.LabelCountryISO))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Name:             %d", companySet.LabelID))
		// level.Debug(lg).Log("msg", fmt.Sprintf("===== Info ====="))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Status:           %s", companySet.IgnoreStatus))
		// level.Debug(lg).Log("msg", fmt.Sprintf("===== Metrics ====="))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Current Baseline: %d", companySet.BaselineCurrent))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Stats60:          %d", companySet.Stats60))
		// level.Debug(lg).Log("msg", fmt.Sprintf("Status:           %d", companySet.NumStatus))

		// create empty array to hold labels
		labels := make([]string, 0)
		// create empty array to hold label values
		labelValues := make([]string, 0)

		// reflect to get members of struct
		nd := reflect.ValueOf(nukiDevice)
		typeOfNukiDevice := nd.Type()

		// Loop over all struct members and collect all fields starting with Label in array of labels
		level.Debug(lg).Log("msg", fmt.Sprintf(""))
		level.Debug(lg).Log("msg", fmt.Sprintf("Looping over NukiDevice"))

		for i := 0; i < nd.NumField(); i++ {
			key := typeOfNukiDevice.Field(i).Name
			value := nd.Field(i).Interface()
			level.Debug(lg).Log("msg", fmt.Sprintf("Field: %s, Value: %v", key, value))
			if strings.HasPrefix(key, "Label") {
				// labels have lower case names
				labels = append(labels, strings.ToLower(strings.TrimPrefix(key, "Label")))
				var labelValue string
				// IDs are returned as integers, convert to string
				if nd.Field(i).Type().Name() == "string" {
					labelValue = nd.Field(i).String()
				} else {
					labelValue = strconv.FormatInt(nd.Field(i).Int(), 10)
				}
				labelValues = append(labelValues, labelValue)
			}
		}
		level.Debug(lg).Log("msg", fmt.Sprintf(""))
		level.Debug(lg).Log("msg", fmt.Sprintf("Labels: %v", labels))

		// Loop over all struct fields and set Exporter to value with list of labels if they don't
		// start with Label or Ignore
		level.Debug(lg).Log("msg", fmt.Sprintf(""))
		for i := 0; i < nd.NumField(); i++ {
			key := typeOfNukiDevice.Field(i).Name
			if !(strings.HasPrefix(key, "Label") || strings.HasPrefix(key, "Ignore")) {
				value := nd.Field(i).Int()
				setPrometheusMetric(key, int(value), labels, labelValues)
			}
		}
	}
}

func setPrometheusMetric(key string, value int, labels []string, labelValues []string) {
	level.Debug(lg).Log("msg", fmt.Sprintf("Key: %s, Value: %d, Labels: %v", key, value, labels))
	// Check if metric is already registered, if not, register it
	_, ok := exposed[key]
	if !ok {
		exposed[key] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nuki_" + key,
			Help: "N/A",
		},
			labels,
		)

		prometheus.MustRegister(exposed[key])
	}

	// Now set the value
	exposed[key].WithLabelValues(labelValues...).Set(float64(value))

	// Update lastUpdate so we immediately see when no updates happen anymore
	now := time.Now()
	seconds := now.Unix()
	lastUpdate.WithLabelValues("global").Set(float64(seconds))

}

func systemAlive(listenAddress string, metricsPath string) {

	// systemd alive check
	var metricsURL string
	if !strings.HasPrefix(listenAddress, ":") {
		// User has provided address + port
		metricsURL = "http://" + listenAddress + metricsPath
	} else {
		// User has provided :port only - we need to check ourselves on 127.0.0.1
		metricsURL = "http://127.0.0.1" + listenAddress + metricsPath
	}

	// Call the metrics URL...
	res, err := http.Get(metricsURL)
	if err == nil {
		// ... and notify systemd that everything was ok
		daemon.SdNotify(false, daemon.SdNotifyWatchdog)
	} else {
		// ... do nothing if it was not ok, but log. Systemd will restart soon.
		level.Warn(lg).Log("msg", fmt.Sprintf("liveness check failed: %s", err.Error()))
	}
	// Read all away or else we'll run out of sockets sooner or later
	_, _ = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
}
