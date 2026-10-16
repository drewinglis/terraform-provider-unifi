package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceResource_basic(t *testing.T) {
	mac := testAccGetFirstDeviceMAC(t)
	if mac == "" {
		t.Skip("No device available for testing")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDevice(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceResourceConfig_basic(mac),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_device.test", "id"),
					resource.TestCheckResourceAttr("unifi_device.test", "mac", mac),
					resource.TestCheckResourceAttrSet("unifi_device.test", "type"),
					resource.TestCheckResourceAttrSet("unifi_device.test", "model"),
				),
			},
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateId:           mac,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccDeviceResource_updateName(t *testing.T) {
	mac := testAccGetFirstDeviceMAC(t)
	if mac == "" {
		t.Skip("No device available for testing")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDevice(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceResourceConfig_withName(mac, "tf-acc-test-device"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test", "name", "tf-acc-test-device"),
				),
			},
			{
				Config: testAccDeviceResourceConfig_withName(mac, "tf-acc-test-device-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test", "name", "tf-acc-test-device-updated"),
				),
			},
		},
	})
}

func TestAccDeviceResource_outletOverrides(t *testing.T) {
	mac := testAccGetFirstOutletDeviceMAC(t)
	if mac == "" {
		t.Skip("No device with outlets available for testing")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDevice(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceResourceConfig_outletOverride(mac, "tf-acc-test-outlet", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.#", "1"),
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.index", "1"),
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.name", "tf-acc-test-outlet"),
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.relay_state", "true"),
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.cycle_enabled", "false"),
				),
			},
			{
				Config: testAccDeviceResourceConfig_outletOverride(mac, "tf-acc-test-outlet-updated", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.name", "tf-acc-test-outlet-updated"),
					resource.TestCheckResourceAttr("unifi_device.test", "outlet_overrides.0.cycle_enabled", "true"),
				),
			},
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateId:           mac,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccDeviceResource_mgmtNetwork(t *testing.T) {
	mac := testAccGetFirstSwitchMAC(t)
	if mac == "" {
		t.Skip("No switch available for testing")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckSwitch(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceResourceConfig_configNetwork(mac),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_device.test", "config_network.type", "dhcp"),
					resource.TestCheckResourceAttrSet("unifi_device.test", "mgmt_network_id"),
				),
			},
			{
				ResourceName:            "unifi_device.test",
				ImportState:             true,
				ImportStateId:           mac,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func testAccDeviceResourceConfig_basic(mac string) string {
	return fmt.Sprintf(`
%s

resource "unifi_device" "test" {
  mac = %q
}
`, testAccProviderConfig, mac)
}

func testAccDeviceResourceConfig_withName(mac, name string) string {
	return fmt.Sprintf(`
%s

resource "unifi_device" "test" {
  mac  = %q
  name = %q
}
`, testAccProviderConfig, mac, name)
}

func testAccDeviceResourceConfig_outletOverride(mac, name string, cycleEnabled bool) string {
	return fmt.Sprintf(`
%s

resource "unifi_device" "test" {
  mac = %q

  outlet_overrides = [{
    index         = 1
    name          = %q
    relay_state   = true
    cycle_enabled = %t
  }]
}
`, testAccProviderConfig, mac, name, cycleEnabled)
}

func testAccDeviceResourceConfig_configNetwork(mac string) string {
	return fmt.Sprintf(`
%s

data "unifi_network" "default" {
  name = "Default"
}

resource "unifi_device" "test" {
  mac             = %q
  mgmt_network_id = data.unifi_network.default.id

  config_network = {
    type = "dhcp"
  }
}
`, testAccProviderConfig, mac)
}
