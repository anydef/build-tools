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

var _ resource.Resource = &HAProxyACLResource{}

type HAProxyACLResource struct {
	client *OPNsenseClient
}

type HAProxyACLModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Expression    types.String `tfsdk:"expression"`
	Value         types.String `tfsdk:"value"`
	Negate        types.String `tfsdk:"negate"`
	CaseSensitive types.String `tfsdk:"case_sensitive"`
}

func NewHAProxyACLResource() resource.Resource {
	return &HAProxyACLResource{}
}

func (r *HAProxyACLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_acl"
}

func (r *HAProxyACLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy ACL (condition) in OPNsense.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the ACL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "ACL name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "ACL description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"expression": schema.StringAttribute{
				Description: "ACL expression type (e.g. hdr, hdr_end, hdr_beg, path_beg, src).",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "The match value. Stored in the field matching the expression (e.g. expression=hdr puts value in the 'hdr' field).",
				Required:    true,
			},
			"negate": schema.StringAttribute{
				Description: "Negate the ACL match (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"case_sensitive": schema.StringAttribute{
				Description: "Case sensitive matching (0 or 1).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
		},
	}
}

func (r *HAProxyACLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildPayload builds the ACL API payload. The value is placed in the field
// that matches the expression type (e.g. expression="hdr" -> "hdr": value).
func (r *HAProxyACLResource) buildPayload(plan *HAProxyACLModel) map[string]interface{} {
	acl := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"description":   plan.Description.ValueString(),
		"expression":    plan.Expression.ValueString(),
		"negate":        plan.Negate.ValueString(),
		"caseSensitive": plan.CaseSensitive.ValueString(),
	}
	// Place the value in the field matching the expression
	expr := plan.Expression.ValueString()
	acl[expr] = plan.Value.ValueString()

	return map[string]interface{}{
		"acl": acl,
	}
}

func (r *HAProxyACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyACLModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid, err := r.client.Create(ctx, "/api/haproxy/settings/addAcl", r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy ACL", err.Error())
		return
	}

	plan.ID = types.StringValue(uuid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyACLModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getAcl/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy ACL", err.Error())
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

	data, ok := result["acl"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Error parsing HAProxy ACL response", "missing 'acl' key")
		return
	}

	state.Name = types.StringValue(extractStringField(data, "name"))
	state.Description = types.StringValue(extractStringField(data, "description"))

	// The expression field might be returned as a map with selected values
	expr := extractExpressionValue(data, "expression")
	state.Expression = types.StringValue(expr)

	// Read the value from the field matching the expression
	state.Value = types.StringValue(extractStringField(data, expr))

	state.Negate = types.StringValue(extractStringField(data, "negate"))
	state.CaseSensitive = types.StringValue(extractStringField(data, "caseSensitive"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HAProxyACLModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyACLModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setAcl/%s", state.ID.ValueString()), r.buildPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy ACL", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyACLModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/api/haproxy/settings/delAcl/%s", state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy ACL", err.Error())
		return
	}
}

// extractExpressionValue handles the expression field which may be a string
// or a map of {value: label, selected: 0|1} entries from the API.
func extractExpressionValue(data map[string]interface{}, key string) string {
	val, ok := data[key]
	if !ok {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case map[string]interface{}:
		// Find the selected entry
		for k, entry := range v {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				sel, _ := entryMap["selected"]
				switch s := sel.(type) {
				case float64:
					if s == 1 {
						return k
					}
				case string:
					if s == "1" {
						return k
					}
				}
			}
		}
	}
	return ""
}
