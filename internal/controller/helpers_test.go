package controller

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func Test_findEnvVarIndex(t *testing.T) {
	type args struct {
		envVarName string
		envVarList []corev1.EnvVar
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "correctly finds the index of the env var",
			args: args{
				envVarName: "test",
				envVarList: []corev1.EnvVar{
					{
						Name:  "test",
						Value: "test",
					},
				},
			},
			want: 0,
		},
		{
			name: "correctly finds the index of the env var",
			args: args{
				envVarName: "test",
				envVarList: []corev1.EnvVar{
					{
						Name:  "test1",
						Value: "test",
					},
				},
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findEnvVarIndex(tt.args.envVarName, tt.args.envVarList); got != tt.want {
				t.Errorf("findEnvVarIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unpatchEnvVarValue(t *testing.T) {
	type args struct {
		origValue    string
		removalValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "correctly removes the value from the env var",
			args: args{
				origValue:    "test",
				removalValue: "test",
			},
			want: "",
		},
		{
			name: "not found substring",
			args: args{
				origValue:    "test",
				removalValue: "test1",
			},
			want: "test",
		},
		{
			name: "with space",
			args: args{
				origValue:    "test this string",
				removalValue: " this",
			},
			want: "test string",
		},
		{
			name: "unpatch empty value",
			args: args{
				origValue:    "test this string",
				removalValue: "",
			},
			want: "test this string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unpatchEnvVarValue(tt.args.origValue, tt.args.removalValue); got != tt.want {
				t.Errorf("unpatchEnvVarValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_agentEnvVarArgument(t *testing.T) {
	type args struct {
		mountPath     string
		agentCliFlags string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "correctly returns the agent env var argument",
			args: args{
				mountPath:     "test",
				agentCliFlags: "test",
			},
			want:    " -agentpath:test/agent/lightrun_agent.so=test",
			wantErr: false,
		},
		{
			name: "correctly returns the agent env var argument",
			args: args{
				mountPath:     "test",
				agentCliFlags: "",
			},
			want:    " -agentpath:test/agent/lightrun_agent.so",
			wantErr: false,
		},
		{
			name: "return error when agentpath with agentCliFlags has more than 1024 chars",
			args: args{
				mountPath:     "test",
				agentCliFlags: strings.Repeat("a", 1024),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := agentEnvVarArgument(tt.args.mountPath, tt.args.agentCliFlags)
			if (err != nil) != tt.wantErr {
				t.Errorf("agentEnvVarArgument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("agentEnvVarArgument() = %v, want %v", got, tt.want)
			}
		})
	}
}
