from google.cloud import run_v2
from google.cloud import monitoring_v3
from . import gcp_utils, tools
import statistics, time, json

def get_run_service(service=None, project_id=None, region=None):
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    service_id = f"projects/{project_id}/locations/{region}/services/{service}"
    run_client = run_v2.ServicesClient()
    run_service = run_client.get_service(name=service_id)
    return run_service

def get_run_revision(revision=None, service=None, project_id=None, region=None):
    if revision == None:
        revision = gcp_utils.get_revision_id()

    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    revision_id = f"projects/{project_id}/locations/{region}/services/{service}/revisions/{revision}"
    run_client = run_v2.RevisionsClient()
    run_revision = run_client.get_revision(name=revision_id)

    return run_revision

def get_latest_revision(service=None, project_id=None, region=None):
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_service = get_run_service(service=service, project_id=project_id, region=region)
    service_revision = gcp_utils.get_resource_from_path(run_service.latest_ready_revision)
    return service_revision

def get_service_url(service=None, project_id=None, region=None):
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_service = get_run_service(service=service, project_id=project_id, region=region)
    return run_service.uri

def get_last_update_ts(service=None, project_id=None, region=None):
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_service = get_run_service(service=service, project_id=project_id, region=region)
    return tools.get_pacific_timestamp(run_service.update_time)

def no_metric_results_error(monitoring_metric, resource_filter, interval_s, aggregation_s, group_by, interval, results):
    inputs = {'metric':monitoring_metric, 'filter':resource_filter, 'interval_s':interval_s, 'agg_s':aggregation_s, 'group_by':group_by}
    print(f'ERR: No timeseries data with {inputs}')
    
    start_ts = tools.get_pacific_timestamp(interval.start_time)
    end_ts = tools.get_pacific_timestamp(interval.end_time)
    start_time = start_ts.strftime('%H:%M:%S')
    end_time = end_ts.strftime('%H:%M:%S')
    print(f"ERR: Results: {results}; Start time: {start_time} - End time: {end_time}")

def get_metric_interval(interval_s):
    now = time.time()
    end_seconds = int(now)
    nanos = int((now - end_seconds) * 10**9)
    start_seconds = end_seconds - interval_s

    interval = monitoring_v3.TimeInterval(
        {
            "end_time": {"seconds": end_seconds, "nanos": nanos},
            "start_time": {"seconds": start_seconds, "nanos": nanos},
        }
    )

    return interval

def get_metric_avg(monitoring_metric, 
                   resource_filter=True, 
                   interval_s=240, 
                   aggregation_s=60, 
                   group_by=None, 
                   project_id=None):
    client = monitoring_v3.MetricServiceClient()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    project_name = f"projects/{project_id}"
    interval = get_metric_interval(interval_s=interval_s)
    if resource_filter == None:
        filter = f'metric.type = "{monitoring_metric}"'
    else:
        filter = f'metric.type = "{monitoring_metric}" AND {resource_filter}'

    pubsub_aggregation = monitoring_v3.Aggregation(
        {
            "alignment_period": {"seconds": aggregation_s},
            "per_series_aligner": monitoring_v3.Aggregation.Aligner.ALIGN_MEAN,
            "cross_series_reducer": monitoring_v3.Aggregation.Reducer.REDUCE_SUM,
            "group_by_fields": {group_by},
        }
    )

    metric_request = monitoring_v3.ListTimeSeriesRequest(
        name=project_name,
        filter=filter,
        interval=interval,
        view=monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
        aggregation=pubsub_aggregation,
    )

    results = client.list_time_series(request=metric_request)
    metric_data = []
    if len(results.time_series) == 0:
        no_metric_results_error(monitoring_metric=monitoring_metric,
                                resource_filter=resource_filter, 
                                interval_s=interval_s, 
                                aggregation_s=aggregation_s, 
                                group_by=group_by,
                                interval=interval,
                                results=results)
        return [0]

    for data_point in results.time_series[0].points:
        metric_value = data_point.value.double_value
        metric_data_point = metric_value
        metric_data.append(metric_data_point)
    return metric_data

def get_metric_rate(monitoring_metric, 
                    resource_filter=True, 
                    interval_s=240, 
                    aggregation_s=60, 
                    group_by=None, 
                    project_id=None):
    client = monitoring_v3.MetricServiceClient()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    project_name = f"projects/{project_id}"
    interval = get_metric_interval(interval_s=interval_s)
    if resource_filter == None:
        filter = f'metric.type = "{monitoring_metric}"'
    else:
        filter = f'metric.type = "{monitoring_metric}" AND {resource_filter}'

    pubsub_aggregation = monitoring_v3.Aggregation(
        {
            "alignment_period": {"seconds": aggregation_s},
            "per_series_aligner": monitoring_v3.Aggregation.Aligner.ALIGN_RATE,
            "cross_series_reducer": monitoring_v3.Aggregation.Reducer.REDUCE_SUM,
            "group_by_fields": {group_by},
        }
    )

    metric_request = monitoring_v3.ListTimeSeriesRequest(
        name=project_name,
        filter=filter,
        interval=interval,
        view=monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
        aggregation=pubsub_aggregation,
    )

    results = client.list_time_series(request=metric_request)
    metric_data = []
    if len(results.time_series) == 0:
        no_metric_results_error(monitoring_metric=monitoring_metric,
                                resource_filter=resource_filter, 
                                interval_s=interval_s, 
                                aggregation_s=aggregation_s, 
                                group_by=group_by,
                                interval=interval,
                                results=results)
        return [0]

    for data_point in results.time_series[0].points:
        metric_value = data_point.value.double_value
        metric_data_point = metric_value
        metric_data.append(metric_data_point)
    return metric_data

