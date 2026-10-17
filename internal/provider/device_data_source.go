package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/resnickio/unifi-go-sdk/pkg/unifi"
)

var _ datasource.DataSource = &DeviceDataSource{}

type DeviceDataSource struct {
	client *AutoLoginClient
}

type DeviceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	MAC             types.String `tfsdk:"mac"`
	Name            types.String `tfsdk:"name"`
	Model           types.String `tfsdk:"model"`
	Type            types.String `tfsdk:"type"`
	Version         types.String `tfsdk:"version"`
	IP              types.String `tfsdk:"ip"`
	Adopted         types.Bool   `tfsdk:"adopted"`
	State           types.Int64  `tfsdk:"state"`
	MgmtNetworkID   types.String `tfsdk:"mgmt_network_id"`
	ConfigNetwork   types.Object `tfsdk:"config_network"`
	OutletOverrides types.List   `tfsdk:"outlet_overrides"`
}

var deviceDataSourceConfigNetworkAttrTypes = map[string]attr.Type{
	"type":            types.StringType,
	"ip":              types.StringType,
	"netmask":         types.StringType,
	"gateway":         types.StringType,
	"dns1":            types.StringType,
	"dns2":            types.StringType,
	"dnssuffix":       types.StringType,
	"bonding_enabled": types.BoolType,
}

var deviceDataSourceOutletAttrTypes = map[string]attr.Type{
	"index":         types.Int64Type,
	"name":          types.StringType,
	"relay_state":   types.BoolType,
	"cycle_enabled": types.BoolType,
}

