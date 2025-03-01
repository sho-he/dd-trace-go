// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package telemetrytest

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationInfo verifies that an integration leveraging instrumentation telemetry
// sends the correct data to the telemetry client.
func TestIntegrationInfo(t *testing.T) {
	// mux.NewRouter() uses the net/http and gorilla/mux integration
	mux.NewRouter()
	integrations := telemetry.Integrations()
	require.Len(t, integrations, 2)
	assert.Equal(t, integrations[0].Name, "net/http")
	assert.True(t, integrations[0].Enabled)
	assert.Equal(t, integrations[1].Name, "gorilla/mux")
	assert.True(t, integrations[1].Enabled)
}

type contribPkg struct {
	ImportPath string
	Name       string
	Imports    []string
	Dir        string
}

var TelemetryImport = "gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"

func readPackage(t *testing.T, path string) contribPkg {
	cmd := exec.Command("go", "list", "-json", path)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	require.NoError(t, err)
	p := contribPkg{}
	err = json.Unmarshal(output, &p)
	require.NoError(t, err)
	return p
}

func (p *contribPkg) hasTelemetryImport(t *testing.T) bool {
	for _, imp := range p.Imports {
		if imp == TelemetryImport {
			return true
		}
	}
	// if we didn't find it imported directly, it might be imported in one of sub-package imports
	for _, imp := range p.Imports {
		if strings.HasPrefix(imp, p.ImportPath) {
			p := readPackage(t, imp)
			if p.hasTelemetryImport(t) {
				return true
			}
		}
	}
	return false
}

// TestTelemetryEnabled verifies that the expected contrib packages leverage instrumentation telemetry
func TestTelemetryEnabled(t *testing.T) {
	body, err := exec.Command("go", "list", "-json", "../../...").Output()
	require.NoError(t, err)

	var packages []contribPkg
	stream := json.NewDecoder(strings.NewReader(string(body)))
	for stream.More() {
		var out contribPkg
		err := stream.Decode(&out)
		require.NoError(t, err)
		packages = append(packages, out)
	}
	for _, pkg := range packages {
		if strings.Contains(pkg.ImportPath, "/test") || strings.Contains(pkg.ImportPath, "/internal") || strings.Contains(pkg.ImportPath, "/cmd") {
			continue
		}
		if !pkg.hasTelemetryImport(t) {
			t.Fatalf(`package %q is expected use instrumentation telemetry. For more info see https://github.com/DataDog/dd-trace-go/blob/main/contrib/README.md#instrumentation-telemetry`, pkg.ImportPath)
		}
	}
}
