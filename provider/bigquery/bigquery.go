package bigquery

import (
	"context"
	"fmt"

	bq "cloud.google.com/go/bigquery"
	bqApi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type bigQueryClient struct {
	projectID string
	client    *bq.Client
	apiClient *bqApi.Service
}

func newBigQueryClient(projectID string, credentialsJSON []byte) (*bigQueryClient, error) {
	ctx := context.Background()
	client, err := bq.NewClient(ctx, projectID, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, err
	}

	apiClient, err := bqApi.NewService(ctx, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, err
	}

	return &bigQueryClient{
		projectID: projectID,
		client:    client,
		apiClient: apiClient,
	}, nil
}

func NewDefaultBigQueryClient(projectID string) (*bigQueryClient, error) {
	ctx := context.Background()
	client, err := bq.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	apiClient, err := bqApi.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return &bigQueryClient{
		projectID: projectID,
		client:    client,
		apiClient: apiClient,
	}, nil
}

// GetDatasets returns all datasets within a project
func (c *bigQueryClient) GetDatasets(ctx context.Context) ([]*Dataset, error) {
	var results []*Dataset
	it := c.client.Datasets(ctx)
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		results = append(results, &Dataset{
			ProjectID: dataset.ProjectID,
			DatasetID: dataset.DatasetID,
		})
	}

	return results, nil
}

// GetTables returns all tables within a dataset
func (c *bigQueryClient) GetTables(ctx context.Context, datasetID string) ([]*Table, error) {
	var results []*Table
	it := c.client.Dataset(datasetID).Tables(ctx)
	for {
		table, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		results = append(results, &Table{
			ProjectID: table.ProjectID,
			DatasetID: table.DatasetID,
			TableID:   table.TableID,
		})
	}

	return results, nil
}

func (c *bigQueryClient) ResolveDatasetRole(role string) (bq.AccessRole, error) {
	switch role {
	case DatasetRoleReader:
		return bq.ReaderRole, nil
	case DatasetRoleWriter:
		return bq.WriterRole, nil
	case DatasetRoleOwner:
		return bq.OwnerRole, nil
	default:
		return "", ErrInvalidRole
	}
}

func (c *bigQueryClient) GrantDatasetAccess(ctx context.Context, d *Dataset, user, role string) error {
	dataset := c.client.Dataset(d.DatasetID)
	metadata, err := dataset.Metadata(ctx)
	if err != nil {
		return err
	}

	bqRole, err := c.ResolveDatasetRole(role)
	if err != nil {
		return err
	}
	for _, a := range metadata.Access {
		if a.Entity == user && a.Role == bqRole {
			return ErrPermissionAlreadyExists
		}
	}
	update := bq.DatasetMetadataToUpdate{
		Access: append(metadata.Access, &bq.AccessEntry{
			Role:       bqRole,
			EntityType: bq.UserEmailEntity,
			Entity:     user,
		}),
	}

	_, err = dataset.Update(ctx, update, metadata.ETag)
	return err
}

func (c *bigQueryClient) RevokeDatasetAccess(ctx context.Context, d *Dataset, user, role string) error {
	dataset := c.client.Dataset(d.DatasetID)
	metadata, err := dataset.Metadata(ctx)
	if err != nil {
		return err
	}

	bqRole, err := c.ResolveDatasetRole(role)
	if err != nil {
		return err
	}
	var removeIndex int
	for i, a := range metadata.Access {
		if a.Entity == user && a.Role == bqRole {
			removeIndex = i
			break
		}
	}
	if removeIndex == 0 {
		return ErrPermissionNotFound
	}

	update := bq.DatasetMetadataToUpdate{
		Access: append(metadata.Access[:removeIndex], metadata.Access[removeIndex+1:]...),
	}

	_, err = dataset.Update(ctx, update, metadata.ETag)
	return err
}

func (c *bigQueryClient) GrantTableAccess(ctx context.Context, t *Table, user, role string) error {
	resourceName := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", c.projectID, t.DatasetID, t.TableID)
	member := fmt.Sprintf("user:%s", user)

	tableService := c.apiClient.Tables
	getIamPolicyRequest := &bqApi.GetIamPolicyRequest{
		Options: &bqApi.GetPolicyOptions{
			RequestedPolicyVersion: 1,
		},
	}
	policy, err := tableService.GetIamPolicy(resourceName, getIamPolicyRequest).Do()
	if err != nil {
		return err
	}
	roleExists := false
	for _, b := range policy.Bindings {
		if b.Role == role {
			roleExists = true
			if containsString(b.Members, member) {
				return ErrPermissionAlreadyExists
			}
			b.Members = append(b.Members, member)
		}
	}
	if !roleExists {
		policy.Bindings = append(policy.Bindings, &bqApi.Binding{
			Role:    role,
			Members: []string{member},
		})
	}

	setIamPolicyRequest := &bqApi.SetIamPolicyRequest{
		Policy: policy,
	}
	_, err = tableService.SetIamPolicy(resourceName, setIamPolicyRequest).Do()
	return err
}

func (c *bigQueryClient) RevokeTableAccess(ctx context.Context, t *Table, user, role string) error {
	resourceName := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", c.projectID, t.DatasetID, t.TableID)
	member := fmt.Sprintf("user:%s", user)

	tableService := c.apiClient.Tables
	getIamPolicyRequest := &bqApi.GetIamPolicyRequest{
		Options: &bqApi.GetPolicyOptions{
			RequestedPolicyVersion: 1,
		},
	}
	policy, err := tableService.GetIamPolicy(resourceName, getIamPolicyRequest).Do()
	if err != nil {
		return err
	}
	var accessRemoved bool
	for _, b := range policy.Bindings {
		if b.Role == role {
			var removeIndex int
			for i, m := range b.Members {
				if m == member {
					removeIndex = i
				}
			}
			if removeIndex == 0 {
				return ErrPermissionNotFound
			}
			b.Members = append(b.Members[:removeIndex], b.Members[removeIndex+1:]...)
			accessRemoved = true
		}
	}
	if accessRemoved {
		return ErrPermissionNotFound
	}

	setIamPolicyRequest := &bqApi.SetIamPolicyRequest{
		Policy: policy,
	}
	_, err = tableService.SetIamPolicy(resourceName, setIamPolicyRequest).Do()
	return err
}

func containsString(arr []string, v string) bool {
	for _, item := range arr {
		if item == v {
			return true
		}
	}
	return false
}
