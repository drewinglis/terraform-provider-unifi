package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/resnickio/unifi-go-sdk/pkg/unifi"
)

var (
	_ resource.Resource                = &DeviceResource{}
	_ resource.ResourceWithImportState = &DeviceResource{}
)

type DeviceResource struct {
	client *AutoLoginClient
}

type DeviceResourceModel struct {
	ID                         types.String   `tfsdk:"id"`
	MAC                        types.String   `tfsdk:"mac"`
	Name                       types.String   `tfsdk:"name"`
	LedOverride                types.String   `tfsdk:"led_override"`
	LedOverrideColor           types.String   `tfsdk:"led_override_color"`
	LedOverrideColorBrightness types.Int64    `tfsdk:"led_override_color_brightness"`
	SNMPContact                types.String   `tfsdk:"snmp_contact"`
	SNMPLocation               types.String   `tfsdk:"snmp_location"`
	RadioOverrides             types.List     `tfsdk:"radio_overrides"`
	OutletOverrides            types.List     `tfsdk:"outlet_overrides"`
	MgmtNetworkID              types.String   `tfsdk:"mgmt_network_id"`
	ConfigNetwork              types.Object   `tfsdk:"config_network"`
	SiteID                     types.String   `tfsdk:"site_id"`
	Model                      types.String   `tfsdk:"model"`
	Type                       types.String   `tfsdk:"type"`
	IP                         types.String   `tfsdk:"ip"`
	Version                    types.String   `tfsdk:"version"`
	State                      types.Int64    `tfsdk:"state"`
	Adopted                    types.Bool     `tfsdk:"adopted"`
	Timeouts                   timeouts.Value `tfsdk:"timeouts"`
}

type RadioOverrideModel struct {
	Radio                 types.String `tfsdk:"radio"`
	Name                  types.String `tfsdk:"name"`
	Channel               types.Int64  `tfsdk:"channel"`
	ChannelWidth          types.Int64  `tfsdk:"channel_width"`
	TxPowerMode           types.String `tfsdk:"tx_power_mode"`
	TxPower               types.Int64  `tfsdk:"tx_power"`
	MinRSSIEnabled        types.Bool   `tfsdk:"min_rssi_enabled"`
	MinRSSI               types.Int64  `tfsdk:"min_rssi"`
	AntennaGain           types.Int64  `tfsdk:"antenna_gain"`
	VWireEnabled          types.Bool   `tfsdk:"vwire_enabled"`
	LoadBalanceEnabled    types.Bool   `tfsdk:"load_balance_enabled"`
	HardNoiseFloorEnabled types.Bool   `tfsdk:"hard_noise_floor_enabled"`
	SensLevelEnabled      types.Bool   `tfsdk:"sens_level_enabled"`
}

var radioOverrideAttrTypes = map[string]attr.Type{
	"radio":                    types.StringType,
	"name":                     types.StringType,
	"channel":                  types.Int64Type,
	"channel_width":            types.Int64Type,
	"tx_power_mode":            types.StringType,
	"tx_power":                 types.Int64Type,
	"min_rssi_enabled":         types.BoolType,
	"min_rssi":                 types.Int64Type,
	"antenna_gain":             types.Int64Type,
	"vwire_enabled":            types.BoolType,
	"load_balance_enabled":     types.BoolType,
	"hard_noise_floor_enabled": types.BoolType,
	"sens_level_enabled":       types.BoolType,
}

type OutletOverrideModel struct {
	Index        types.Int64  `tfsdk:"index"`
	Name         types.String `tfsdk:"name"`
	RelayState   types.Bool   `tfsdk:"relay_state"`
	CycleEnabled types.Bool   `tfsdk:"cycle_enabled"`
}

var outletOverrideAttrTypes = map[string]attr.Type{
	"index":         types.Int64Type,
	"name":          types.StringType,
	"relay_state":   types.BoolType,
	"cycle_enabled": types.BoolType,
}

// outletOverride is the controller's outlet_overrides element. The SDK carries
// these as raw JSON, so the provider owns the shape.
type outletOverride struct {
	Index        int     `json:"index"`
	Name         *string `json:"name,omitempty"`
	RelayState   *bool   `json:"relay_state,omitempty"`
	CycleEnabled *bool   `json:"cycle_enabled,omitempty"`
}

