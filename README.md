# Prometheus nuki-exporter
Provide status data of your Nuki device (lock, opener) so that it can be scraped by Prometheus.

Start like this:
`nuki-exporter -v DEBUG -b 192.168.1.2 -c /path/to/nuki-credentials.yaml`

Help output:
```nuki-exporter --help
Incorrect Usage. flag: help requested


  NAME:
     nuki-exporter - report metrics of nuki api

  USAGE:
     nuki-exporter [global options]

  AUTHOR:
     Torben Frey <torben@torben.dev>

  GLOBAL OPTIONS:
     --credentials_file value, -c value  file containing credentials for nuki api. Credentials file is in YAML format and contains token field. Alternatively give token directly, it wins over credentials file. [$CREDENTIALS_FILE]
     --bridge_host value, -b value       fqdn or ip address of bridge [$BRIDGE]
     --token value, -t value             token, wins over credentials file [$TOKEN]
     --listen_address value, -l value    [optional] address to listen on, either :port or address:port (default: ":9314") [$LISTEN_ADDRESS]
     --metrics_path value, -m value      [optional] URL path where metrics are exposed (default: "/metrics") [$METRICS_PATH]
     --log_level value, -v value         [optional] log level, choose from DEBUG, INFO, WARN, ERROR (default: "ERROR") [$LOG_LEVEL]
```
     
Example output:
```# HELP nuki_BatteryChargeState N/A
# TYPE nuki_BatteryChargeState gauge
nuki_BatteryChargeState{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 92
nuki_BatteryChargeState{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 0
# HELP nuki_DoorsensorState N/A
# TYPE nuki_DoorsensorState gauge
nuki_DoorsensorState{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 2
nuki_DoorsensorState{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 0
# HELP nuki_Mode N/A
# TYPE nuki_Mode gauge
nuki_Mode{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 2
nuki_Mode{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 2
# HELP nuki_NumBatteryCharging N/A
# TYPE nuki_NumBatteryCharging gauge
nuki_NumBatteryCharging{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 0
nuki_NumBatteryCharging{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 0
# HELP nuki_NumBatteryCritical N/A
# TYPE nuki_NumBatteryCritical gauge
nuki_NumBatteryCritical{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 0
nuki_NumBatteryCritical{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 0
# HELP nuki_State N/A
# TYPE nuki_State gauge
nuki_State{devicetype="0",firmwareversion="2.8.15",name="Nuki",nukiid="XXX"} 3
nuki_State{devicetype="2",firmwareversion="1.5.3",name="Nuki Opener",nukiid="XXX"} 1
# HELP nuki_lastUpdate Last update timestamp in epoch seconds
# TYPE nuki_lastUpdate gauge
nuki_lastUpdate{scope="global"} 1.601880208e+09```
