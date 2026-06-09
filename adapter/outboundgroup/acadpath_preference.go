package outboundgroup

import (
	"strings"

	C "github.com/metacubex/mihomo/constant"
)

func acadpathPreferHKGroup(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "auto", "自动选择", "自动选线":
		return true
	default:
		return false
	}
}

func acadpathIsHKProxyName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "hk") ||
		strings.Contains(normalized, "hong kong") ||
		strings.Contains(normalized, "香港")
}

func acadpathPreferredAliveProxies(groupName string, proxies []C.Proxy, testURL string) []C.Proxy {
	if !acadpathPreferHKGroup(groupName) {
		return proxies
	}

	preferred := make([]C.Proxy, 0, len(proxies))
	for _, proxy := range proxies {
		if !acadpathIsHKProxyName(proxy.Name()) {
			continue
		}
		if !proxy.AliveForTestUrl(testURL) {
			continue
		}
		preferred = append(preferred, proxy)
	}

	if len(preferred) > 0 {
		return preferred
	}
	return proxies
}
