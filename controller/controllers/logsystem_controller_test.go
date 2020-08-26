package controllers

import (
	"github.com/kalmhq/kalm/controller/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPLGMonolithicLokiConfig(t *testing.T) {
	// build a mock reconciler task
	r := &LogSystemReconcilerTask{
		logSystem: &v1alpha1.LogSystem{
			Spec: v1alpha1.LogSystemSpec{
				PLGConfig: &v1alpha1.PLGConfig{
					Loki: &v1alpha1.LokiConfig{
						RetentionDays: 0,
					},
				},
			},
		},
	}

	expected := `auth_enabled: false
server:
  http_listen_port: 3100
ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 5m
  chunk_retain_period: 30s
  max_transfer_retries: 0
schema_config:
  configs:
    - from: 2018-04-15
      store: boltdb
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 168h
storage_config:
  boltdb:
    directory: /data/loki/index
  filesystem:
    directory: /data/loki/chunks
limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 0s
chunk_store_config:
  max_look_back_period: 0s
table_manager:
  retention_deletes_enabled: false
  retention_period: 0s
`
	res := r.GetPLGMonolithicLokiConfig()

	assert.Equal(t, expected, res)

	r.logSystem.Spec.PLGConfig.Loki.RetentionDays = 6
	expected = `auth_enabled: false
server:
  http_listen_port: 3100
ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 5m
  chunk_retain_period: 30s
  max_transfer_retries: 0
schema_config:
  configs:
    - from: 2018-04-15
      store: boltdb
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 168h
storage_config:
  boltdb:
    directory: /data/loki/index
  filesystem:
    directory: /data/loki/chunks
limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 144h
chunk_store_config:
  max_look_back_period: 144h
table_manager:
  retention_deletes_enabled: true
  retention_period: 144h
`

	res = r.GetPLGMonolithicLokiConfig()
	assert.Equal(t, expected, res)
}