type DeviceConfigNetworkModel struct {
	Type           types.String `tfsdk:"type"`
	IP             types.String `tfsdk:"ip"`
	Netmask        types.String `tfsdk:"netmask"`
	Gateway        types.String `tfsdk:"gateway"`
	DNS1           types.String `tfsdk:"dns1"`
	DNS2           types.String `tfsdk:"dns2"`
	DNSSuffix      types.String `tfsdk:"dnssuffix"`
	BondingEnabled types.Bool   `tfsdk:"bonding_enabled"`
}

var deviceConfigNetworkAttrTypes = map[string]attr.Type{
	"type":            types.StringType,
	"ip":              types.StringType,
	"netmask":         types.StringType,
	"gateway":         types.StringType,
	"dns1":            types.StringType,
	"dns2":            types.StringType,
	"dnssuffix":       types.StringType,
	"bonding_enabled": types.BoolType,
}

func NewDeviceResource() resource.Resource {
	return &DeviceResource{}
}

func (r *DeviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *DeviceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi device's configuration. Devices are physically adopted hardware — " +
			"this resource manages writable settings (name, LED, SNMP) on an existing device. " +
			"Create looks up the device by MAC; delete removes it from Terraform state without forgetting the device.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the device.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of the device. Used to identify the device on create.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the device.",
				Optional:    true,
				Computed:    true,
			},
			"led_override": schema.StringAttribute{
				Description: "LED override mode. Valid values: 'default', 'on', 'off'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("default", "on", "off"),
				},
			},
			"led_override_color": schema.StringAttribute{
				Description: "LED override color (hex).",
				Optional:    true,
				Computed:    true,
			},
			"led_override_color_brightness": schema.Int64Attribute{
				Description: "LED override color brightness (0-100).",
				Optional:    true,
				Computed:    true,
			},
			"snmp_contact": schema.StringAttribute{
				Description: "SNMP contact string.",
				Optional:    true,
				Computed:    true,
			},
			"snmp_location": schema.StringAttribute{
				Description: "SNMP location string.",
				Optional:    true,
				Computed:    true,
			},
			"radio_overrides": schema.ListNestedAttribute{
				Description: "Radio configuration overrides for access points.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"radio": schema.StringAttribute{
							Description: "Radio band. Valid values: 'ng' (2.4GHz), 'na' (5GHz), '6e' (6GHz).",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("ng", "na", "6e"),
							},
						},
						"name": schema.StringAttribute{
							Description: "Radio name.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"channel": schema.Int64Attribute{
							Description: "Radio channel.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"channel_width": schema.Int64Attribute{
							Description: "Channel width (HT mode).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"tx_power_mode": schema.StringAttribute{
							Description: "Transmit power mode. Valid values: 'auto', 'low', 'medium', 'high', 'custom'.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "low", "medium", "high", "custom"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"tx_power": schema.Int64Attribute{
							Description: "Custom transmit power (dBm).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"min_rssi_enabled": schema.BoolAttribute{
							Description: "Enable minimum RSSI.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"min_rssi": schema.Int64Attribute{
							Description: "Minimum RSSI value.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"antenna_gain": schema.Int64Attribute{
							Description: "Antenna gain (dBi).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"vwire_enabled": schema.BoolAttribute{
							Description: "Enable virtual wire.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"load_balance_enabled": schema.BoolAttribute{
							Description: "Enable load balancing.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"hard_noise_floor_enabled": schema.BoolAttribute{
							Description: "Enable hard noise floor.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"sens_level_enabled": schema.BoolAttribute{
							Description: "Enable sensitivity level.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"outlet_overrides": schema.ListNestedAttribute{
				Description: "Outlet configuration overrides for PDUs and other devices with switchable power outlets.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.List{
					outletIndexRequired{},
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						// Required in practice (see outletIndexRequired), but declared
						// Optional+Computed: a Required attribute nested in a Computed one
						// makes Terraform propose null for it whenever outlet_overrides is
						// omitted, which plans a spurious update on every PDU.
						"index": schema.Int64Attribute{
							Description: "Outlet index (1-based). Required.",
							Optional:    true,
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Outlet name.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"relay_state": schema.BoolAttribute{
							Description: "Whether the outlet relay is on (outlet powered).",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"cycle_enabled": schema.BoolAttribute{
							Description: "Enable automatic power cycling of the outlet when the connected device stops responding.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"mgmt_network_id": schema.StringAttribute{
				Description: "The ID of the network (VLAN) the device uses for management traffic.",
				Optional:    true,
				Computed:    true,
			},
			"config_network": schema.SingleNestedAttribute{
				Description: "Management IP configuration of the device.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Management IP assignment. Valid values: 'dhcp', 'static'.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("dhcp", "static"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ip": schema.StringAttribute{
						Description: "Static management IP address.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"netmask": schema.StringAttribute{
						Description: "Static management netmask.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"gateway": schema.StringAttribute{
						Description: "Static management gateway.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"dns1": schema.StringAttribute{
						Description: "Primary DNS server for static management IP.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"dns2": schema.StringAttribute{
						Description: "Secondary DNS server for static management IP.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"dnssuffix": schema.StringAttribute{
						Description: "DNS suffix for static management IP.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"bonding_enabled": schema.BoolAttribute{
						Description: "Enable link aggregation (bonding) of the device's uplinks.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the device is adopted.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "The device model identifier.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The device type (uap, usw, ugw, uxg, udm).",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "The IP address of the device.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "The firmware version of the device.",
				Computed:    true,
			},
			"state": schema.Int64Attribute{
				Description: "The device state (0=offline, 1=connected, 2=pending adoption).",
				Computed:    true,
			},
			"adopted": schema.BoolAttribute{
				Description: "Whether the device is adopted.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *DeviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*AutoLoginClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AutoLoginClient, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *DeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	mac := plan.MAC.ValueString()
	device, err := r.client.GetDeviceByMAC(ctx, mac)
	if err != nil {
		handleSDKError(&resp.Diagnostics, err, "find", "device")
		return
	}

	updateDevice := r.planToSDK(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updateDevice.MAC = mac
	updated, err := r.client.UpdateDevice(ctx, device.ID, updateDevice)
	if err != nil {
		handleSDKError(&resp.Diagnostics, err, "update", "device")
		return
	}

	resp.Diagnostics.Append(r.sdkToState(ctx, updated, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, 2*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	device, err := r.client.GetDeviceByMAC(ctx, state.MAC.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		handleSDKError(&resp.Diagnostics, err, "read", "device")
		return
	}

	resp.Diagnostics.Append(r.sdkToState(ctx, device, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	var state DeviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateDevice := r.planToSDK(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updateDevice.MAC = plan.MAC.ValueString()

	updated, err := r.client.UpdateDevice(ctx, state.ID.ValueString(), updateDevice)
	if err != nil {
		handleSDKError(&resp.Diagnostics, err, "update", "device")
		return
	}

	resp.Diagnostics.Append(r.sdkToState(ctx, updated, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Devices are physical hardware — delete just removes from Terraform state.
	// The device remains adopted on the controller.
}

func (r *DeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	mac := req.ID

	device, err := r.client.GetDeviceByMAC(ctx, mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Could not find device with MAC %q: %s. Import requires the device MAC address (e.g., aa:bb:cc:dd:ee:ff).", mac, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), device.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), device.MAC)...)
}

func (r *DeviceResource) planToSDK(ctx context.Context, plan *DeviceResourceModel, diags *diag.Diagnostics) *unifi.DeviceConfig {
	device := &unifi.DeviceConfig{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		device.Name = plan.Name.ValueString()
	}
	if !plan.LedOverride.IsNull() && !plan.LedOverride.IsUnknown() {
		device.LedOverride = plan.LedOverride.ValueString()
	}
	if !plan.LedOverrideColor.IsNull() && !plan.LedOverrideColor.IsUnknown() {
		device.LedOverrideColor = plan.LedOverrideColor.ValueString()
	}
	if !plan.LedOverrideColorBrightness.IsNull() && !plan.LedOverrideColorBrightness.IsUnknown() {
		v := int(plan.LedOverrideColorBrightness.ValueInt64())
		device.LedOverrideColorBrightness = &v
	}
	if !plan.SNMPContact.IsNull() && !plan.SNMPContact.IsUnknown() {
		device.SNMPContact = plan.SNMPContact.ValueString()
	}
	if !plan.SNMPLocation.IsNull() && !plan.SNMPLocation.IsUnknown() {
		device.SNMPLocation = plan.SNMPLocation.ValueString()
	}

	if !plan.RadioOverrides.IsNull() && !plan.RadioOverrides.IsUnknown() {
		var overrides []RadioOverrideModel
		diags.Append(plan.RadioOverrides.ElementsAs(ctx, &overrides, false)...)
		for _, o := range overrides {
			ro := unifi.RadioOverride{
				Radio: o.Radio.ValueString(),
			}
			if !o.Name.IsNull() && !o.Name.IsUnknown() {
				ro.Name = o.Name.ValueString()
			}
			if !o.Channel.IsNull() && !o.Channel.IsUnknown() {
				ro.Channel = intPtr(o.Channel.ValueInt64())
			}
			if !o.ChannelWidth.IsNull() && !o.ChannelWidth.IsUnknown() {
				ro.ChannelWidth = intPtr(o.ChannelWidth.ValueInt64())
			}
			if !o.TxPowerMode.IsNull() && !o.TxPowerMode.IsUnknown() {
				ro.TxPowerMode = o.TxPowerMode.ValueString()
			}
			if !o.TxPower.IsNull() && !o.TxPower.IsUnknown() {
				ro.TxPower = intPtr(o.TxPower.ValueInt64())
			}
			if !o.MinRSSIEnabled.IsNull() && !o.MinRSSIEnabled.IsUnknown() {
				ro.MinRSSIEnabled = boolPtr(o.MinRSSIEnabled.ValueBool())
			}
			if !o.MinRSSI.IsNull() && !o.MinRSSI.IsUnknown() {
				ro.MinRSSI = intPtr(o.MinRSSI.ValueInt64())
			}
			if !o.AntennaGain.IsNull() && !o.AntennaGain.IsUnknown() {
				ro.AntennaGain = intPtr(o.AntennaGain.ValueInt64())
			}
			if !o.VWireEnabled.IsNull() && !o.VWireEnabled.IsUnknown() {
				ro.VWireEnabled = boolPtr(o.VWireEnabled.ValueBool())
			}
			if !o.LoadBalanceEnabled.IsNull() && !o.LoadBalanceEnabled.IsUnknown() {
				ro.LoadBalanceEnable = boolPtr(o.LoadBalanceEnabled.ValueBool())
			}
			if !o.HardNoiseFloorEnabled.IsNull() && !o.HardNoiseFloorEnabled.IsUnknown() {
				ro.HardNoiseFloor = boolPtr(o.HardNoiseFloorEnabled.ValueBool())
			}
			if !o.SensLevelEnabled.IsNull() && !o.SensLevelEnabled.IsUnknown() {
				ro.SensLevelEnabled = boolPtr(o.SensLevelEnabled.ValueBool())
			}
			device.RadioOverrides = append(device.RadioOverrides, ro)
		}
	}

	if !plan.OutletOverrides.IsNull() && !plan.OutletOverrides.IsUnknown() {
		var overrides []OutletOverrideModel
		diags.Append(plan.OutletOverrides.ElementsAs(ctx, &overrides, false)...)
		raw, err := outletOverridesToSDK(overrides)
		if err != nil {
			diags.AddError("Invalid Outlet Overrides", err.Error())
		}
		device.OutletOverrides = raw
	}

	if !plan.MgmtNetworkID.IsNull() && !plan.MgmtNetworkID.IsUnknown() {
		device.MgmtNetworkID = plan.MgmtNetworkID.ValueString()
	}

	if !plan.ConfigNetwork.IsNull() && !plan.ConfigNetwork.IsUnknown() {
		var cn DeviceConfigNetworkModel
		diags.Append(plan.ConfigNetwork.As(ctx, &cn, basetypes.ObjectAsOptions{})...)
		device.ConfigNetwork = &unifi.DeviceConfigNetwork{}
		if !cn.Type.IsNull() && !cn.Type.IsUnknown() {
			device.ConfigNetwork.Type = cn.Type.ValueString()
		}
		if !cn.IP.IsNull() && !cn.IP.IsUnknown() {
			device.ConfigNetwork.IP = cn.IP.ValueString()
		}
		if !cn.Netmask.IsNull() && !cn.Netmask.IsUnknown() {
			device.ConfigNetwork.Netmask = cn.Netmask.ValueString()
		}
		if !cn.Gateway.IsNull() && !cn.Gateway.IsUnknown() {
			device.ConfigNetwork.Gateway = cn.Gateway.ValueString()
		}
		if !cn.DNS1.IsNull() && !cn.DNS1.IsUnknown() {
			device.ConfigNetwork.DNS1 = cn.DNS1.ValueString()
		}
		if !cn.DNS2.IsNull() && !cn.DNS2.IsUnknown() {
			device.ConfigNetwork.DNS2 = cn.DNS2.ValueString()
		}
		if !cn.DNSSuffix.IsNull() && !cn.DNSSuffix.IsUnknown() {
			device.ConfigNetwork.DNSSuffix = cn.DNSSuffix.ValueString()
		}
		if !cn.BondingEnabled.IsNull() && !cn.BondingEnabled.IsUnknown() {
			device.ConfigNetwork.BondingEnabled = boolPtr(cn.BondingEnabled.ValueBool())
		}
	}

	return device
}

func outletOverridesToSDK(overrides []OutletOverrideModel) ([]json.RawMessage, error) {
	raw := make([]json.RawMessage, 0, len(overrides))
	for _, o := range overrides {
		oo := outletOverride{
			Index: int(o.Index.ValueInt64()),
		}
		if !o.Name.IsNull() && !o.Name.IsUnknown() {
			oo.Name = o.Name.ValueStringPointer()
		}
		if !o.RelayState.IsNull() && !o.RelayState.IsUnknown() {
			oo.RelayState = boolPtr(o.RelayState.ValueBool())
		}
		if !o.CycleEnabled.IsNull() && !o.CycleEnabled.IsUnknown() {
			oo.CycleEnabled = boolPtr(o.CycleEnabled.ValueBool())
		}
		b, err := json.Marshal(oo)
		if err != nil {
			return nil, fmt.Errorf("encoding outlet override %d: %w", oo.Index, err)
		}
		raw = append(raw, b)
	}
	return raw, nil
}

func outletOverridesFromSDK(raw []json.RawMessage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: outletOverrideAttrTypes}

	values := make([]attr.Value, 0, len(raw))
	for i, r := range raw {
		var oo outletOverride
		if err := json.Unmarshal(r, &oo); err != nil {
			diags.AddError("Invalid Outlet Override", fmt.Sprintf("decoding outlet override %d: %s", i, err))
			continue
		}
		obj, d := types.ObjectValue(outletOverrideAttrTypes, map[string]attr.Value{
			"index":         types.Int64Value(int64(oo.Index)),
			"name":          types.StringPointerValue(oo.Name),
			"relay_state":   types.BoolPointerValue(oo.RelayState),
			"cycle_enabled": types.BoolPointerValue(oo.CycleEnabled),
		})
		diags.Append(d...)
		values = append(values, obj)
	}
	list, d := types.ListValue(objType, values)
	diags.Append(d...)
	return list, diags
}

func (r *DeviceResource) sdkToState(ctx context.Context, device *unifi.DeviceConfig, state *DeviceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(device.ID)
	state.MAC = types.StringValue(device.MAC)
	state.SiteID = types.StringValue(device.SiteID)
	state.Name = stringValueOrNull(device.Name)
	state.Model = stringValueOrNull(device.Model)
	state.Type = stringValueOrNull(device.Type)
	state.IP = stringValueOrNull(device.IP)
	state.Version = stringValueOrNull(device.Version)
	state.LedOverride = stringValueOrNull(device.LedOverride)
	state.LedOverrideColor = stringValueOrNull(device.LedOverrideColor)
	state.SNMPContact = stringValueOrNull(device.SNMPContact)
	state.SNMPLocation = stringValueOrNull(device.SNMPLocation)
	state.MgmtNetworkID = stringValueOrNull(device.MgmtNetworkID)

	if cn := device.ConfigNetwork; cn != nil {
		obj, d := types.ObjectValue(deviceConfigNetworkAttrTypes, map[string]attr.Value{
			"type":            stringValueOrNull(cn.Type),
			"ip":              stringValueOrNull(cn.IP),
			"netmask":         stringValueOrNull(cn.Netmask),
			"gateway":         stringValueOrNull(cn.Gateway),
			"dns1":            stringValueOrNull(cn.DNS1),
			"dns2":            stringValueOrNull(cn.DNS2),
			"dnssuffix":       stringValueOrNull(cn.DNSSuffix),
			"bonding_enabled": types.BoolPointerValue(cn.BondingEnabled),
		})
		diags.Append(d...)
		state.ConfigNetwork = obj
	} else {
		state.ConfigNetwork = types.ObjectNull(deviceConfigNetworkAttrTypes)
	}

	if device.LedOverrideColorBrightness != nil {
		state.LedOverrideColorBrightness = types.Int64Value(int64(*device.LedOverrideColorBrightness))
	} else {
		state.LedOverrideColorBrightness = types.Int64Null()
	}

	if device.State != nil {
		state.State = types.Int64Value(int64(*device.State))
	} else {
		state.State = types.Int64Null()
	}

	state.Adopted = types.BoolValue(derefBool(device.Adopted))

	if len(device.RadioOverrides) > 0 {
		overrideValues := make([]attr.Value, len(device.RadioOverrides))
		for i, ro := range device.RadioOverrides {
			vals := map[string]attr.Value{
				"radio":                    stringValueOrNull(ro.Radio),
				"name":                     stringValueOrNull(ro.Name),
				"tx_power_mode":            stringValueOrNull(ro.TxPowerMode),
				"min_rssi_enabled":         types.BoolValue(derefBool(ro.MinRSSIEnabled)),
				"vwire_enabled":            types.BoolValue(derefBool(ro.VWireEnabled)),
				"load_balance_enabled":     types.BoolValue(derefBool(ro.LoadBalanceEnable)),
				"hard_noise_floor_enabled": types.BoolValue(derefBool(ro.HardNoiseFloor)),
				"sens_level_enabled":       types.BoolValue(derefBool(ro.SensLevelEnabled)),
			}
			if ro.Channel != nil {
				vals["channel"] = types.Int64Value(int64(*ro.Channel))
			} else {
				vals["channel"] = types.Int64Null()
			}
			if ro.ChannelWidth != nil {
				vals["channel_width"] = types.Int64Value(int64(*ro.ChannelWidth))
			} else {
				vals["channel_width"] = types.Int64Null()
			}
			if ro.TxPower != nil {
				vals["tx_power"] = types.Int64Value(int64(*ro.TxPower))
			} else {
				vals["tx_power"] = types.Int64Null()
			}
			if ro.MinRSSI != nil {
				vals["min_rssi"] = types.Int64Value(int64(*ro.MinRSSI))
			} else {
				vals["min_rssi"] = types.Int64Null()
			}
			if ro.AntennaGain != nil {
				vals["antenna_gain"] = types.Int64Value(int64(*ro.AntennaGain))
			} else {
				vals["antenna_gain"] = types.Int64Null()
			}

			obj, d := types.ObjectValue(radioOverrideAttrTypes, vals)
			diags.Append(d...)
			overrideValues[i] = obj
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: radioOverrideAttrTypes}, overrideValues)
		diags.Append(d...)
		state.RadioOverrides = list
	} else {
		state.RadioOverrides = types.ListValueMust(types.ObjectType{AttrTypes: radioOverrideAttrTypes}, []attr.Value{})
	}

	outlets, d := outletOverridesFromSDK(device.OutletOverrides)
	diags.Append(d...)
	state.OutletOverrides = outlets

	return diags
}

// outletIndexRequired errors when a configured outlet override has no index.
type outletIndexRequired struct{}

func (v outletIndexRequired) Description(ctx context.Context) string {
	return "each outlet override must set index"
}

func (v outletIndexRequired) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v outletIndexRequired) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for i, elem := range req.ConfigValue.Elements() {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsUnknown() {
			continue
		}
		if index, ok := obj.Attributes()["index"].(types.Int64); ok && index.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i).AtName("index"),
				"Missing outlet index",
				"Each outlet_overrides entry must set index.",
			)
		}
	}
}
