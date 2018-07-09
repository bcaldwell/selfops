# ynab-influx-importer

## Example configuration

##### ./config.yaml
``` yaml
currencies:
  - USD
  - CAD
budgets:
  - name: USD
  - name: CAD
```

##### ./secrets.json
``` json
{
  "ynab_access_token":
    "token",
  "influx_endpoint": "http://localhost:8086"
}

```