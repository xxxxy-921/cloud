package engine

import "testing"

func TestLooksLikeVPNRequestKindSpecAcceptsNaturalLanguageSpec(t *testing.T) {
	spec := `员工在 IT 服务台申请开通 VPN 时，服务台需要确认 VPN 账号、准备用什么设备或场景使用，以及这次访问的主要原因。
访问原因包括线上支持、故障排查、生产应急、网络接入问题、外部协作、长期远程办公、跨境访问和安全合规事项。
线上支持、故障排查、生产应急、网络接入问题偏网络连通与业务支持，交给信息部网络管理员处理；外部协作、长期远程办公、跨境访问、安全合规事项涉及外部、长期、跨境或合规风险，交给信息部信息安全管理员处理。
处理人完成处理后流程结束。`
	if !looksLikeVPNRequestKindSpec(spec) {
		t.Fatal("expected natural-language VPN collaboration spec to enable request_kind routing guard")
	}
}
