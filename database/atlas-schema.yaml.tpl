apiVersion: db.atlasgo.io/v1alpha1
kind: AtlasSchema
metadata:
  name: psdb-atlas
spec:
  urlFrom:
    secretKeyRef:
      key: url_ssl_disable
      name: postgres-secret-config
  schema:
    sql: |
{{ .SQL }}
  policy:
    lint:
      destructive:
        error: false
    diff:
      skip:
        drop_column: false