func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about an existing UniFi device (switch, access point, gateway). Lookup by MAC address or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the device.",
				Computed:    true,
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of the device. Specify either mac or name.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the device. Specify either mac or name.",
				Optional:    true,
				Computed:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the device exists.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "The model of the device (e.g., 'USW-Pro-24-PoE', 'U6-Pro').",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of device: 'usw' (switch), 'uap' (access point), 'ugw' (gateway), 'udm' (dream machine), 'uxg' (next-gen gateway).",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "The firmware version of the device.",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "The IP address of the device.",
				Computed:    true,
			},
			"adopted": schema.BoolAttribute{
				Description: "Whether the device has been adopted.",
				Computed:    true,
			},
			"state": schema.Int64Attribute{
				Description: "The state of the device: 0=offline, 1=connected, 2=pending.",
				Computed:    true,
			},
			"mgmt_network_id": schema.StringAttribute{
				Description: "The ID of the network (VLAN) the device uses for management traffic.",
				Computed:    true,
			},
			"config_network": schema.SingleNestedAttribute{
				Description: "Management IP configuration of the device.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Management IP assignment: 'dhcp' or 'static'.",
						Computed:    true,
					},
					"ip": schema.StringAttribute{
						Description: "Static management IP address.",
						Computed:    true,
					},
					"netmask": schema.StringAttribute{
						Description: "Static management netmask.",
						Computed:    true,
					},
					"gateway": schema.StringAttribute{
						Description: "Static management gateway.",
						Computed:    true,
					},
					"dns1": schema.StringAttribute{
						Description: "Primary DNS server for static management IP.",
						Computed:    true,
					},
					"dns2": schema.StringAttribute{
						Description: "Secondary DNS server for static management IP.",
						Computed:    true,
					},
					"dnssuffix": schema.StringAttribute{
						Description: "DNS suffix for static management IP.",
						Computed:    true,
					},
					"bonding_enabled": schema.BoolAttribute{
						Description: "Whether link aggregation (bonding) of the device's uplinks is enabled.",
						Computed:    true,
					},
				},
			},
			"outlet_overrides": schema.ListNestedAttribute{
				Description: "Outlet configuration overrides for PDUs and other devices with switchable power outlets.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							Description: "Outlet index (1-based).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Outlet name.",
							Computed:    true,
						},
						"relay_state": schema.BoolAttribute{
							Description: "Whether the outlet relay is on (outlet powered).",
							Computed:    true,
						},
						"cycle_enabled": schema.BoolAttribute{
							Description: "Whether automatic power cycling of the outlet is enabled.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*AutoLoginClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *AutoLoginClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DeviceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasMAC := !config.MAC.IsNull() && config.MAC.ValueString() != ""
	hasName := !config.Name.IsNull() && config.Name.ValueString() != ""

	if !hasMAC && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'mac' or 'name' must be specified to look up a device.",
		)
		return
	}

	var device *unifi.DeviceConfig
	var err error

	if hasMAC {
		mac := strings.ToLower(strings.ReplaceAll(config.MAC.ValueString(), "-", ":"))
		device, err = d.client.GetDeviceByMAC(ctx, mac)
		if err != nil {
			handleSDKError(&resp.Diagnostics, err, "read", "device")
			return
		}
	} else {
		devices, err := d.client.ListDevices(ctx)
		if err != nil {
			handleSDKError(&resp.Diagnostics, err, "list", "devices")
			return
		}

		searchName := config.Name.ValueString()
		for i := range devices.NetworkDevices {
			if devices.NetworkDevices[i].Name == searchName {
				mac := devices.NetworkDevices[i].MAC
				device, err = d.client.GetDeviceByMAC(ctx, mac)
				if err != nil {
					handleSDKError(&resp.Diagnostics, err, "read", "device")
					return
				}
				break
			}
		}

		if device == nil {
			resp.Diagnostics.AddError(
				"Device Not Found",
				fmt.Sprintf("No device found with name '%s'.", searchName),
			)
			return
		}
	}

	resp.Diagnostics.Append(d.sdkToState(ctx, device, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func (d *DeviceDataSource) sdkToState(ctx context.Context, device *unifi.DeviceConfig, state *DeviceDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(device.ID)
	state.SiteID = stringValueOrNull(device.SiteID)
	state.MAC = types.StringValue(device.MAC)
	state.Name = stringValueOrNull(device.Name)
	state.Model = stringValueOrNull(device.Model)
	state.Type = stringValueOrNull(device.Type)
	state.Version = stringValueOrNull(device.Version)
	state.IP = stringValueOrNull(device.IP)

	if device.Adopted != nil {
		state.Adopted = types.BoolValue(*device.Adopted)
	} else {
		state.Adopted = types.BoolNull()
	}

	if device.State != nil {
		state.State = types.Int64Value(int64(*device.State))
	} else {
		state.State = types.Int64Null()
	}

	state.MgmtNetworkID = stringValueOrNull(device.MgmtNetworkID)

	if cn := device.ConfigNetwork; cn != nil {
		obj, objDiags := types.ObjectValue(deviceDataSourceConfigNetworkAttrTypes, map[string]attr.Value{
			"type":            stringValueOrNull(cn.Type),
			"ip":              stringValueOrNull(cn.IP),
			"netmask":         stringValueOrNull(cn.Netmask),
			"gateway":         stringValueOrNull(cn.Gateway),
			"dns1":            stringValueOrNull(cn.DNS1),
			"dns2":            stringValueOrNull(cn.DNS2),
			"dnssuffix":       stringValueOrNull(cn.DNSSuffix),
			"bonding_enabled": types.BoolPointerValue(cn.BondingEnabled),
		})
		diags.Append(objDiags...)
		state.ConfigNetwork = obj
	} else {
		state.ConfigNetwork = types.ObjectNull(deviceDataSourceConfigNetworkAttrTypes)
	}

	outlets := make([]attr.Value, 0, len(device.OutletOverrides))
	for i, raw := range device.OutletOverrides {
		var oo struct {
			Index        int     `json:"index"`
			Name         *string `json:"name"`
			RelayState   *bool   `json:"relay_state"`
			CycleEnabled *bool   `json:"cycle_enabled"`
		}
		if err := json.Unmarshal(raw, &oo); err != nil {
			diags.AddError("Invalid Outlet Override", fmt.Sprintf("decoding outlet override %d: %s", i, err))
			continue
		}
		obj, objDiags := types.ObjectValue(deviceDataSourceOutletAttrTypes, map[string]attr.Value{
			"index":         types.Int64Value(int64(oo.Index)),
			"name":          types.StringPointerValue(oo.Name),
			"relay_state":   types.BoolPointerValue(oo.RelayState),
			"cycle_enabled": types.BoolPointerValue(oo.CycleEnabled),
		})
		diags.Append(objDiags...)
		outlets = append(outlets, obj)
	}
	list, listDiags := types.ListValue(types.ObjectType{AttrTypes: deviceDataSourceOutletAttrTypes}, outlets)
	diags.Append(listDiags...)
	state.OutletOverrides = list

	return diags
}
