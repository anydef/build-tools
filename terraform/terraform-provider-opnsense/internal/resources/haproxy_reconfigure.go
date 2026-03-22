package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &HAProxyReconfigureResource{}

type HAProxyReconfigureResource struct {
	client *OPNsenseClient
}

type HAProxyReconfigureModel struct {
	ID          types.String `tfsdk:"id"`
	LastApplied types.String `tfsdk:"last_applied"`
}

func NewHAProxyReconfigureResource() resource.Resource {
	return &HAProxyReconfigureResource{}
}

func (r *HAProxyReconfigureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_reconfigure"
}

func (r *HAProxyReconfigureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers an HAProxy reconfigure/restart on OPNsense. Use depends_on to ensure this runs after all HAProxy resource changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Resource ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_applied": schema.StringAttribute{
				Description: "Timestamp of last reconfigure. Changes on every apply to ensure it always runs.",
				Computed:    true,
			},
		},
	}
}

func (r *HAProxyReconfigureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HAProxyReconfigureResource) reconfigure(ctx context.Context) error {
	tflog.Info(ctx, "Triggering HAProxy reconfigure")
	return r.client.Post(ctx, "/api/haproxy/service/reconfigure")
}

func (r *HAProxyReconfigureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if err := r.reconfigure(ctx); err != nil {
		resp.Diagnostics.AddError("Error reconfiguring HAProxy", err.Error())
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state := HAProxyReconfigureModel{
		ID:          types.StringValue("haproxy-reconfigure"),
		LastApplied: types.StringValue(now),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyReconfigureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HAProxyReconfigureModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	// Nothing to read from remote - state is purely local
}

func (r *HAProxyReconfigureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if err := r.reconfigure(ctx); err != nil {
		resp.Diagnostics.AddError("Error reconfiguring HAProxy", err.Error())
		return
	}

	var state HAProxyReconfigureModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.LastApplied = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyReconfigureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to clean up - this is a trigger-only resource
}
