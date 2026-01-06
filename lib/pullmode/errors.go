/*
Copyright 2026. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pullmode

import (
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

type AgentHeartbeatTimeoutError struct {
}

func (e *AgentHeartbeatTimeoutError) Error() string {
	return "AgentLastReportTime is older than max allowed age."
}

// IsAgentTimeoutError checks if the SveltosCluster failure message matches AgentTimeoutError
func IsAgentTimeoutError(sveltosCluster *libsveltosv1beta1.SveltosCluster) bool {
	if sveltosCluster.Status.FailureMessage == nil {
		return false
	}

	// Create an instance of the error to get the authoritative message
	err := &AgentHeartbeatTimeoutError{}
	return *sveltosCluster.Status.FailureMessage == err.Error()
}
