# Setting up

First, create a Google Cloud project


## Initialize the infrastructure
Set your project in `infra/main.tfvars`

```
cd infra
terraform init
terraform apply -var-file=main.tfvars
```

## Import database schema
You need the SQL Connection Name and password of the db.  
```
cd infra # Make sure you're install in this dir
```

SQL password: 
```
terraform output -json sql-password | jq -r .result
```

Set it into an env var: 
```
export PGPASSWORD=$(terraform output -json sql-password | jq -r .result)
```

Connection name: 
```
terraform output -json sql-connection-name
```

Put the connection name in a shell env
```
CONNECTION_NAME=$(terraform output -json sql-connection-name | jq -r)
```

Start the cloud sql proxy on localhost: 
```
cloud-sql-proxy --address 0.0.0.0 --port 1234 $CONNECTION_NAME &
```

Start `psql`:
```
psql -h localhost -p 1234 -U app -d analytics
```

(`brew install libpq` if you don't have psql)

Load the schema in `sql/schema.sql` using \i:
``` 
analytics=> \i ../sql/schema.sql
CREATE TABLE
CREATE INDEX
CREATE INDEX
CREATE TABLE
CREATE TABLE
CREATE TABLE
INSERT 0 1
```

## Build and deploy services
Set the project in gloud local config: 
```
gcloud config set project [PROJECT]
```

Make sure Cloud Build [can deploy Cloud Run services](https://cloud.google.com/build/docs/securing-builds/configure-access-for-cloud-build-service-account). Assign **Cloud Run Admin** and **Service Account User**.

**Dashboard**
```
cd services/dashboard
./deploy.sh
```

**Loader**
```
cd services/loader
./deploy.sh
```

## Configure and execute a test
1. Open psql to configure a URL.
```
UPDATE loader_config SET href='https://cloud.google.com/run/', method='GET', body='';
```

2. Deploy a new revision of the loader service and set minimum instances to 15.

3. Open the dashboard service and send traffic. 

4. Stop the test by hitting "No traffic". There's no timeout to this test, it'll continue running if you don't stop it!

5. Double click on the grey area to reset instance count and reboot all loader instances. (I know, worst UI decision ever)

## Power down

1. Stop any running test by hitting "No traffic"

2. Deploy a new revision of the loader service with minimum instances set to ZERO. 

3. Wait for the gradual deployment to finish!

3. When the new revision is 100% live, kill any remaining loader instances by double clicking on the grey area. 

4. (Optional) Verify the loader_instances table is empty. `SELECT * from loader_instances`

5. Pause the SQL instance. 

6. Set ingress to the dashboard service to INTERNAL, just in case anyone remembered the URL. 






