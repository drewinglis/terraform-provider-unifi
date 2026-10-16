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
