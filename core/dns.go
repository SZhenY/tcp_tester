// dns.go — DNS 解析与域名校验

package core

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// dnsServers 自定义 DNS 服务器列表（国内公共 DNS）
var dnsServers = []string{"223.5.5.5", "119.29.29.29"}

// validateDomain 校验域名格式
func validateDomain(domain string) string {
	if domain == "" {
		return "域名不能为空"
	}
	if len(domain) > 253 {
		return "域名长度不能超过253个字符"
	}
	for _, c := range domain {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_') {
			return "域名包含非法字符"
		}
	}
	return ""
}

// lookupIPWithFallback 使用自定义 DNS 解析域名，失败时回退到系统 DNS
func lookupIPWithFallback(domain string) ([]net.IP, error) {
	// 尝试自定义 DNS 服务器
	for _, dns := range dnsServers {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "udp", dns+":53")
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ips, err := resolver.LookupIP(ctx, "ip", domain)
		cancel()
		if err == nil && len(ips) > 0 {
			return ips, nil
		}
	}

	// 回退到系统 DNS
	return net.LookupIP(domain)
}

// ResolveDomain 解析域名，返回分类后的 IPv4/IPv6 地址字符串
func (a *App) ResolveDomain(domain string) (string, error) {
	if err := validateDomain(domain); err != "" {
		return "", fmt.Errorf(err)
	}

	ips, err := lookupIPWithFallback(domain)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("解析失败: %v", err)
	}

	// 按 IPv4/IPv6 分类
	var ipv4s, ipv6s []string
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4s = append(ipv4s, ip.String())
		} else {
			ipv6s = append(ipv6s, ip.String())
		}
	}

	// 拼接结果
	var parts []string
	if len(ipv4s) > 0 {
		parts = append(parts, "IPv4:\n"+strings.Join(ipv4s, "\n"))
	}
	if len(ipv6s) > 0 {
		parts = append(parts, "IPv6:\n"+strings.Join(ipv6s, "\n"))
	}
	return strings.Join(parts, "\n"), nil
}
