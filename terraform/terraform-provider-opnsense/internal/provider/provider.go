package provider

import (
	"context"

	"github.com/anydef/build-tools/terraform/terraform-provider-opnsense/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &OPNsenseProvider{}

type OPNsenseProvider struct {
	version string
}

type OPNsenseProviderModel struct {
	URL       types.String `tfsdk:"url"`
	APIKey    types.String `tfsdk:"api_key"`
	APISecret types.String `tfsdk:"api_secret"`
	Insecure  types.Bool   `tfsdk:"insecure"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OPNsenseProvider{
			version: version,
		}
	}
}

func (p *OPNsenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "opnsense"
	resp.Version = p.version
}

func (p *OPNsenseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing OPNsense HAProxy and Unbound resources via the OPNsense API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "OPNsense base URL (e.g. https://192.168.1.1). Can also be set via OPNSENSE_URL env var.",
				Required:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "OPNsense API key. Can also be set via OPNSENSE_API_KEY env var.",
				Required:    true,
				Sensitive:   true,
			},
			"api_secret": schema.StringAttribute{
				Description: "OPNsense API secret. Can also be set via OPNSENSE_API_SECRET env var.",
				Required:    true,
				Sensitive:   true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Defaults to false.",
				Optional:    true,
			},
		},
	}
}

func (p *OPNsenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OPNsenseProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	insecure := false
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	}

	client := resources.NewOPNsenseClient(
		config.URL.ValueString(),
		config.APIKey.ValueString(),
		config.APISecret.ValueString(),
		insecure,
	)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *OPNsenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewHAProxyServerResource,
		resources.NewHAProxyBackendResource,
		resources.NewHAProxyACLResource,
		resources.NewHAProxyActionResource,
		resources.NewHAProxyFrontendActionResource,
		resources.NewHAProxyReconfigureResource,
		resources.NewUnboundHostOverrideResource,
	}
}

func (p *OPNsenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		resources.NewHaproxyFrontendDataSource,
	}
}
