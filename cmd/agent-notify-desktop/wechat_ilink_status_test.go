package main

import "testing"

func TestMergeWechatIlinkStatusKeepsConnectionFieldsWhenLoginIsConfirmed(t *testing.T) {
	health := WechatIlinkStatus{LoggedIn: true, Bound: true, UserID: "owner@im.wechat"}
	login := WechatIlinkStatus{Status: "confirmed"}

	got := mergeWechatIlinkStatus(health, login)
	if !got.LoggedIn || !got.Bound || got.UserID != "owner@im.wechat" || got.Status != "confirmed" {
		t.Fatalf("merged status = %#v", got)
	}
}
