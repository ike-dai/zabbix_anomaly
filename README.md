# Zabbix Anomaly Detectior plugin

Anomaly detection score monitoring plugin for Zabbix

# Features

* ChangeFinder score monitoring for a Zabbix item history data
* And, you can detect the change point for any Zabbix monitoring data.

# Requirements

* Zabbix >= 3.0

# Installation & Setting

## 1. Download

Download zabbix anomaly analysis tool.

Please get the binary file that is appropriate for your environment architecture.

[binary file url](https://github.com/ike-dai/zabbix_anomaly/releases)


## 2. Copy to Externalscripts directory

Please copy this tool file to your zabbix servers externalscripts directory.

for example:

    $ cp zabbix_anomaly-linux-amd64 /usr/lib/zabbix/externalscripts/zabbix_anomaly

## 3. Register 2 items 

1. Score check item

* Item name: any
* Type: External check
* Key: zabbix_anomaly["-i","<target item id>","-user","<Zabbix API username>","-pass","<Zabbix API user password","-interval","set check interval(seconds)"]
* Update interval: any(This number should match zabbix_anomaly command -interval option number)
* Type of Information: Character

2. Score register item

* Item name: any
* Type: Zabbix trapper
* Key: anomaly.<original item key> (e.g. anomaly.system.cpu.load[percpu,avg1])
* Type of Information: Numeric(float)

# Contact

Please send feedback to me.

Daisuke IKEDA

Twitter: [@ike_dai](https://twitter.com/ike_dai)

e-mail: <dai.ikd123@gmail.com>

# License

Licensed under the Apache License, Version 2.0. The Apache v2 full text is published at this [link](http://www.apache.org/licenses/LICENSE-2.0).

This tool is based on [go-anomalydetector](https://github.com/chobie/go-anomalydetector).

Copyright 2016 Daisuke IKEDA.
