package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HAProxyFrontendActionResource{}

type HAProxyFrontendActionResource struct {
	client *OPNsenseClient
}

type HAProxyFrontendActionModel struct {
	ID         types.String `tfsdk:"id"`
	FrontendID types.String `tfsdk:"frontend_id"`
	ActionID   types.String `tfsdk:"action_id"`
	Prepend    types.Bool   `tfsdk:"prepend"`
}

func NewHAProxyFrontendActionResource() resource.Resource {
	return &HAProxyFrontendActionResource{}
}

func (r *HAProxyFrontendActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_frontend_action"
}

func (r *HAProxyFrontendActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links an HAProxy action to a frontend. This is a join resource that manages the linkedActions field on the frontend.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite ID (frontend_id/action_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"frontend_id": schema.StringAttribute{
				Description: "UUID of the frontend to link the action to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action_id": schema.StringAttribute{
				Description: "Comma-separated UUIDs of actions to link to the frontend.",
				Required:    true,
			},
			"prepend": schema.BoolAttribute{
				Description: "Prepend the action before existing actions (true) or append after (false). Per-service rules should be prepended before catch-all mapfile rules.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *HAProxyFrontendActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// getLinkedActions reads the current frontend and returns the list of currently
// linked (selected) action UUIDs.
func (r *HAProxyFrontendActionResource) getLinkedActions(ctx context.Context, frontendID string) ([]string, error) {
	body, err := r.client.Read(ctx, fmt.Sprintf("/api/haproxy/settings/getFrontend/%s", frontendID))
	if err != nil {
		return nil, fmt.Errorf("failed to read frontend: %w", err)
	}

	// result parsed below
	result, err := ParseResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontend response: %w", err)
	}

	frontendData, ok := result["frontend"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing 'frontend' key in response")
	}

	linkedRaw, ok := frontendData["linkedActions"]
	if !ok {
		return []string{}, nil
	}

	switch v := linkedRaw.(type) {
	case string:
		if v == "" {
			return []string{}, nil
		}
		return strings.Split(v, ","), nil
	case map[string]interface{}:
		var selected []string
		for uuid, entry := range v {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				sel, _ := entryMap["selected"]
				switch s := sel.(type) {
				case float64:
					if s == 1 {
						selected = append(selected, uuid)
					}
				case string:
					if s == "1" {
						selected = append(selected, uuid)
					}
				}
			}
		}
		return selected, nil
	default:
		return []string{}, nil
	}
}

// setLinkedActions updates the frontend's linkedActions to the given list.
func (r *HAProxyFrontendActionResource) setLinkedActions(ctx context.Context, frontendID string, actions []string) error {
	payload := map[string]interface{}{
		"frontend": map[string]interface{}{
			"linkedActions": strings.Join(actions, ","),
		},
	}
	return r.client.Update(ctx, fmt.Sprintf("/api/haproxy/settings/setFrontend/%s", frontendID), payload)
}

func (r *HAProxyFrontendActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HAProxyFrontendActionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	frontendID := plan.FrontendID.ValueString()
	actionIDs := strings.Split(plan.ActionID.ValueString(), ",")

	current, err := r.getLinkedActions(ctx, frontendID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading frontend", err.Error())
		return
	}

	// Build set of current actions for quick lookup
	currentSet := make(map[string]bool)
	for _, id := range current {
		currentSet[id] = true
	}

	// Find actions not yet linked
	var toAdd []string
	for _, id := range actionIDs {
		id = strings.TrimSpace(id)
		if id != "" && !currentSet[id] {
			toAdd = append(toAdd, id)
		}
	}

	if len(toAdd) > 0 {
		var newActions []string
		if plan.Prepend.ValueBool() {
			newActions = append(toAdd, current...)
		} else {
			newActions = append(current, toAdd...)
		}

		if err := r.setLinkedActions(ctx, frontendID, newActions); err != nil {
			resp.Diagnostics.AddError("Error linking actions to frontend", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(frontendID + "/" + plan.ActionID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyFrontendActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyFrontendActionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	frontendID := state.FrontendID.ValueString()
	actionIDs := strings.Split(state.ActionID.ValueString(), ",")

	current, err := r.getLinkedActions(ctx, frontendID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading frontend", err.Error())
		return
	}

	currentSet := make(map[string]bool)
	for _, id := range current {
		currentSet[id] = true
	}

	// Check all action IDs are linked
	allLinked := true
	for _, id := range actionIDs {
		id = strings.TrimSpace(id)
		if id != "" && !currentSet[id] {
			allLinked = false
			break
		}
	}

	if !allLinked {
		// Some actions are no longer linked - remove from state so they get recreated
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyFrontendActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Frontend_id and action_id both have RequiresReplace, so Update should
	// only be called if prepend changes. In that case we don't need to do
	// anything since the action is already linked.
	var plan HAProxyFrontendActionModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HAProxyFrontendActionModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyFrontendActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HAProxyFrontendActionModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	frontendID := state.FrontendID.ValueString()
	actionIDs := strings.Split(state.ActionID.ValueString(), ",")

	current, err := r.getLinkedActions(ctx, frontendID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading frontend", err.Error())
		return
	}

	// Build set of actions to remove
	removeSet := make(map[string]bool)
	for _, id := range actionIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			removeSet[id] = true
		}
	}

	// Filter out removed actions
	var newActions []string
	for _, id := range current {
		if !removeSet[id] {
			newActions = append(newActions, id)
		}
	}

	if err := r.setLinkedActions(ctx, frontendID, newActions); err != nil {
		resp.Diagnostics.AddError("Error unlinking actions from frontend", err.Error())
		return
	}
}
