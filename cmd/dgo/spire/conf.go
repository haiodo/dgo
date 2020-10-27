// Copyright (c) 2020 Andrey Sobolev.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spire

import (
	"fmt"
	"path"
	"strings"
)

const SocketEnv = "SPIFFE_ENDPOINT_SOCKET"

func conf(name string, options ...string) string {
	result := name + " {"
	nl := false
	for _, opt := range options {
		if !nl {
			result += "\n"
			nl = true
		}
		for _, o := range strings.Split(opt, "\n") {
			result += fmt.Sprintf("\t%s\n", o)
		}
	}
	result += "}\n"
	return result
}

func confN(name, param string, options ...string) string {
	result := name + " \"" + param + "\"" + " {"
	nl := false
	for _, opt := range options {
		if !nl {
			result += "\n"
			nl = true
		}
		for _, o := range strings.Split(opt, "\n") {
			result += fmt.Sprintf("\t%s\n", o)
		}
	}
	result += "}\n"
	return result
}

func optS(name, value string) string {
	return fmt.Sprintf("%s = \"%s\"", name, value)
}
func opt(name, value string) string {
	return fmt.Sprintf("%s = %s", name, value)
}
func optA(name string, value ...string) string {
	sb := strings.Builder{}
	for _, v := range value {
		if sb.Len() != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("\"%s\"", v))
	}
	return fmt.Sprintf("%s = [%s]", name, sb.String())
}

const (
	spireAgentConfFilename = "agent/agent.conf"
	spireEndpointSocket    = "agent.sock"

	spireServerConfFileName = "server/server.conf"
	spireServerRegSock      = "spire-registration.sock"
)

func genSpireConfig(basePath, socketName string) string {
	dataPath := path.Join(basePath, ".data")
	return conf("agent",
		optS("data_dir", dataPath),
		optS("log_level", "WARN"),
		optS("server_address", "127.0.0.1"),
		optS("server_port", "8081"),
		optS("socket_path", path.Join(basePath, socketName)),
		opt("insecure_bootstrap", "true"),
		optS("trust_domain", "example.org"),
	) +
		conf("plugins",
			confN("NodeAttestor", "join_token",
				conf("plugin_data"),
			),
			confN("KeyManager", "disk",
				conf("plugin_data",
					optS("directory", dataPath)),
			),
			confN("WorkloadAttestor", "unix",
				conf("plugin_data"),
			),
		)
}

func genServerConf(basePath, socketName string) string {
	dataPath := path.Join(basePath, ".data")
	return conf("server",
		optS("bind_address", "127.0.0.1"),
		optS("bind_port", "8081"),
		optS("registration_uds_path", socketName),
		optS("trust_domain", "example.org"),
		optS("data_dir", path.Join(basePath, ".data")),
		optS("log_level", "DEBUG"),
		optS("ca_key_type", "rsa-2048"),
		optS("default_svid_ttl", "1h"),
		opt("ca_subject", conf("",
			optA("country", "US"),
			optA("organization", "SPIFFE"),
			optS("common_name", ""),
		)),
	) +
		conf("plugins",
			confN("DataStore", "sql",
				conf("plugin_data",
					optS("database_type", "sqlite3"),
					optS("connection_string", path.Join(dataPath, "datastore.sqlite3"))),
			),
			confN("NodeAttestor", "join_token",
				conf("plugin_data"),
			),
			confN("NodeResolver", "noop",
				conf("plugin_data"),
			),
			confN("KeyManager", "memory",
				conf("plugin_data"),
			),
		)
}
