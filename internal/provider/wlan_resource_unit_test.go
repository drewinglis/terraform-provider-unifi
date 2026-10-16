package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWLANPrivatePresharedKeysRoundTrip(t *testing.T) {
	keys := []WLANPrivatePresharedKeyModel{
		{Password: types.StringValue("first-key"), NetworkID: types.StringValue("net-a")},
		{Password: types.StringValue("second-key"), NetworkID: types.StringValue("net-b")},
	}

	raw, err := wlanPrivatePresharedKeysToSDK(keys)
	if err != nil {
		t.Fatalf("wlanPrivatePresharedKeysToSDK: %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("got %d raw keys, want 2", len(raw))
	}

	var first map[string]any
	if err := json.Unmarshal(raw[0], &first); err != nil {
		t.Fatalf("unmarshal raw[0]: %v", err)
	}
	if first["password"] != "first-key" || first["networkconf_id"] != "net-a" {
		t.Fatalf("raw[0] = %v, want password=first-key networkconf_id=net-a", first)
	}

	set, diags := wlanPrivatePresharedKeysFromSDK(raw)
	if diags.HasError() {
		t.Fatalf("wlanPrivatePresharedKeysFromSDK: %v", diags)
	}
	var got []WLANPrivatePresharedKeyModel
	if diags := set.ElementsAs(context.Background(), &got, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	want := map[string]string{"first-key": "net-a", "second-key": "net-b"}
	if len(got) != len(want) {
		t.Fatalf("got %d keys, want %d", len(got), len(want))
	}
	for _, k := range got {
		if want[k.Password.ValueString()] != k.NetworkID.ValueString() {
			t.Fatalf("key %q has network %q, want %q",
				k.Password.ValueString(), k.NetworkID.ValueString(), want[k.Password.ValueString()])
		}
	}
}

func TestWLANPrivatePresharedKeysFromSDKEmpty(t *testing.T) {
	set, diags := wlanPrivatePresharedKeysFromSDK(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !set.IsNull() {
		t.Fatalf("got %v, want null set", set)
	}
}

func TestWLANPrivatePresharedKeysFromSDKInvalid(t *testing.T) {
	_, diags := wlanPrivatePresharedKeysFromSDK([]json.RawMessage{json.RawMessage(`"not-an-object"`)})
	if !diags.HasError() {
		t.Fatal("expected an error decoding a non-object key")
	}
}
