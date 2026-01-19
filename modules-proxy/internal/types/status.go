package types

// ProxyStatusPayload describes the gateway attachment status reported back to the UI.
type ProxyStatusPayload struct {
	OK                bool     `json:"ok"`
	Message           string   `json:"message,omitempty"`
	Network           string   `json:"network_name,omitempty"`
	ConnectedNetworks []string `json:"connected_networks,omitempty"`
	MissingNetworks   []string `json:"missing_networks,omitempty"`
	Reattached        bool     `json:"reattached,omitempty"`
}
