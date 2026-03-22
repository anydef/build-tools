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
)

var _ resource.Resource = &HAProxyActionResource{}

type HAProxyActionResource struct {
	client *OPNsenseClient
}

type HAProxyActionModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type       types.String `tfsdk:"type"`
	TestType   types.String `tfsdk:"test_type"`
	LinkedACLs types.String `tfsdk:"linked_acls"`
	Operator   types.String `tfsdk:"operator"`
	UseBackend types.String `tfsdk:"use_backend"`
}

func NewHAProxyActionResource() resource.Resource {
	return &HAProxyActionResource{}
}

func (r *HAProxyActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_action"
}

func (r *HAProxyActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy action (rule) in OPNsense.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the action.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Action name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Action description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"type": schema.StringAttribute{
				Description: "Action type (e.g. use_backend, map_use_backend).",
				Required:    true,
			},
			"test_type": schema.StringAttribute{
				Description: "Test type (if or unless).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("if"),
			},
			"linked_acls": schema.StringAttribute{
				Description: "Comma-separated list of ACL UUIDs.",
				Required:    true,
			},
			"operator": schema.StringAttribute{
				Description: "ACL operator (and or or).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("and"),
			},
			"use_backend": schema.StringAttribute{
				Description: "Backend UUID (for type=use_backend).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
		},
	}
}

func (r *HAProxyActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HAProxyActionResource) buildPayload(plan *HAProxyActionModel) map[string]interface{} {
	return map[string]interface{}{
		"action": map[string]interface{}{
			"name":        plan.Name.ValueString(),
			"description": plan.Description.ValueString(),
			"type":        plan.Type.ValueString(),
			"testType":    plan.TestType.ValueString(),
			"linkedAcls":  plan.LinkedACLs.ValueString(),
			"operator":    plan.Operator.ValueString(),
			"use_backend": plan.UseBackend.ValueString(),
		},
	}
}

func (r *HAProxyActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyActionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.client.Create(ctx, "/api/haproxy/settings/addAction", r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy action", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyActionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getAction/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy action", err.Error())
		return
	}

	// result parsed below
	result, err := ParseResponse(body)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing HAProxy action response", err.Error())
		return
	}

	data, ok := result["action"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing HAProxy action response", "missing 'action' key")
		return
	}

	state.Name = types.StringValue(extractStringField(data, "name"))
	state.Description = types.StringValue(extractStringField(data, "description"))
	state.Type = types.StringValue(extractExpressionValue(data, "type"))
	state.TestType = types.StringValue(extractExpressionValue(data, "testType"))
	state.LinkedACLs = types.StringValue(extractSelectedUUIDs(data, "linkedAcls"))
	state.Operator = types.StringValue(extractExpressionValue(data, "operator"))
	state.UseBackend = types.StringValue(extractSelectedUUIDs(data, "use_backend"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HAProxyActionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyActionModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setAction/%s", state.ID.ValueString()), r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy action", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyActionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/haproxy/settings/delAction/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy action", err.Error())
		return
	}
}
