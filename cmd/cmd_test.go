//  Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rsvihladremio/dremio-stress/cmd"
	"github.com/rsvihladremio/dremio-stress/pkg/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileReader struct {
	MockReadFile func(filename string) ([]byte, error)
}

func (m MockFileReader) ReadFile(filename string) ([]byte, error) {
	return m.MockReadFile(filename)
}

type MockEngine struct {
	mock.Mock
}

func (m *MockEngine) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEngine) Execute(string) error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockEngine) Name() string {
	return "MyMock"
}

// Implement other methods of protocol.Engine in a similar way

func TestExecute(t *testing.T) {
	t.Run("should close engine and return error if engine initialization fails", func(t *testing.T) {
		// Arrange
		args := conf.Args{
			StressArgs: conf.StressArgs{
				JSONConfigPath: "./mock.json",
			},
			Protocol: conf.ODBC,
		}

		mockEngine := new(MockEngine)
		mockEngine.On("Close").Return(nil)

		mockFileReader := MockFileReader{
			MockReadFile: func(filename string) ([]byte, error) {
				return []byte("mock data"), nil
			},
		}

		// Act
		err := cmd.ExecuteWithEngine(args, mockEngine, mockFileReader)

		// Assert
		assert.Error(t, err)
		mockEngine.AssertCalled(t, "Close")
	})

}

func TestParseProtocol(t *testing.T) {
	t.Run("should return HTTP when input is http", func(t *testing.T) {
		result, err := cmd.ParseProtocol("http")
		assert.NoError(t, err)
		assert.Equal(t, conf.HTTP, result)
	})

	t.Run("should return ODBC when input is odbc", func(t *testing.T) {
		result, err := cmd.ParseProtocol("odbc")
		assert.NoError(t, err)
		assert.Equal(t, conf.ODBC, result)
	})

	t.Run("should return error when input is unsupported", func(t *testing.T) {
		_, err := cmd.ParseProtocol("unsupported")
		assert.Error(t, err)
	})
}

func TestParseArgs(t *testing.T) {
	t.Run("should parse command line arguments correctly", func(t *testing.T) {
		os.Args = []string{
			"dremio-stress",
			"-user=dremio",
			"-password=dremio123",
			"-url=http://localhost:9047",
			"-conf=./stress.json",
			"-protocol=http",
		}

		args, err := cmd.ParseArgs()
		assert.NoError(t, err)
		assert.Equal(t, "dremio", args.ProtocolArgs.User)
		assert.Equal(t, "dremio123", args.ProtocolArgs.Password)
		assert.Equal(t, "http://localhost:9047", args.ProtocolArgs.URL)
		assert.Equal(t, "./stress.json", args.StressArgs.JSONConfigPath)
		assert.Equal(t, conf.HTTP, args.Protocol)
	})
}

func TestGetEngine(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "{\"token\":\"mytoken\"}")
	}))

	// Defer server close so that it will be closed at the end of the test
	defer server.Close()

	// Test HTTP Engine
	httpArgs := conf.Args{
		Protocol: conf.HTTP,
		ProtocolArgs: conf.ProtocolArgs{
			User:     "dremio",
			Password: "dremio123",
			URL:      server.URL,
			SkipSSL:  false,
			Timeout:  60,
		},
	}

	httpEngine, err := cmd.GetEngine(httpArgs)
	assert.NoError(t, err)
	assert.NotNil(t, httpEngine)

	// Test ODBC Engine
	odbcArgs := conf.Args{
		Protocol:     conf.ODBC,
		ProtocolArgs: conf.ProtocolArgs{},
	}

	odbcEngine, err := cmd.GetEngine(odbcArgs)
	assert.NoError(t, err)
	assert.NotNil(t, odbcEngine)

	// Test invalid engine
	invalidArgs := conf.Args{
		Protocol:     10,
		ProtocolArgs: conf.ProtocolArgs{},
	}

	_, err = cmd.GetEngine(invalidArgs)
	assert.Error(t, err)
}
