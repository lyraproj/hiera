package provider

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// AzureKeyVaultLookupKey looks up a single value from an Azure Key Vault
func AzureKeyVaultLookupKey(c hieraapi.ProviderContext, key string, options map[string]px.Value) px.Value {
	vaultName, ok := options[`vault_name`]
	if !ok {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `vault_name`}))
	}
	authorizer, err := kvauth.NewAuthorizerFromCLI()
	if err != nil {
		panic(px.Error(hieraapi.MissingRequiredOption, issue.H{`option`: `auth`}))
	}
	// if os.Getenv("AZURE_TENANT_ID") == "" || os.Getenv("AZURE_CLIENT_ID") == "" || os.Getenv("AZURE_CLIENT_SECRET") == "" {

	// }

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	secretResp, err := basicClient.GetSecret(context.Background(), "https://"+vaultName.String()+".vault.azure.net", key, "")
	if err != nil {
		return nil
	}
	return types.WrapString(*secretResp.Value)
}
