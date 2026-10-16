package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOutletOverridesFromSDK(t *testing.T) {
	raw := []json.RawMessage{
		json.RawMessage(`{"cycle_enabled":false,"name":"Modem","relay_state":true,"index":14}`),
		json.RawMessage(`{"cycle_enabled":true,"name":"","relay_state":false,"index":7}`),
		json.RawMessage(`{"index":3}`),
	}

	list, diags := outletOverridesFromSDK(raw)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got []OutletOverrideModel
	if d := list.ElementsAs(t.Context(), &got, false); d.HasError() {
		t.Fatalf("ElementsAs: %v", d)
	}
	want := []OutletOverrideModel{
		{Index: types.Int64Value(14), Name: types.StringValue("Modem"),
			RelayState: types.BoolValue(true), CycleEnabled: types.BoolValue(false)},
		{Index: types.Int64Value(7), Name: types.StringValue(""),
			RelayState: types.BoolValue(false), CycleEnabled: types.BoolValue(true)},
		{Index: types.Int64Value(3), Name: types.StringNull(),
			RelayState: types.BoolNull(), CycleEnabled: types.BoolNull()},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d overrides, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("override %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestOutletOverridesFromSDK_empty(t *testing.T) {
	list, diags := outletOverridesFromSDK(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || len(list.Elements()) != 0 {
		t.Fatalf("got %v, want empty non-null list", list)
	}
}

func TestOutletOverridesFromSDK_invalid(t *testing.T) {
	_, diags := outletOverridesFromSDK([]json.RawMessage{json.RawMessage(`{"index":"one"}`)})
	if !diags.HasError() {
		t.Fatal("expected an error for a non-numeric index")
	}
}

func TestOutletOverridesToSDK(t *testing.T) {
	overrides := []OutletOverrideModel{
		{Index: types.Int64Value(5), Name: types.StringValue("Dream Machine"),
			RelayState: types.BoolValue(true), CycleEnabled: types.BoolValue(false)},
		{Index: types.Int64Value(7), Name: types.StringValue(""),
			RelayState: types.BoolUnknown(), CycleEnabled: types.BoolNull()},
	}

	raw, err := outletOverridesToSDK(overrides)
	if err != nil {
		t.Fatalf("outletOverridesToSDK: %v", err)
	}
	want := []string{
		`{"index":5,"name":"Dream Machine","relay_state":true,"cycle_enabled":false}`,
		`{"index":7,"name":""}`,
	}
	if len(raw) != len(want) {
		t.Fatalf("got %d overrides, want %d", len(raw), len(want))
	}
	for i := range want {
		if string(raw[i]) != want[i] {
			t.Errorf("override %d = %s, want %s", i, raw[i], want[i])
		}
	}
}
