package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/resnickio/unifi-go-sdk/pkg/unifi"
)

func TestMergePortOverrideKeepsUnmanagedFields(t *testing.T) {
	var existing []unifi.PortOverride
	live := `[{"port_idx":1,"name":"old","forward":"customize","tagged_vlan_mgmt":"auto",` +
		`"stormctrl_bcast_rate":100,"dot1x_ctrl":"force_authorized","stp_edge_state":"auto"},` +
		`{"port_idx":2,"name":"other","eee_enabled":false}]`
	if err := json.Unmarshal([]byte(live), &existing); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	r := &DevicePortOverrideResource{}
	merged := r.mergePortOverride(existing, &unifi.PortOverride{PortIdx: intPtr(1), Name: "new"}, 1)

	out, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	for _, want := range []string{
		`"name":"new"`, `"forward":"customize"`, `"tagged_vlan_mgmt":"auto"`, `"stormctrl_bcast_rate":100`,
		`"dot1x_ctrl":"force_authorized"`, `"stp_edge_state":"auto"`, `"eee_enabled":false`,
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
	if strings.Contains(string(out), `"name":"old"`) {
		t.Errorf("planned name should replace the old one: %s", out)
	}
}

func TestMergePortOverridePlannedValueWins(t *testing.T) {
	existing := []unifi.PortOverride{{PortIdx: intPtr(1), Forward: "customize"}}
	r := &DevicePortOverrideResource{}
	merged := r.mergePortOverride(existing, &unifi.PortOverride{PortIdx: intPtr(1), Forward: "all"}, 1)
	if merged[0].Forward != "all" {
		t.Errorf("Forward = %q, want all", merged[0].Forward)
	}
}
