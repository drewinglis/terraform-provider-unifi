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
