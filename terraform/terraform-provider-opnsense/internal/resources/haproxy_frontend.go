package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HAProxyFrontendResource{}
var _ resource.ResourceWithImportState = &HAProxyFrontendResource{}

type HAProxyFrontendResource struct {
	client *OPNsenseClient
}

type HAProxyFrontendModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Bind                  types.String `tfsdk:"bind"`
	BindOptions           types.String `tfsdk:"bind_options"`
	Mode                  types.String `tfsdk:"mode"`
	DefaultBackend        types.String `tfsdk:"default_backend"`
	SSLEnabled            types.String `tfsdk:"ssl_enabled"`
	SSLCertificates       types.String `tfsdk:"ssl_certificates"`
	SSLDefaultCertificate types.String `tfsdk:"ssl_default_certificate"`
	SSLHSTSEnabled        types.String `tfsdk:"ssl_hsts_enabled"`
	SSLHSTSMaxAge         types.String `tfsdk:"ssl_hsts_max_age"`
	SSLMinVersion         types.String `tfsdk:"ssl_min_version"`
	HTTP2Enabled          types.String `tfsdk:"http2_enabled"`
	HTTP2EnabledNonTLS    types.String `tfsdk:"http2_enabled_nontls"`
	AdvertisedProtocols   types.String `tfsdk:"advertised_protocols"`
	ForwardFor            types.String `tfsdk:"forward_for"`
	ConnectionBehaviour   types.String `tfsdk:"connection_behaviour"`
	LinkedActions         types.String `tfsdk:"linked_actions"`
	Enabled               types.String `tfsdk:"enabled"`
}

func NewHAProxyFrontendResource() resource.Resource {
	return &HAProxyFrontendResource{}
}

func (r *HAProxyFrontendResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_frontend"
}

func (r *HAProxyFrontendResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy frontend in OPNsense.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the frontend.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Frontend name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Frontend description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"bind": schema.StringAttribute{
				Description: "Bind address and port (e.g. 0.0.0.0:443).",
				Required:    true,
			},
			"bind_options": schema.StringAttribute{
				Description: "Additional bind options.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"mode": schema.StringAttribute{
				Description: "Frontend mode (http or tcp).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("http"),
			},
			"default_backend": schema.StringAttribute{
				Description: "UUID of the default backend.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"ssl_enabled": schema.StringAttribute{
				Description: "Enable SSL offloading (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"ssl_certificates": schema.StringAttribute{
				Description: "Comma-separated list of SSL certificate UUIDs.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"ssl_default_certificate": schema.StringAttribute{
				Description: "UUID of the default SSL certificate.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"ssl_hsts_enabled": schema.StringAttribute{
				Description: "Enable HSTS (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"ssl_hsts_max_age": schema.StringAttribute{
				Description: "HSTS max-age in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("15768000"),
			},
			"ssl_min_version": schema.StringAttribute{
				Description: "Minimum SSL/TLS version.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"http2_enabled": schema.StringAttribute{
				Description: "Enable HTTP/2 (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"http2_enabled_nontls": schema.StringAttribute{
				Description: "Enable HTTP/2 without TLS (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"advertised_protocols": schema.StringAttribute{
				Description: "ALPN advertised protocols (e.g. h2,http11).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("h2,http11"),
			},
			"forward_for": schema.StringAttribute{
				Description: "Enable X-Forwarded-For header (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"connection_behaviour": schema.StringAttribute{
				Description: "Connection behaviour (e.g. http-keep-alive).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("http-keep-alive"),
			},
			"linked_actions": schema.StringAttribute{
				Description: "Comma-separated list of action UUIDs.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.StringAttribute{
				Description: "Enable frontend (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1"),
			},
		},
	}
}

