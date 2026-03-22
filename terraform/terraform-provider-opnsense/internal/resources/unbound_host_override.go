package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &UnboundHostOverrideResource{}

type UnboundHostOverrideResource struct {
	client *OPNsenseClient
}

type UnboundHostOverrideModel struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	Domain   types.String `tfsdk:"domain"`
	Server   types.String `tfsdk:"server"`
	RR       types.String `tfsdk:"rr"`
	Enabled  types.String `tfsdk:"enabled"`
}

func NewUnboundHostOverrideResource() resource.Resource {
	return &UnboundHostOverrideResource{}
}

func (r *UnboundHostOverrideResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_unbound_host_override"
}

func (r *UnboundHostOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Unbound DNS host override in OPNsense. Automatically triggers Unbound reconfigure after changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the host override.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "Hostname (e.g. myapp).",
				Required:    true,
			},
			"domain": schema.StringAttribute{
				Description: "Domain (e.g. lab.anydef.de).",
				Required:    true,
			},
			"server": schema.StringAttribute{
				Description: "IP address the hostname should resolve to.",
				Required:    true,
			},
			"rr": schema.StringAttribute{
				Description: "DNS record type (A, AAAA, MX).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("A"),
			},
			"enabled": schema.StringAttribute{
				Description: "Enable host override (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1"),
			},
		},
	}
}

func (r *UnboundHostOverrideResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*OPNsenseClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *OPNsenseClient")
		return
	}
	r.client = client
}

func (r *UnboundHostOverrideResource) buildPayload(plan *UnboundHostOverrideModel) map[string]interface{} {
	return map[string]interface{}{
		"host": map[string]interface{}{
			"enabled":  plan.Enabled.ValueString(),
			"hostname": plan.Hostname.ValueString(),
			"domain":   plan.Domain.ValueString(),
			"rr":       plan.RR.ValueString(),
			"server":   plan.Server.ValueString(),
		},
	}
}

func (r *UnboundHostOverrideResource) reconfigureUnbound(ctx context.Context) {
	tflog.Info(ctx, "Triggering Unbound reconfigure")
	if err := r.client.Post(ctx, "/api/unbound/service/reconfigure"); err != nil {
		tflog.Warn(ctx, "Unbound reconfigure failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (r *UnboundHostOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UnboundHostOverrideModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.client.Create(ctx, "/api/unbound/settings/addHostOverride", r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Unbound host override", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	r.reconfigureUnbound(ctx)
}

func (r *UnboundHostOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UnboundHostOverrideModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/unbound/settings/getHostOverride/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading Unbound host override", err.Error())
		return
	}

	// result parsed below
	result, err := ParseResponse(body)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing Unbound host override response", err.Error())
		return
	}

	data, ok := result["host"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing Unbound host override response", "missing 'host' key")
		return
	}

	state.Hostname = types.StringValue(extractStringField(data, "hostname"))
	state.Domain = types.StringValue(extractStringField(data, "domain"))
	state.Server = types.StringValue(extractStringField(data, "server"))
	state.RR = types.StringValue(extractExpressionValue(data, "rr"))
	state.Enabled = types.StringValue(extractStringField(data, "enabled"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UnboundHostOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UnboundHostOverrideModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UnboundHostOverrideModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/unbound/settings/setHostOverride/%s", state.ID.ValueString()), r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Unbound host override", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	r.reconfigureUnbound(ctx)
}

func (r *UnboundHostOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UnboundHostOverrideModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/unbound/settings/delHostOverride/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Unbound host override", err.Error())
		return
	}

	r.reconfigureUnbound(ctx)
}
