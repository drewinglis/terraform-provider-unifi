package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/resnickio/unifi-go-sdk/pkg/unifi"
)

// wanNetworkFixture mirrors what the controller returns for a WAN network:
// WAN settings only, none of the LAN keys.
func wanNetworkFixture() *unifi.Network {
	n := &unifi.Network{
		ID:      "wan1",
		SiteID:  "site",
		Name:    "Internet 1",
		Purpose: "wan",
	}
	n.WANType = "dhcp"
	n.WANTypeV6 = "disabled"
	n.WANNetworkGroup = "WAN"
	n.WANDNSPreference = "auto"
	n.WANLoadBalanceType = "weighted"
	n.WANLoadBalanceWeight = func() *int { v := 99; return &v }()
	n.WANFailoverPriority = func() *int { v := 1; return &v }()
	n.WANSmartQEnabled = boolPtr(false)
	n.ReportWANEvent = boolPtr(true)
	n.RoutingTableID = func() *int { v := 201; return &v }()
	n.FirewallZoneID = "zone"
	return n
}

func TestNetworkSDKToStateWAN(t *testing.T) {
	r := &NetworkResource{}
	var state NetworkResourceModel
	if diags := r.sdkToState(context.Background(), wanNetworkFixture(), &state); diags.HasError() {
		t.Fatalf("sdkToState: %v", diags)
	}

	nulls := map[string]bool{
		"enabled":                      state.Enabled.IsNull(),
		"dhcp_enabled":                 state.DHCPEnabled.IsNull(),
		"dhcp_lease":                   state.DHCPLease.IsNull(),
		"igmp_snooping":                state.IGMPSnooping.IsNull(),
		"internet_access_enabled":      state.InternetAccessEnabled.IsNull(),
		"intra_network_access_enabled": state.IntraNetworkAccessEnabled.IsNull(),
		"nat_enabled":                  state.NATEnabled.IsNull(),
		"mdns_enabled":                 state.MDNSEnabled.IsNull(),
		"upnp_lan_enabled":             state.UPnPLANEnabled.IsNull(),
		"network_group":                state.NetworkGroup.IsNull(),
		"ipv6":                         state.IPv6.IsNull(),
		"wan_vlan_enabled":             state.WANVLANEnabled.IsNull(),
	}
	for name, isNull := range nulls {
		if !isNull {
			t.Errorf("%s: want null on a WAN network", name)
		}
	}

	if got := state.WANType.ValueString(); got != "dhcp" {
		t.Errorf("wan_type = %q, want dhcp", got)
	}
	if got := state.WANNetworkGroup.ValueString(); got != "WAN" {
		t.Errorf("wan_network_group = %q, want WAN", got)
	}
	if got := state.WANLoadBalanceWeight.ValueInt64(); got != 99 {
		t.Errorf("wan_load_balance_weight = %d, want 99", got)
	}
	if got := state.WANFailoverPriority.ValueInt64(); got != 1 {
		t.Errorf("wan_failover_priority = %d, want 1", got)
	}
	if !state.ReportWANEvent.ValueBool() || state.WANSmartQEnabled.IsNull() || state.WANSmartQEnabled.ValueBool() {
		t.Errorf("report_wan_event = %v, wan_smartq_enabled = %v; want true, false",
			state.ReportWANEvent, state.WANSmartQEnabled)
	}
	if got := state.RoutingTableID.ValueInt64(); got != 201 {
		t.Errorf("routing_table_id = %d, want 201", got)
	}
}

