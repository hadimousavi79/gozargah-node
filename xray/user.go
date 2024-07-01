package xray

import (
	"github.com/google/uuid"
	"marzban-node/xray_api/proto/proxy/shadowsocks"
	"marzban-node/xray_api/types"
	"slices"
)

func SetupUserAccount(user User, email string) types.ProxySettings {
	settings := types.ProxySettings{}

	if user.Proxies.Vmess != nil && user.Proxies.Vmess.ID != uuid.Nil {
		settings.Vmess = &types.VMessAccount{
			BaseAccount: types.BaseAccount{
				Email: email,
				Level: uint32(0),
			},
			ID: user.Proxies.Vmess.ID,
		}
	}

	if user.Proxies.Vless != nil && user.Proxies.Vless.ID != uuid.Nil {
		settings.Vless = &types.VLESSAccount{
			BaseAccount: types.BaseAccount{
				Email: email,
				Level: uint32(0),
			},
			ID:   user.Proxies.Vless.ID,
			Flow: user.Proxies.Vless.Flow,
		}
	}

	if user.Proxies.Trojan != nil && &user.Proxies.Trojan.Password != nil {
		settings.Trojan = &types.TrojanAccount{
			BaseAccount: types.BaseAccount{
				Email: email,
				Level: uint32(0),
			},
			Password: user.Proxies.Trojan.Password,
		}
	}

	if user.Proxies.Shadowsocks != nil && &user.Proxies.Shadowsocks.Password != nil {
		settings.Shadowsocks = &types.ShadowsocksAccount{
			BaseAccount: types.BaseAccount{
				Email: email,
				Level: uint32(0),
			},
			Password: user.Proxies.Trojan.Password,
			Method:   user.Proxies.Shadowsocks.Method,
		}
	}

	return settings
}

func IsActiveInbound(inbound Inbound, user User, settings types.ProxySettings) (types.Account, bool) {
	switch inbound.Protocol {
	case Vmess:
		if slices.Contains(user.Inbounds.Vmess, inbound.Tag) {
			return settings.Vmess, true
		}
	case Vless:
		if slices.Contains(user.Inbounds.Vless, inbound.Tag) {
			account := *settings.Vless

			network, networkOk := inbound.StreamSettings["network"].(string)
			tls, tlsOk := inbound.StreamSettings["security"].(string)

			headerMap, headerMapOk := inbound.StreamSettings["header"].(map[string]interface{})
			headerType, headerTypeOk := "", false
			if headerMapOk {
				headerType, headerTypeOk = headerMap["Type"].(string)
			}

			if user.Proxies.Vless.Flow != types.NONE {
				if networkOk && (network == "tcp" || network == "kcp") {
					if !(tlsOk && (tls == "tls" || tls == "reality")) || (headerTypeOk && headerType == "http") {
						account.Flow = types.NONE
					}
				} else if headerTypeOk && headerType == "http" {
					account.Flow = types.NONE
				} else {
					account.Flow = types.NONE
				}
			}
			return &account, true
		}
	case Trojan:
		if slices.Contains(user.Inbounds.Trojan, inbound.Tag) {
			return settings.Trojan, true
		}
	case Shadowsocks:
		if slices.Contains(user.Inbounds.Shadowsocks, inbound.Tag) {
			return settings.Shadowsocks, true
		}
	}
	return nil, false
}

type VmessSetting struct {
	ID uuid.UUID `json:"id"`
}

type VlessSetting struct {
	ID   uuid.UUID       `json:"id"`
	Flow types.XTLSFlows `json:"flow"`
}

type TrojanSetting struct {
	Password string          `json:"password"`
	Flow     types.XTLSFlows `json:"flow"`
}

type ShadowsocksSetting struct {
	Password string                 `json:"password"`
	Method   shadowsocks.CipherType `json:"method"`
}

type Proxy struct {
	Vmess       *VmessSetting       `json:"vmess,omitempty"`
	Vless       *VlessSetting       `json:"vless,omitempty"`
	Trojan      *TrojanSetting      `json:"trojan,omitempty"`
	Shadowsocks *ShadowsocksSetting `json:"shadowsocks,omitempty"`
}

type Inbounds struct {
	Vmess       []string `json:"vmess,omitempty"`
	Vless       []string `json:"vless,omitempty"`
	Trojan      []string `json:"trojan,omitempty"`
	Shadowsocks []string `json:"shadowsocks,omitempty"`
}

// User struct used to get detail of a user from main panel
type User struct {
	ID       int      `json:"id"`
	Username string   `json:"username,omitempty"`
	Proxies  Proxy    `json:"proxies,omitempty"`
	Inbounds Inbounds `json:"Inbounds,omitempty"`
}
