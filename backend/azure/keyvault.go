package azure

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/mitchellh/mapstructure"
	"github.com/rapdev-io/datadog-secret-backend/secret"
	log "github.com/sirupsen/logrus"
)

type AzureKeyVaultBackendConfig struct {
	AzureSession AzureSessionBackendConfig   `mapstructure:"azure_session"`
	BackendType  string                      `mapstructure:"backend_type"`
	KeyVaultURL  string                      `mapstructure:"keyvaulturl"`
	SecretId     string                      `mapstructure:"secret_id"`
}

type AzureKeyVaultBackend struct {
	BackendId string
	Config    AzureKeyVaultBackendConfig
	Secret    map[string]string
}

func NewAzureKeyVaultBackend(backendId string, bc map[string]interface{}) (*AzureKeyVaultBackend, error) {
	backendConfig := AzureKeyVaultBackendConfig{}
	err := mapstructure.Decode(bc, &backendConfig)
	if err != nil {
		log.WithError(err).Error("failed to map backend configuration")
		return nil, err
	}

	cfg, err := NewAzureConfigFromBackendConfig(backendId, backendConfig.AzureSession)
	if err != nil {
		log.WithFields(log.Fields{
			"backend_id": backendId,
		}).WithError(err).Error("failed to initialize Azure session")
		return nil, err
	}
	client := keyvault.New()
	client.Authorizer = *cfg

	out, err := client.GetSecret(context.Background(), backendConfig.KeyVaultURL, backendConfig.SecretId, "latest")
	if err != nil {
		log.WithFields(log.Fields{
			"backend_id": backendId,
			"backend_type": backendConfig.BackendType,
			"secret_id": backendConfig.SecretId,
			"keyvaulturl": backendConfig.KeyVaultURL,
		}).WithError(err).Error("failed to retrieve secret value")
		return nil, err
	}

	secretValue := make(map[string]string, 0)
	if err := json.Unmarshal([]byte(*out.Value), &secretValue); err != nil {
		log.WithFields(log.Fields{
			"backend_id": backendId,
			"backend_type": backendConfig.BackendType,
			"secret_id": backendConfig.SecretId,
			"keyvaulturl": backendConfig.KeyVaultURL,
		}).WithError(err).Error("failed to retrieve secret value")
		return nil, err
	}

	backend := &AzureKeyVaultBackend{
		BackendId: backendId,
		Config:    backendConfig,
		Secret:    secretValue,
	}

	return backend, nil
}

func (b *AzureKeyVaultBackend) GetSecretOutput(secretKey string) secret.SecretOutput {
	if val, ok := b.Secret[secretKey]; ok {
		return secret.SecretOutput{Value: &val, Error: nil}
	}
	es := errors.New("backend does not provide secret key").Error()
	
	log.WithFields(log.Fields{
		"backend_id":   b.BackendId,
		"backend_type": b.Config.BackendType,
		"secret_id":    b.Config.SecretId,
		"keyvaulturl":  b.Config.KeyVaultURL,
		"secret_key":   secretKey,
	}).Error("backend does not provide secret key")
	return secret.SecretOutput{Value: nil, Error: &es}
}