def get_metric_delta(monitoring_metric, 
                     resource_filter=True, 
                     interval_s=240, 
                     aggregation_s=60, 
                     group_by=None, 
                     project_id=None):
    client = monitoring_v3.MetricServiceClient()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    project_name = f"projects/{project_id}"
    interval = get_metric_interval(interval_s=interval_s)
    if resource_filter == None:
        filter = f'metric.type = "{monitoring_metric}"'
    else:
        filter = f'metric.type = "{monitoring_metric}" AND {resource_filter}'

    pubsub_aggregation = monitoring_v3.Aggregation(
        {
            "alignment_period": {"seconds": aggregation_s},
            "per_series_aligner": monitoring_v3.Aggregation.Aligner.ALIGN_DELTA,
            "cross_series_reducer": monitoring_v3.Aggregation.Reducer.REDUCE_PERCENTILE_50,
            "group_by_fields": {group_by},
        }
    )

    metric_request = monitoring_v3.ListTimeSeriesRequest(
        name=project_name,
        filter=filter,
        interval=interval,
        view=monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
        aggregation=pubsub_aggregation,
    )

    results = client.list_time_series(request=metric_request)
    metric_data = []

    if len(results.time_series) == 0:
        no_metric_results_error(monitoring_metric=monitoring_metric,
                                resource_filter=resource_filter, 
                                interval_s=interval_s, 
                                aggregation_s=aggregation_s, 
                                group_by=group_by,
                                interval=interval,
                                results=results)
        return [0]

    for data_point in results.time_series[0].points:
        metric_value = data_point.value.double_value
        metric_data_point = metric_value
        metric_data.append(metric_data_point)
    return metric_data

def get_instance_count(service=None, project_id=None, region=None):
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    monitoring_metric = "run.googleapis.com/container/instance_count"
    aggregation_s = 60
    interval_s = 240 # Instance count reporting delay up to 180s (errors with just 180)
    group_by = 'resource.labels.service_name'
    service_filter = f'resource.labels.service_name = "{service}" AND resource.labels.location = "{region}"'

    metric_data = get_metric_avg(monitoring_metric=monitoring_metric, 
                                resource_filter=service_filter, 
                                interval_s=interval_s, 
                                aggregation_s=aggregation_s, 
                                group_by=group_by,
                                project_id=project_id)
    return_val = round(statistics.fmean(metric_data))
    return return_val

def get_service_min_instances(service=None, project_id=None, region=None, access_token=None):
    # Get service-level min-instances setting
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()
    run_service = gcp_utils.call_admin_api(path=f'services/{service}', 
                                url='https://run.googleapis.com/v2',
                                op="GET", 
                                project_id=project_id,
                                region=region,
                                access_token=access_token
                                )
    
    try:
        min_instances = run_service['scaling']['minInstanceCount']
    except:
        # print("Error reading Service Min Instances")
        min_instances = 0

    return min_instances

def set_service_min_instances(instances, service=None, project_id=None, region=None, access_token=None):
    # Manually change Run instance count (via Service-level min-instances)
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_service = gcp_utils.call_admin_api(path=f'services/{service}', 
                                url='https://run.googleapis.com/v2',
                                op="GET",
                                project_id=project_id,
                                region=region,
                                access_token=access_token
                                )
    run_service['scaling']['minInstanceCount'] = instances
    op_response = gcp_utils.call_admin_api(path=f'services/{service}', 
                                url='https://run.googleapis.com/v2',
                                op="PATCH",
                                payload=json.dumps(run_service),
                                project_id=project_id,
                                region=region,
                                access_token=access_token
                                )
    
    return op_response

def get_revision_max_instances(revision=None, service=None, project_id=None, region=None):
    # Get revision-level max-instances setting
    if revision == None:
        revision = gcp_utils.get_revision_id()
    
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_revision = get_run_revision(revision=revision, service=service, project_id=project_id, region=region)
    
    try:
        max_instances = run_revision.scaling.max_instance_count
    except:
        # print("Error reading Revision Max Instances")
        max_instances = None

    return max_instances

def get_revision_min_instances(revision=None, service=None, project_id=None, region=None):
    # Get revision-level min-instances setting
    if revision == None:
        revision = gcp_utils.get_revision_id()
    
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_revision = get_run_revision(revision=revision, service=service, project_id=project_id, region=region)
        
    try:
        min_instances = run_revision.scaling.min_instance_count
    except:
        # print("Error reading Revision Min Instances")
        min_instances = 0

    return min_instances

def get_revision_max_concurrency(revision=None, service=None, project_id=None, region=None):
    # Get revision-level max-concurrency setting
    if revision == None:
        revision = gcp_utils.get_revision_id()
    
    if service == None:
        service = gcp_utils.get_service_id()

    if project_id == None:
        project_id = gcp_utils.get_project_id()
    
    if region == None:
        region = gcp_utils.get_gcp_region()

    run_revision = get_run_revision(revision=revision, service=service, project_id=project_id, region=region)
    
    try:
        max_concurrency = run_revision.max_instance_request_concurrency
    except:
        print("Error reading Revision Max Concurrency")
        max_concurrency = None

    return max_concurrency