package provider

import (
	"context"
	"testing"

	"github.com/resnickio/unifi-go-sdk/pkg/unifi"
)

func TestNetworkDataSourceSDKToState_WAN(t *testing.T) {
	network := &unifi.Network{
		ID:      "wan1",
		Name:    "Internet 1",
		Purpose: "wan",
		NetworkWAN: unifi.NetworkWAN{
			WANType:         "dhcp",
			WANNetworkGroup: "WAN",
		},
	}
	network.WANFailoverPriority = unifi.IntPtr(1)
	network.WANSmartQEnabled = unifi.BoolPtr(false)
	network.RoutingTableID = unifi.IntPtr(201)

	var state NetworkDataSourceModel
	if diags := (&NetworkDataSource{}).sdkToState(context.Background(), network, &state); diags.HasError() {
		t.Fatalf("sdkToState: %v", diags)
	}

	if got := state.WANType.ValueString(); got != "dhcp" {
		t.Errorf("wan_type = %q, want dhcp", got)
	}
	if got := state.WANNetworkGroup.ValueString(); got != "WAN" {
		t.Errorf("wan_network_group = %q, want WAN", got)
	}
	if got := state.WANFailoverPriority.ValueInt64(); got != 1 {
		t.Errorf("wan_failover_priority = %d, want 1", got)
	}
	if state.WANSmartQEnabled.IsNull() || state.WANSmartQEnabled.ValueBool() {
		t.Errorf("wan_smartq_enabled = %v, want false", state.WANSmartQEnabled)
	}
	if got := state.RoutingTableID.ValueInt64(); got != 201 {
		t.Errorf("routing_table_id = %d, want 201", got)
	}
	if !state.WANLoadBalanceWeight.IsNull() {
		t.Errorf("wan_load_balance_weight = %v, want null", state.WANLoadBalanceWeight)
	}
	for name, null := range map[string]bool{
		"enabled":                      state.Enabled.IsNull(),
		"dhcp_enabled":                 state.DHCPEnabled.IsNull(),
		"igmp_snooping":                state.IGMPSnooping.IsNull(),
		"internet_access_enabled":      state.InternetAccessEnabled.IsNull(),
		"intra_network_access_enabled": state.IntraNetworkAccessEnabled.IsNull(),
		"nat_enabled":                  state.NATEnabled.IsNull(),
		"network_group":                state.NetworkGroup.IsNull(),
		"ipv6":                         state.IPv6.IsNull(),
	} {
		if !null {
			t.Errorf("%s should be null for a WAN network", name)
		}
	}
}

func TestNetworkDataSourceSDKToState_LANDefaultsUnchanged(t *testing.T) {
	network := &unifi.Network{ID: "lan", Name: "Default", Purpose: "corporate"}
	network.NetworkGroup = "LAN"

	var state NetworkDataSourceModel
	if diags := (&NetworkDataSource{}).sdkToState(context.Background(), network, &state); diags.HasError() {
		t.Fatalf("sdkToState: %v", diags)
	}

	if state.Enabled.IsNull() || state.Enabled.ValueBool() {
		t.Errorf("enabled = %v, want false", state.Enabled)
	}
	if got := state.NetworkGroup.ValueString(); got != "LAN" {
		t.Errorf("network_group = %q, want LAN", got)
	}
	if state.IPv6.IsNull() {
		t.Error("ipv6 should not be null for a LAN network")
	}
	if !state.WANType.IsNull() {
		t.Errorf("wan_type = %v, want null", state.WANType)
	}
}
