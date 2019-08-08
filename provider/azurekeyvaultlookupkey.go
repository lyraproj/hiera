package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// AzureKeyVaultLookupKey looks up a single value from an Azure Key Vault
func AzureKeyVaultLookupKey(hc hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	if key == `lookup_options` {
		hc.NotFound()
	}
	vaultName, ok := options[`vault_name`]
	if !ok {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `vault_name`}))
	}
	var authorizer autorest.Authorizer
	var err error
	if os.Getenv("AZURE_TENANT_ID") != "" && os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != "" {
		authorizer, err = kvauth.NewAuthorizerFromEnvironment()
		if err != nil {
			panic(err)
		}
	} else {
		authorizer, err = kvauth.NewAuthorizerFromCLI()
		if err != nil {
			panic(err)
		}
	}
	client := keyvault.New()
	client.Authorizer = authorizer
	resp, err := client.GetSecret(context.Background(), "https://"+vaultName.String()+".vault.azure.net", key, "")
	if err != nil {
		if ResponseWasStatusCode(resp.Response, http.StatusNotFound) {
			hc.NotFound()
		}
		panic(err)
	}
	return types.WrapString(*resp.Value)
}

func ResponseWasStatusCode(resp autorest.Response, statusCode int) bool {
	if r := resp.Response; r != nil {
		if r.StatusCode == statusCode {
			return true
		}
	}
	return false
}