func (r *HAProxyFrontendResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HAProxyFrontendResource) buildPayload(plan *HAProxyFrontendModel) map[string]interface{} {
	return map[string]interface{}{
		"frontend": map[string]interface{}{
			"enabled":                plan.Enabled.ValueString(),
			"name":                   plan.Name.ValueString(),
			"description":            plan.Description.ValueString(),
			"bind":                   plan.Bind.ValueString(),
			"bindOptions":            plan.BindOptions.ValueString(),
			"mode":                   plan.Mode.ValueString(),
			"defaultBackend":         plan.DefaultBackend.ValueString(),
			"ssl_enabled":            plan.SSLEnabled.ValueString(),
			"ssl_certificates":       plan.SSLCertificates.ValueString(),
			"ssl_default_certificate": plan.SSLDefaultCertificate.ValueString(),
			"ssl_hstsEnabled":        plan.SSLHSTSEnabled.ValueString(),
			"ssl_hstsMaxAge":         plan.SSLHSTSMaxAge.ValueString(),
			"ssl_minVersion":         plan.SSLMinVersion.ValueString(),
			"http2Enabled":           plan.HTTP2Enabled.ValueString(),
			"http2Enabled_nontls":    plan.HTTP2EnabledNonTLS.ValueString(),
			"advertised_protocols":   plan.AdvertisedProtocols.ValueString(),
			"forwardFor":             plan.ForwardFor.ValueString(),
			"connectionBehaviour":    plan.ConnectionBehaviour.ValueString(),
			"linkedActions":          plan.LinkedActions.ValueString(),
		},
	}
}

func (r *HAProxyFrontendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyFrontendModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.client.Create(ctx, "/api/haproxy/settings/addFrontend", r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy frontend", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyFrontendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyFrontendModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getFrontend/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy frontend", err.Error())
		return
	}

	result, err := ParseResponse(body)
	if errors.Is(err, ErrResourceNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	data, ok := result["frontend"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing HAProxy frontend response", "missing 'frontend' key")
		return
	}

	state.Name = types.StringValue(extractStringField(data, "name"))
	state.Description = types.StringValue(extractStringField(data, "description"))
	state.Bind = types.StringValue(extractStringField(data, "bind"))
	state.BindOptions = types.StringValue(extractStringField(data, "bindOptions"))
	state.Mode = types.StringValue(extractStringField(data, "mode"))
	state.DefaultBackend = types.StringValue(extractSelectedUUIDs(data, "defaultBackend"))
	state.SSLEnabled = types.StringValue(extractStringField(data, "ssl_enabled"))
	state.SSLCertificates = types.StringValue(extractSelectedUUIDs(data, "ssl_certificates"))
	state.SSLDefaultCertificate = types.StringValue(extractSelectedUUIDs(data, "ssl_default_certificate"))
	state.SSLHSTSEnabled = types.StringValue(extractStringField(data, "ssl_hstsEnabled"))
	state.SSLHSTSMaxAge = types.StringValue(extractStringField(data, "ssl_hstsMaxAge"))
	state.SSLMinVersion = types.StringValue(extractStringField(data, "ssl_minVersion"))
	state.HTTP2Enabled = types.StringValue(extractStringField(data, "http2Enabled"))
	state.HTTP2EnabledNonTLS = types.StringValue(extractStringField(data, "http2Enabled_nontls"))
	state.AdvertisedProtocols = types.StringValue(extractStringField(data, "advertised_protocols"))
	state.ForwardFor = types.StringValue(extractStringField(data, "forwardFor"))
	state.ConnectionBehaviour = types.StringValue(extractStringField(data, "connectionBehaviour"))
	state.LinkedActions = types.StringValue(extractSelectedUUIDs(data, "linkedActions"))
	state.Enabled = types.StringValue(extractStringField(data, "enabled"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyFrontendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HAProxyFrontendModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyFrontendModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setFrontend/%s", state.ID.ValueString()), r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy frontend", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyFrontendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyFrontendModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/haproxy/settings/delFrontend/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy frontend", err.Error())
		return
	}
}

func (r *HAProxyFrontendResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
