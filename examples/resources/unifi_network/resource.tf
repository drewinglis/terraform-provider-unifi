# Create a VLAN network with DHCP
resource "unifi_network" "iot" {
  name         = "IoT Network"
  purpose      = "corporate"
  vlan_id      = 100
  subnet       = "10.0.100.1/24"
  dhcp_enabled = true
  dhcp_start   = "10.0.100.10"
  dhcp_stop    = "10.0.100.254"
  dhcp_dns     = ["8.8.8.8", "8.8.4.4"]
  domain_name  = "iot.local"

  # Let the controller size the DHCP range, and keep this network off the
  # LTE failover connection.
  auto_scale_enabled = true
  lte_lan_enabled    = false
}

# Create a VLAN-only network (no routing/DHCP)
resource "unifi_network" "vlan_only" {
  name    = "External VLAN"
  purpose = "vlan-only"
  vlan_id = 200
}

# Manage the settings of an existing WAN network (WAN networks are created by
# the controller, so import them). LAN settings such as DHCP don't apply.
resource "unifi_network" "internet_2" {
  name                  = "Internet 2"
  purpose               = "wan"
  wan_type              = "dhcp"
  wan_network_group     = "WAN2"
  wan_load_balance_type = "failover-only"
  wan_failover_priority = 2
}

# An isolated network: its clients can reach the internet but no other local network
resource "unifi_network" "work_jail" {
  name                      = "Work Jail"
  purpose                   = "corporate"
  subnet                    = "10.0.12.1/24"
  vlan_id                   = 12
  network_isolation_enabled = true
}
