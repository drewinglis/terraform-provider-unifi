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

# Name and configure PDU outlets
resource "unifi_device" "pdu" {
  mac  = "aa:bb:cc:dd:ee:03"
  name = "Rack PDU"

  outlet_overrides = [
    {
      index         = 1
      name          = "Gateway"
      relay_state   = true
      cycle_enabled = false
    },
    {
      index         = 2
      name          = "Core Switch"
      relay_state   = true
      cycle_enabled = true
    },
  ]
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
