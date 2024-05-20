import os, re, json
import requests 

def get_resource_from_path(resource_path):
    result = re.search(r'.*/(.*)$',resource_path) 
    return result.group(1)

def get_project_id():
    gcp_project_id = requests.get(
        'http://metadata.google.internal/computeMetadata/v1/project/project-id', headers={'Metadata-Flavor': 'Google'}).text
    return gcp_project_id

def get_project_number():
    gcp_project_number = requests.get(
        'http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id', headers={'Metadata-Flavor': 'Google'}).text
    return gcp_project_number

def get_instance_id():
    instance_id = requests.get(
        'http://metadata.google.internal/computeMetadata/v1/instance/id', headers={'Metadata-Flavor': 'Google'}).text
    return instance_id

def get_instance_id_short():
    instance_id = get_instance_id()
    short_id = re.search(r'.*(.{10})$', instance_id)
    return short_id.group(1)

def get_gcp_region():
    r = requests.get(
        'http://metadata.google.internal/computeMetadata/v1/instance/region', headers={'Metadata-Flavor': 'Google'}).text
    region = get_resource_from_path(r)
    return region

def get_service_account():
    service_account = requests.get(
        'http://metadata.google.internal//computeMetadata/v1/instance/service-accounts/default/email', headers={'Metadata-Flavor': 'Google'}).text
    return service_account

def get_access_token():
    r = (requests.get(
    'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token', headers={'Metadata-Flavor': 'Google'}).json())
    access_token = r["access_token"]
    return access_token

def get_service_id():
    service_id = os.environ.get('K_SERVICE', os.environ.get('GAE_SERVICE'))
    return service_id

def get_revision_id():
    revision_id = os.environ.get('K_REVISION', os.environ.get('GAE_VERSION'))
    return revision_id

def call_admin_api(path, url='https://run.googleapis.com/v2',op='GET', payload=None, project_id=None, region=None, access_token=None):
    if project_id == None:
        project_id = get_project_id()
    if region == None:
        region = get_gcp_region()
    path_base = f"projects/{project_id}/locations/{region}"
   
    if access_token == None:
        access_token = get_access_token()
    headers = {"Authorization": f"Bearer {access_token}"}

    full_url = f'{url}/{path_base}/{path}'

    # print(f"Calling API: {full_url} | Operation: {op}")
    if op == "GET":
        r = requests.get(full_url, headers=headers)
    elif op == "PATCH":
        r = requests.patch(full_url, headers=headers, data=payload)
    elif op == "DELETE":
        r = requests.delete(full_url, headers=headers)
    elif op == "POST":
        r = requests.post(full_url, headers=headers, data=payload)

    try:
        return_value = r.json()
    except (requests.exceptions.JSONDecodeError):
        return_value = r.text

    return return_value
