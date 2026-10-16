# Manage an adopted device's settings
resource "unifi_device" "switch" {
  mac  = "aa:bb:cc:dd:ee:01"
  name = "Office Switch"
}

# Manage device with LED override
resource "unifi_device" "ap" {
  mac          = "aa:bb:cc:dd:ee:02"
  name         = "Lobby AP"
  led_override = "off"
}

# Manage a switch on a dedicated management VLAN with a static IP
resource "unifi_device" "core_switch" {
  mac             = "aa:bb:cc:dd:ee:04"
  name            = "Core Switch"
  mgmt_network_id = unifi_network.management.id

  config_network = {
    type    = "static"
    ip      = "10.0.10.2"
    netmask = "255.255.255.0"
    gateway = "10.0.10.1"
    dns1    = "10.0.10.1"
  }
}
