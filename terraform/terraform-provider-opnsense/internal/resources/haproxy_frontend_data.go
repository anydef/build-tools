package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HaproxyFrontendDataSource{}

type HaproxyFrontendDataSource struct {
	client *OPNsenseClient
}

type HaproxyFrontendDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewHaproxyFrontendDataSource() datasource.DataSource {
	return &HaproxyFrontendDataSource{}
}

func (d *HaproxyFrontendDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_frontend"
}

func (d *HaproxyFrontendDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an HAProxy frontend by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the frontend.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the frontend to look up.",
			},
		},
	}
}

func (d *HaproxyFrontendDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*OPNsenseClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", "Expected *OPNsenseClient")
		return
	}
	d.client = client
}

func (d *HaproxyFrontendDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HaproxyFrontendDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all settings and find frontend by name
	body, err := d.client.Read(ctx, "/api/haproxy/settings/get")
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to get HAProxy settings: %s", err))
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(body, &settings); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Failed to parse settings: %s", err))
		return
	}

	haproxy, ok := settings["haproxy"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Parse Error", "Missing 'haproxy' key in settings")
		return
	}
	frontends, ok := haproxy["frontends"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Parse Error", "Missing 'frontends' key")
		return
	}
	frontendMap, ok := frontends["frontend"].(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError("Parse Error", "Missing 'frontend' key")
		return
	}

	targetName := data.Name.ValueString()
	for uuid, v := range frontendMap {
		entry, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if name == targetName {
			data.ID = types.StringValue(uuid)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Frontend '%s' not found", targetName))
}
