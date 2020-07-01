package main

import (
	"context"
	"net/http"

	"cloud.google.com/go/bigquery"
	"github.com/zatte/tealiumtobqaudiencedump/internal/models"
	"google.golang.org/api/googleapi"
)

func getOrCreateDataSet(
	ctx context.Context,
	client *bigquery.Client,
	project, dataset string,
) (*bigquery.Dataset, error) {
	ds := client.DatasetInProject(project, dataset)

	// If there is metadata, the table exists
	_, err := ds.Metadata(ctx)
	if err == nil {
		return ds, nil
	}

	if e, ok := err.(*googleapi.Error); !ok || e.Code != http.StatusNotFound {
		return nil, err // 404 is expected, anything else is cray cray
	}

	return ds, ds.Create(ctx, &bigquery.DatasetMetadata{
		Name:     dataset,
		Location: bigQueryDefaultLocation,
	})
}

func getOrCreateTable(
	ctx context.Context,
	client *bigquery.Client,
	project, dataset, table string,
	meta *bigquery.TableMetadata,
) (*bigquery.Table, error) {
	ds, err := getOrCreateDataSet(ctx, client, project, dataset)
	if err != nil {
		return nil, err
	}

	t := ds.Table(table)
	if _, err = t.Metadata(ctx); err == nil {
		return t, nil
	}
	if e, ok := err.(*googleapi.Error); !ok || e.Code != http.StatusNotFound {
		return nil, err // 404 is expected, anything else is cray cray
	}

	t = ds.Table(table)
	return t, t.Create(ctx, meta)
}

func getTableInserter(ctx context.Context, projectID, datasetID, tableID string) (*bigquery.Inserter, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tableSchema, err := bigquery.InferSchema(models.AudienceTable{})
	if err != nil {
		return nil, err
	}

	t, err := getOrCreateTable(ctx, client, projectID, datasetID, tableID, &bigquery.TableMetadata{
		TimePartitioning: &bigquery.TimePartitioning{},
		Schema:           tableSchema,
	})
	if err != nil {
		return nil, err
	}

	return t.Inserter(), nil
}
