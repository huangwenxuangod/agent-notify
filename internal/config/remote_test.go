package config

import (
	"os"
	"testing"
)

func TestMigrateRemoteDeliveryUsesExistingCodexChannel(t *testing.T) {
	cfg := Default()
	cfg.Remote = RemoteDeliveryConfig{}
	cfg.Notify.Codex.Channels.Ntfy.Enabled = true
	cfg.Notify.Codex.Channels.Ntfy.TopicURL = "https://ntfy.sh/existing-topic"

	if !migrateRemoteDelivery(&cfg) {
		t.Fatal("migrateRemoteDelivery() = false, want migration")
	}
	if !cfg.Remote.Ntfy.Enabled || cfg.Remote.Ntfy.TopicURL != "https://ntfy.sh/existing-topic" {
		t.Fatalf("remote ntfy = %#v, want existing Codex configuration", cfg.Remote.Ntfy)
	}
}

func TestMigrateRemoteDeliveryPreservesExistingGlobalConfiguration(t *testing.T) {
	cfg := Default()
	cfg.Remote.Ntfy.Enabled = true
	cfg.Remote.Ntfy.TopicURL = "https://ntfy.sh/global-topic"
	cfg.Notify.Codex.Channels.Ntfy.Enabled = true
	cfg.Notify.Codex.Channels.Ntfy.TopicURL = "https://ntfy.sh/old-topic"

	if migrateRemoteDelivery(&cfg) {
		t.Fatal("migrateRemoteDelivery() = true, want false")
	}
	if cfg.Remote.Ntfy.TopicURL != "https://ntfy.sh/global-topic" {
		t.Fatalf("remote ntfy topic = %q", cfg.Remote.Ntfy.TopicURL)
	}
}

func TestLoadMigratesVersionTwoRobotCredentials(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(path, []byte(`version: 2
remote:
  feishu:
    enabled: true
    webhook_url: https://open.feishu.cn/open-apis/bot/v2/hook/example
    signing_secret: feishu-secret
  dingtalk:
    enabled: true
    webhook_url: https://oapi.dingtalk.com/robot/send?access_token=example
    signing_secret: ding-secret
  ntfy:
    enabled: true
    topic_url: https://ntfy.sh/example
    access_token: ntfy-token
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 3 {
		t.Fatalf("version = %d, want 3", cfg.Version)
	}
	if cfg.Remote.Feishu.SigningSecret != "feishu-secret" || cfg.Remote.DingTalk.SigningSecret != "ding-secret" || cfg.Remote.Ntfy.AccessToken != "ntfy-token" {
		t.Fatalf("credentials were not preserved: %#v", cfg.Remote)
	}
}