func TestNetworkSDKToStateCorporateKeepsDefaults(t *testing.T) {
	r := &NetworkResource{}
	n := &unifi.Network{ID: "lan", Name: "IoT", Purpose: "corporate", Enabled: boolPtr(true)}
	n.NetworkGroup = "LAN"
	var state NetworkResourceModel
	if diags := r.sdkToState(context.Background(), n, &state); diags.HasError() {
		t.Fatalf("sdkToState: %v", diags)
	}

	// Absent LAN keys keep their existing non-null fallbacks on LAN networks.
	if state.NATEnabled.IsNull() || state.NATEnabled.ValueBool() {
		t.Errorf("nat_enabled = %v, want false", state.NATEnabled)
	}
	if got := state.DHCPLease.ValueInt64(); got != defaultDHCPLease {
		t.Errorf("dhcp_lease = %d, want %d", got, defaultDHCPLease)
	}
	if state.IPv6.IsNull() {
		t.Error("ipv6: want an object on a LAN network")
	}
	if !state.WANType.IsNull() || !state.WANNetworkGroup.IsNull() || !state.RoutingTableID.IsNull() {
		t.Errorf("WAN attributes should be null on a LAN network: %v %v %v",
			state.WANType, state.WANNetworkGroup, state.RoutingTableID)
	}
}

func planToSDKJSON(t *testing.T, plan *NetworkResourceModel) map[string]any {
	t.Helper()
	r := &NetworkResource{}
	var diags diag.Diagnostics
	n := r.planToSDK(context.Background(), plan, &diags)
	if diags.HasError() {
		t.Fatalf("planToSDK: %v", diags)
	}
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNetworkPlanToSDKWANOmitsLANSettings(t *testing.T) {
	// A WAN plan as the wanNetworkUseState modifier leaves it: unconfigured LAN
	// attributes are null on update and unknown on create.
	plan := &NetworkResourceModel{
		Name:                      types.StringValue("Internet 1"),
		Purpose:                   types.StringValue("wan"),
		Enabled:                   types.BoolUnknown(),
		DHCPEnabled:               types.BoolUnknown(),
		DHCPLease:                 types.Int64Unknown(),
		IGMPSnooping:              types.BoolNull(),
		InternetAccessEnabled:     types.BoolUnknown(),
		IntraNetworkAccessEnabled: types.BoolUnknown(),
		NATEnabled:                types.BoolNull(),
		NetworkGroup:              types.StringUnknown(),
		DHCPDNSEnabled:            types.BoolUnknown(),
		WANType:                   types.StringValue("dhcp"),
		WANNetworkGroup:           types.StringValue("WAN2"),
		WANLoadBalanceType:        types.StringValue("failover-only"),
		WANFailoverPriority:       types.Int64Value(2),
		ReportWANEvent:            types.BoolValue(false),
		RoutingTableID:            types.Int64Value(202),
	}
	m := planToSDKJSON(t, plan)

	for _, k := range []string{
		"enabled", "dhcpd_enabled", "dhcpd_leasetime", "igmp_snooping", "internet_access_enabled",
		"intra_network_access_enabled", "is_nat", "networkgroup", "dhcpd_dns_enabled", "routing_table_id",
	} {
		if v, ok := m[k]; ok {
			t.Errorf("%s = %v sent for a WAN network", k, v)
		}
	}
	want := map[string]any{
		"wan_type":              "dhcp",
		"wan_networkgroup":      "WAN2",
		"wan_load_balance_type": "failover-only",
		"wan_failover_priority": float64(2),
		"report_wan_event":      false,
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
}

func TestNetworkPlanToSDKCorporateSendsLANSettings(t *testing.T) {
	plan := &NetworkResourceModel{
		Name:                      types.StringValue("IoT"),
		Purpose:                   types.StringValue("corporate"),
		Enabled:                   types.BoolValue(true),
		DHCPEnabled:               types.BoolValue(true),
		IGMPSnooping:              types.BoolValue(false),
		InternetAccessEnabled:     types.BoolValue(true),
		IntraNetworkAccessEnabled: types.BoolUnknown(),
		NATEnabled:                types.BoolValue(true),
		NetworkGroup:              types.StringValue("LAN"),
	}
	m := planToSDKJSON(t, plan)

	want := map[string]any{
		"enabled":                 true,
		"dhcpd_enabled":           true,
		"igmp_snooping":           false,
		"internet_access_enabled": true,
		"is_nat":                  true,
		"networkgroup":            "LAN",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
	if _, ok := m["wan_type"]; ok {
		t.Error("wan_type sent for a LAN network")
	}
	if _, ok := m["intra_network_access_enabled"]; ok {
		t.Error("unset intra_network_access_enabled was sent")
	}
}
