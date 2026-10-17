package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWLANDataSourcePrivatePresharedKeys(t *testing.T) {
	raw := []json.RawMessage{
		json.RawMessage(`{"password":"first-key","networkconf_id":"net-a"}`),
		json.RawMessage(`{"password":"second-key","networkconf_id":"net-b"}`),
	}

	set, diags := wlanDataSourcePrivatePresharedKeys(raw)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	want := map[string]string{"first-key": "net-a", "second-key": "net-b"}
	elems := set.Elements()
	if len(elems) != len(want) {
		t.Fatalf("got %d keys, want %d", len(elems), len(want))
	}
	for _, e := range elems {
		attrs := e.(types.Object).Attributes()
		password := attrs["password"].(types.String).ValueString()
		networkID := attrs["network_id"].(types.String).ValueString()
		if want[password] != networkID {
			t.Fatalf("key %q has network %q, want %q", password, networkID, want[password])
		}
	}
}

func TestWLANDataSourcePrivatePresharedKeysEmpty(t *testing.T) {
	set, diags := wlanDataSourcePrivatePresharedKeys(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !set.IsNull() {
		t.Fatalf("got %v, want null set", set)
	}
}

func TestWLANDataSourcePrivatePresharedKeysInvalid(t *testing.T) {
	_, diags := wlanDataSourcePrivatePresharedKeys([]json.RawMessage{json.RawMessage(`"not-an-object"`)})
	if !diags.HasError() {
		t.Fatal("expected an error decoding a non-object key")
	}
}
