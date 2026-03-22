package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HAProxyBackendResource{}

type HAProxyBackendResource struct {
	client *OPNsenseClient
}

type HAProxyBackendModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Mode              types.String `tfsdk:"mode"`
	Algorithm         types.String `tfsdk:"algorithm"`
	LinkedServers     types.String `tfsdk:"linked_servers"`
	HTTP2Enabled      types.String `tfsdk:"http2_enabled"`
	Persistence       types.String `tfsdk:"persistence"`
	StickinessPattern types.String `tfsdk:"stickiness_pattern"`
	StickinessExpire  types.String `tfsdk:"stickiness_expire"`
	StickinessSize    types.String `tfsdk:"stickiness_size"`
	TuningHTTPReuse   types.String `tfsdk:"tuning_httpreuse"`
	HealthCheckEnabled types.String `tfsdk:"health_check_enabled"`
	HealthCheck       types.String `tfsdk:"health_check"`
	Enabled           types.String `tfsdk:"enabled"`
}

func NewHAProxyBackendResource() resource.Resource {
	return &HAProxyBackendResource{}
}

func (r *HAProxyBackendResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_backend"
}

func (r *HAProxyBackendResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy backend pool in OPNsense.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the backend.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Backend name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Backend description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"mode": schema.StringAttribute{
				Description: "Backend mode (http or tcp).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("http"),
			},
			"algorithm": schema.StringAttribute{
				Description: "Load balancing algorithm.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("source"),
			},
			"linked_servers": schema.StringAttribute{
				Description: "Comma-separated list of server UUIDs.",
				Required:    true,
			},
			"http2_enabled": schema.StringAttribute{
				Description: "Enable HTTP/2 (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"persistence": schema.StringAttribute{
				Description: "Session persistence mode.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("sticktable"),
			},
			"stickiness_pattern": schema.StringAttribute{
				Description: "Stickiness pattern.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("sourceipv4"),
			},
			"stickiness_expire": schema.StringAttribute{
				Description: "Stickiness expiration.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("30m"),
			},
			"stickiness_size": schema.StringAttribute{
				Description: "Stickiness table size.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("50k"),
			},
			"tuning_httpreuse": schema.StringAttribute{
				Description: "HTTP connection reuse mode.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("safe"),
			},
			"health_check_enabled": schema.StringAttribute{
				Description: "Enable health checking (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"health_check": schema.StringAttribute{
				Description: "Health check type (e.g. HTTP, TCP, or empty).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.StringAttribute{
				Description: "Enable backend (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1"),
			},
		},
	}
}

func (r *HAProxyBackendResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HAProxyBackendResource) buildPayload(plan *HAProxyBackendModel) map[string]interface{} {
	return map[string]interface{}{
		"backend": map[string]interface{}{
			"enabled":            plan.Enabled.ValueString(),
			"name":               plan.Name.ValueString(),
			"description":        plan.Description.ValueString(),
			"mode":               plan.Mode.ValueString(),
			"algorithm":          plan.Algorithm.ValueString(),
			"linkedServers":      plan.LinkedServers.ValueString(),
			"http2Enabled":       plan.HTTP2Enabled.ValueString(),
			"persistence":        plan.Persistence.ValueString(),
			"stickiness_pattern": plan.StickinessPattern.ValueString(),
			"stickiness_expire":  plan.StickinessExpire.ValueString(),
			"stickiness_size":    plan.StickinessSize.ValueString(),
			"tuning_httpreuse":   plan.TuningHTTPReuse.ValueString(),
			"healthCheckEnabled": plan.HealthCheckEnabled.ValueString(),
			"healthCheck":        plan.HealthCheck.ValueString(),
		},
	}
}

func (r *HAProxyBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyBackendModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.client.Create(ctx, "/api/haproxy/settings/addBackend", r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy backend", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyBackendModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getBackend/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy backend", err.Error())
		return
	}

	// result parsed below
	result, err := ParseResponse(body)
	if errors.Is(err, ErrResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	data, ok := result["backend"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing HAProxy backend response", "missing 'backend' key")
		return
	}

	state.Name = types.StringValue(extractStringField(data, "name"))
	state.Description = types.StringValue(extractStringField(data, "description"))
	state.Mode = types.StringValue(extractStringField(data, "mode"))
	state.Algorithm = types.StringValue(extractStringField(data, "algorithm"))
	state.LinkedServers = types.StringValue(extractSelectedUUIDs(data, "linkedServers"))
	state.HTTP2Enabled = types.StringValue(extractStringField(data, "http2Enabled"))
	state.Persistence = types.StringValue(extractStringField(data, "persistence"))
	state.StickinessPattern = types.StringValue(extractStringField(data, "stickiness_pattern"))
	state.StickinessExpire = types.StringValue(extractStringField(data, "stickiness_expire"))
	state.StickinessSize = types.StringValue(extractStringField(data, "stickiness_size"))
	state.TuningHTTPReuse = types.StringValue(extractStringField(data, "tuning_httpreuse"))
	state.HealthCheckEnabled = types.StringValue(extractStringField(data, "healthCheckEnabled"))
	state.HealthCheck = types.StringValue(extractStringField(data, "healthCheck"))
	state.Enabled = types.StringValue(extractStringField(data, "enabled"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HAProxyBackendModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyBackendModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setBackend/%s", state.ID.ValueString()), r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy backend", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyBackendModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/haproxy/settings/delBackend/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy backend", err.Error())
		return
	}
}
