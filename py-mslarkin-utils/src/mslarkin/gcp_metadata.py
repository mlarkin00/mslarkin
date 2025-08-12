import os, re, logging
import psutil
from psutil._common import bytes2human
import json
import requests

if(os.environ.get('GAE_ENV') == 'standard'):
    app_env = 'gae_standard'
elif(os.environ.get('K_SERVICE', "NO_SERVICE") != "NO_SERVICE"):
    app_env = "cloud_run"
else:
    app_env = "UNKNOWN"
    logging.warning("Unknown App Environment")

def memory_convert(memory_bytes):
    return memory_bytes / 1000000

def get_metadata(path_id):
    # "http://metadata" OR "http://metadata.google.internal"
    query_path = "http://metadata.google.internal%s" % (path_id)
    return requests.get(query_path, headers={'Metadata-Flavor': 'Google'}).text

def get_admin_api(path_id):
    auth_token =  dict(json.loads(get_metadata('/computeMetadata/v1/instance/service-accounts/default/token')))
    auth_header = 'Bearer %s' % auth_token['access_token']

    return requests.get(path_id, headers={"Authorization": auth_header}).text

def get_service(service_query):
    service_data = dict(json.loads(get_admin_api(service_query)))
    return service_data

def get_version(version_query):
    version_data = dict(json.loads(get_admin_api(version_query)))
    return version_data

def get_instance(instance_query):
    instance_data = dict(json.loads(get_admin_api(instance_query)))
    return instance_data

def get_memory(instance_query):
    if(app_env == 'gae_standard'):
        instance_data = get_instance(instance_query)
        memory_usage = memory_convert(int(instance_data['memoryUsage']))
    else:
        memory_usage = 0
    return memory_usage

def log_memory(id, src, mem_val):
    output = {'id':id, 'source':src, 'memory_used_by':round(mem_val)}
    logging.info("MEMORY: %s" % output)

# def no_db_access(u_path):
#     # Common Metadata
#     try:
#         gcp_project_id = get_metadata('/computeMetadata/v1/project/project-id')
#         instance_id = get_metadata('/computeMetadata/v1/instance/id')
#     except:
#         # Gen1 doesn't have the metadata server option
#         gcp_project_id = re.sub('^[a-z].*~', '', os.environ.get('GAE_APPLICATION'))
#         instance_id = os.environ.get('GAE_INSTANCE')

#     service_name = os.environ.get('K_SERVICE', os.environ.get('GAE_SERVICE'))
#     version_id = os.environ.get('K_REVISION', os.environ.get('GAE_VERSION'))

#     # Gen1 returns the major_version, while Gen2 returns minor_version
#     version_check = re.match('.*:(.*)', version_id)
#     if version_check: # If the version_id is major_version (Gen1)
#         version_id = version_check.group(1)

#     request_url = str(request.path)

#     # Get metadata queries set up
#     # if(app_env == 'gae_standard'):
#     #     base_query = 'https://appengine.googleapis.com/v1/apps/%s' % gcp_project_id
#     #     service_query = '%s/services/%s' % (base_query, service_name)
#     #     version_query = '%s/versions/%s' % (service_query, version_id)
#     #     instance_query = '%s/instances/%s' % (version_query, instance_id)
#     #     # logging.warning("Service Query: %s" % service_query)
#     #     # logging.warning("Version Query: %s" % version_query)
#     #     # logging.warning("Instance Query: %s" % instances_query)
#     # elif(app_env == 'cloud_run'):
#     #     gcp_region = get_metadata('/computeMetadata/v1/instance/region')
#     #     base_query = re.sub('/regions/', '/locations/', 'https://run.googleapis.com/v2/%s' % gcp_region)
#     #     service_query = '%s/services/%s' % (base_query, service_name)
#     #     version_query = '%s/revisions/%s' % (service_query, version_id)
#     #     # instance_query = '%s/instances/%s' % (version_query, instance_id)
#     #     logging.warning("Service Query: %s" % service_query)
#     #     logging.warning("Version Query: %s" % version_query)
#         # logging.warning("Instance Query: %s" % instance_query)

#     # logging.info("SERVICE NAME: %s" % service_name)
#     # logging.info("VERSION ID: %s" % version_id)
#     # logging.info("INSTANCE ID: %s" % instance_id)

#     # logging.warning("Service: %s" % get_service(service_query))
#     # logging.warning("Version: %s" % get_version(version_query))


#     logging.info("Total (psutil): %sMB" % round(memory_convert(psutil.virtual_memory().total)))
#     # if(app_env == 'gae_standard'): log_memory(instance_id, 'admin_api', get_memory(instance_query))
#     log_memory(instance_id, 'psutil', psutil.virtual_memory().used)
