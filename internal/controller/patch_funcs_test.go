package controller

import (
	"testing"
)

func Test_configMapDataHash(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
	}{
		{
			name: "empty map",
			data: map[string]string{},
		},
		{
			name: "single key",
			data: map[string]string{
				"config": "key1=value1\nkey2=value2\n",
			},
		},
		{
			name: "multiple keys",
			data: map[string]string{
				"config":   "max_log_size=100\nserver_url=https://app.lightrun.com\n",
				"metadata": `{"tags":[{"name":"production"}],"agentName":"my-agent"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := configMapDataHash(tt.data)
			for i := 0; i < 100; i++ {
				if got := configMapDataHash(tt.data); got != first {
					t.Errorf("configMapDataHash() not deterministic: iteration %d got %v, want %v", i, got, first)
				}
			}
		})
	}
}

func Test_configMapDataHash_different_data(t *testing.T) {
	a := map[string]string{
		"config":   "key=value1\n",
		"metadata": `{"agentName":"agent-a"}`,
	}
	b := map[string]string{
		"config":   "key=value2\n",
		"metadata": `{"agentName":"agent-a"}`,
	}
	c := map[string]string{
		"config":   "key=value1\n",
		"metadata": `{"agentName":"agent-b"}`,
	}

	hashA := configMapDataHash(a)
	hashB := configMapDataHash(b)
	hashC := configMapDataHash(c)

	if hashA == hashB {
		t.Errorf("expected different hashes for different config values, both got %v", hashA)
	}
	if hashA == hashC {
		t.Errorf("expected different hashes for different metadata values, both got %v", hashA)
	}
}

func Test_configMapDataHash_key_order_independent(t *testing.T) {
	data1 := map[string]string{
		"config":   "server=localhost\n",
		"metadata": `{"agentName":"test"}`,
	}
	data2 := map[string]string{
		"metadata": `{"agentName":"test"}`,
		"config":   "server=localhost\n",
	}

	hash1 := configMapDataHash(data1)
	hash2 := configMapDataHash(data2)

	if hash1 != hash2 {
		t.Errorf("hash should be independent of insertion order: got %v and %v", hash1, hash2)
	}
}
