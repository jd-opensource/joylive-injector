package resource

import (
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetApplicationEnvironments(t *testing.T) {
	// 模拟 HTTP 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/ns/test-namespace/application/test-application/environments", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"ENV_VAR1": "value1",
				"ENV_VAR2": "value2"
			}
		}`))
	}))
	defer mockServer.Close()

	// 替换配置中的 ControlPlaneUrl
	originalURL := config.ControlPlaneUrl
	config.ControlPlaneUrl = mockServer.URL
	defer func() { config.ControlPlaneUrl = originalURL }()

	// 设置测试数据
	labels := map[string]string{
		config.ServiceSpaceLabel: "test-namespace",
		config.ApplicationLabel:  "test-application",
	}

	// 调用被测试函数
	data, err := GetApplicationEnvironments(labels)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"ENV_VAR1": "value1",
		"ENV_VAR2": "value2",
	}, data)
}

func TestGetApplicationEnvironments2(t *testing.T) {
	config.ControlPlaneUrl = "http://localhost:8000/v1"

	// 设置测试数据
	labels := map[string]string{
		config.ServiceSpaceLabel: "test-namespace",
		config.ApplicationLabel:  "test-application",
	}

	// 调用被测试函数
	data, err := GetApplicationEnvironments(labels)

	// 验证结果
	assert.NoError(t, err)
	fmt.Print(data)
}
