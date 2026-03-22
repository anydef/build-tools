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

var _ resource.Resource = &HAProxyServerResource{}
var _ resource.ResourceWithImportState = &HAProxyServerResource{}

type HAProxyServerResource struct {
	client *OPNsenseClient
}

type HAProxyServerModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Address     types.String `tfsdk:"address"`
	Port        types.String `tfsdk:"port"`
	SSL         types.String `tfsdk:"ssl"`
	SSLVerify   types.String `tfsdk:"ssl_verify"`
	Enabled     types.String `tfsdk:"enabled"`
}

func NewHAProxyServerResource() resource.Resource {
	return &HAProxyServerResource{}
}

func (r *HAProxyServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_server"
}

func (r *HAProxyServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy real server in OPNsense.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the server.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Server name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Server description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"address": schema.StringAttribute{
				Description: "Server IP address or hostname.",
				Required:    true,
			},
			"port": schema.StringAttribute{
				Description: "Server port.",
				Required:    true,
			},
			"ssl": schema.StringAttribute{
				Description: "Enable SSL to backend server (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"ssl_verify": schema.StringAttribute{
				Description: "Enable SSL certificate verification (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"enabled": schema.StringAttribute{
				Description: "Enable server (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1"),
			},
		},
	}
}

func (r *HAProxyServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HAProxyServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyServerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]interface{}{
		"server": map[string]interface{}{
			"enabled":     plan.Enabled.ValueString(),
			"name":        plan.Name.ValueString(),
			"description": plan.Description.ValueString(),
			"address":     plan.Address.ValueString(),
			"port":        plan.Port.ValueString(),
			"ssl":         plan.SSL.ValueString(),
			"sslVerify":   plan.SSLVerify.ValueString(),
		},
	}

	uuid, err := r.client.Create(ctx, "/api/haproxy/settings/addServer", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy server", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyServerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getServer/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy server", err.Error())
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

	serverData, ok := result["server"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing HAProxy server response", "missing 'server' key")
		return
	}

	state.Name = types.StringValue(extractStringField(serverData, "name"))
	state.Description = types.StringValue(extractStringField(serverData, "description"))
	state.Address = types.StringValue(extractStringField(serverData, "address"))
	state.Port = types.StringValue(extractStringField(serverData, "port"))
	state.SSL = types.StringValue(extractStringField(serverData, "ssl"))
	state.SSLVerify = types.StringValue(extractStringField(serverData, "sslVerify"))
	state.Enabled = types.StringValue(extractStringField(serverData, "enabled"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HAProxyServerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyServerModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := map[string]interface{}{
		"server": map[string]interface{}{
			"enabled":     plan.Enabled.ValueString(),
			"name":        plan.Name.ValueString(),
			"description": plan.Description.ValueString(),
			"address":     plan.Address.ValueString(),
			"port":        plan.Port.ValueString(),
			"ssl":         plan.SSL.ValueString(),
			"sslVerify":   plan.SSLVerify.ValueString(),
		},
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setServer/%s", state.ID.ValueString()), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy server", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyServerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/haproxy/settings/delServer/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy server", err.Error())
		return
	}
}

func (r *HAProxyServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
