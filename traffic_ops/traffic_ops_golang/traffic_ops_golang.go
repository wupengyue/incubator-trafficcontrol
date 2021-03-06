package main

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"

	"github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/common/log"

	_ "github.com/lib/pq"
)

const Version = "0.1"

const DefaultConfigPath = "/opt/traffic_ops/traffic_ops_golang.config"

const OldConfig = true
const OldConfigCDNConfPath = "/opt/traffic_ops/app/conf/cdn.conf"
const OldConfigDBConfPath = "/opt/traffic_ops/app/conf/production/database.conf"

func main() {
	configFileName := flag.String("cfg", "", "The config file path")
	oldConfig := flag.Bool("oldcfg", true, "Whether to look for old Perl Traffic Ops config files")
	flag.Parse()
	if *configFileName == "" {
		*configFileName = DefaultConfigPath
	}

	cfg := Config{}
	err := error(nil)
	if !*oldConfig {
		if cfg, err = LoadConfig(*configFileName); err != nil {
			fmt.Println("Error loading config '" + *configFileName + "': " + err.Error())
			return
		}
	} else {
		if cfg, err = GetPerlConfigs(OldConfigCDNConfPath, OldConfigDBConfPath); err != nil {
			fmt.Println("Error loading old configs '" + OldConfigCDNConfPath + "' and '" + OldConfigDBConfPath + "': " + err.Error())
			return
		}
	}

	if err := log.InitCfg(cfg); err != nil {
		fmt.Println("Error initializing loggers: %v", err)
		return
	}

	sslStr := "require"
	if !cfg.DBSSL {
		sslStr = "disable"
	}

	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", cfg.DBUser, cfg.DBPass, cfg.DBServer, cfg.DBDB, sslStr))
	if err != nil {
		log.Errorf("opening database: %v\n", err)
		return
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.MaxDBConnections)

	if err := RegisterRoutes(ServerData{DB: db, Config: cfg}); err != nil {
		log.Errorf("registering routes: %v\n", err)
		return
	}

	log.Infof("Listening on " + cfg.HTTPPort)
	if err := http.ListenAndServeTLS(":"+cfg.HTTPPort, cfg.CertPath, cfg.KeyPath, nil); err != nil {
		log.Errorf("stopping server: %v\n", err)
		return
	}
}
