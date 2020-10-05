# nuki-exporter
Provide status data of your Nuki device (lock, opener)

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
